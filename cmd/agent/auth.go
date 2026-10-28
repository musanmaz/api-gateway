package main

import (
 "context"
 "crypto/sha256"
 "crypto/subtle"
 "encoding/hex"
 "encoding/base64"
 "net"
 "net/netip"
 "net/url"
 "strings"

 auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
 typ "github.com/envoyproxy/go-control-plane/envoy/type/v3"
 core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
 rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
 "google.golang.org/grpc/codes"

 "gateway-control/internal/model"
 "gateway-control/internal/secretbox"
 "github.com/corazawaf/coraza/v3"
)

type localAuthorizer struct {
 auth.UnimplementedAuthorizationServer
 state *nodeState
 encryptionKey []byte
 waf coraza.WAF
 traffic *trafficCollector
}

func (s *localAuthorizer) Check(_ context.Context, req *auth.CheckRequest) (response *auth.CheckResponse, err error) {
 s.state.authChecks.Add(1)
 defer func(){if response!=nil && response.GetDeniedResponse()!=nil{s.state.authDenied.Add(1)}}()
 attributes:=req.GetAttributes()
 if attributes==nil || attributes.GetRequest()==nil || attributes.GetRequest().GetHttp()==nil {
  return authDenied(),nil
 }
 httpReq:=attributes.GetRequest().GetHttp()
 host:=strings.ToLower(strings.TrimSpace(httpReq.GetHost()))
 if host=="" {host=httpReq.GetHeaders()[":authority"]}
 if h,_,err:=net.SplitHostPort(host);err==nil {host=h}
 host=strings.ToLower(strings.TrimSuffix(host,"."))
 path:=httpReq.GetPath()
 if u,err:=url.ParseRequestURI(path);err==nil {path=u.Path}
 method:=strings.ToUpper(httpReq.GetMethod())
 headers:=httpReq.GetHeaders()
 key:=headers["x-gateway-key"]

 s.state.mu.Lock()
 pkg:=s.state.pkg
 if pkg==nil {s.state.mu.Unlock();return authDenied(),nil}
 var matched bool
 var expected string
 for _,vhost:=range pkg.VirtualHosts {
  found:=false
  for _,domain:=range vhost.Domains {if domain==host {found=true;break}}
  if !found {continue}
  for _,rt:=range vhost.Routes {
   if rt.MatchType==model.MatchExact {
    if path!=rt.Path {continue}
   } else if !strings.HasPrefix(path,rt.Path) {continue}
   if len(rt.Methods)>0 {
    accepted:=false
    for _,m:=range rt.Methods {if m==method {accepted=true;break}}
    if !accepted {continue}
   }
   expected=rt.AuthKeyHash
   matched=true
   break
  }
  if matched {break}
 }
 s.state.mu.Unlock()
 // Envoy supplies trusted per-route metadata, not client-controlled headers.
 // This is the authoritative policy for protected routes on every hostname.
 if routed:=attributes.GetContextExtensions()["gateway-key-sha256"];routed!="" {
  expected=routed
  matched=true
 }
 if expected!="" {
  if len(key)<16 || len(key)>256 {return authDenied(),nil}
  digest:=sha256.Sum256([]byte(key))
  want,err:=hex.DecodeString(expected)
  if err!=nil || len(want)!=len(digest) || subtle.ConstantTimeCompare(digest[:],want)!=1 {return authDenied(),nil}
 }
 context:=attributes.GetContextExtensions()
 if context["gateway-ip-allowlist"]!="" || context["gateway-ip-denylist"]!="" {
  // Header is computed/overwritten by the *trusted* Envoy HTTP connection manager.
  // Never parse user-provided X-Forwarded-For directly in authorization code.
  iptext:=readHeader(headers,"x-envoy-external-address")
  ip,err:=netip.ParseAddr(strings.TrimSpace(iptext))
  if err!=nil {return ipForbidden(),nil}
  denied:=cidrMatch(ip,context["gateway-ip-denylist"])
  allowed:=context["gateway-ip-allowlist"]=="" || cidrMatch(ip,context["gateway-ip-allowlist"])
  if denied || !allowed {return ipForbidden(),nil}
 }
 if mode:=context["gateway-waf-mode"];mode=="detect" || mode=="block" {
  if s.waf==nil {return wafUnavailable(),nil}
  result,err:=inspectWAF(s.waf,httpReq)
  if err!=nil {return wafUnavailable(),nil}
  if result.Matched{
   s.state.wafMatches.Add(1)
   if s.traffic!=nil{s.traffic.observeWAF(context["gateway-service-id"],mode=="block")}
   if mode=="block" {s.state.wafBlocked.Add(1);return wafForbidden(),nil}
  }
 }
 required:=context["gateway-required-header"]
 if required!="" && strings.TrimSpace(readHeader(headers,required))=="" {return authDenied(),nil}
 mode:=context["gateway-upstream-mode"]
 if mode=="" || mode=="none" {return authAllowed(nil),nil}
 encrypted:=context["gateway-upstream-cipher"]
 if encrypted=="" {return authDenied(),nil}
 secret,err:=secretbox.Decrypt(s.encryptionKey,encrypted)
 if err!=nil || secret=="" {return authDenied(),nil}
 var target, value string
 switch mode {
 case "basic":
  target="authorization"
  value="Basic "+base64.StdEncoding.EncodeToString([]byte(context["gateway-upstream-username"]+":"+secret))
 case "bearer":
  target="authorization"
  value="Bearer "+secret
 case "header":
  target=context["gateway-upstream-header"]
  value=secret
 default:
  return authDenied(),nil
 }
 if target=="" || strings.ContainsAny(value,"\r\n") {return authDenied(),nil}
 return authAllowed(&core.HeaderValue{Key:target,Value:value}),nil
}

func readHeader(headers map[string]string,name string) string {
 for k,v:=range headers {if strings.EqualFold(k,name){return v}}
 return ""
}
func authAllowed(injected *core.HeaderValue) *auth.CheckResponse {
 return &auth.CheckResponse{
  Status:&rpcstatus.Status{Code:int32(codes.OK)},
  HttpResponse:&auth.CheckResponse_OkResponse{
   OkResponse:func() *auth.OkHttpResponse {
    response:=&auth.OkHttpResponse{HeadersToRemove:[]string{"x-gateway-key"}}
    if injected!=nil {
     response.Headers=[]*core.HeaderValueOption{{
      Header:injected,
      AppendAction:core.HeaderValueOption_OVERWRITE_IF_EXISTS_OR_ADD,
     }}
    }
    return response
   }(),
  },
 }
}
func authDenied() *auth.CheckResponse {
 return &auth.CheckResponse{
  Status:&rpcstatus.Status{Code:int32(codes.PermissionDenied)},
  HttpResponse:&auth.CheckResponse_DeniedResponse{
   DeniedResponse:&auth.DeniedHttpResponse{
    Status:&typ.HttpStatus{Code:typ.StatusCode_Unauthorized},
    Body:"Gateway API key gerekli veya gecersiz",
   },
  },
 }
}


func cidrMatch(ip netip.Addr,cidrs string)bool {
 for _,raw:=range strings.Split(cidrs,",") {
  prefix,err:=netip.ParsePrefix(raw)
  if err==nil && prefix.Contains(ip) {return true}
 }
 return false
}
func ipForbidden()*auth.CheckResponse{
 return &auth.CheckResponse{
  Status:&rpcstatus.Status{Code:int32(codes.PermissionDenied)},
  HttpResponse:&auth.CheckResponse_DeniedResponse{
   DeniedResponse:&auth.DeniedHttpResponse{
    Status:&typ.HttpStatus{Code:typ.StatusCode_Forbidden},
    Body:"Gateway IP access policy denied",
   },
  },
 }
}
