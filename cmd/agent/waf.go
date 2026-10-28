package main

import (
 "fmt"
 "net/netip"
 "net/url"
 "strings"

 "github.com/corazawaf/coraza/v3"
 crs "github.com/corazawaf/coraza-coreruleset/crs/v4"
 conf "github.com/corazawaf/coraza-coreruleset/coraza/v3"
 "github.com/jcchavezs/mergefs"

 auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
 typ "github.com/envoyproxy/go-control-plane/envoy/type/v3"
 rpcstatus "google.golang.org/genproto/googleapis/rpc/status"
 "google.golang.org/grpc/codes"
)

type wafDecision struct {
 Matched bool
 Status int
}

func createCorazaWAF()(coraza.WAF,error){
 return coraza.NewWAF(coraza.NewWAFConfig().WithRootFS(mergefs.Merge(crs.FS,conf.FS)).
  WithDirectives(`
    Include @coraza.conf-recommended
    SecAuditEngine Off
    SecDebugLogLevel 0
    SecRuleEngine On
    SecRequestBodyAccess On
    SecRequestBodyLimit 1048576
    SecRequestBodyNoFilesLimit 1048576
    Include @crs-setup.conf.example
    Include @owasp_crs/*.conf
  `))
}

func inspectWAF(waf coraza.WAF,httpReq *auth.AttributeContext_HttpRequest) (wafDecision,error){
 tx:=waf.NewTransaction()
 defer func(){tx.ProcessLogging();tx.Close()}()
 client:="127.0.0.1"
 if ip,err:=netip.ParseAddr(strings.TrimSpace(readHeader(httpReq.GetHeaders(),"x-envoy-external-address")));err==nil{client=ip.String()}
 tx.ProcessConnection(client,0,"127.0.0.1",80)
 path:=httpReq.GetPath()
 tx.ProcessURI(path,httpReq.GetMethod(),"HTTP/1.1")
 if u,err:=url.ParseRequestURI(path);err==nil {
  for k,values:=range u.Query(){for _,value:=range values {tx.AddGetRequestArgument(k,value)}}
 }
 tx.SetServerName(httpReq.GetHost())
 tx.AddRequestHeader("Host",httpReq.GetHost())
 for name,value:=range httpReq.GetHeaders(){
  if strings.EqualFold(name,"host") || strings.HasPrefix(name,":"){continue}
  tx.AddRequestHeader(name,value)
 }
 result:=wafDecision{}
 if it:=tx.ProcessRequestHeaders();it!=nil {
  return wafDecision{Matched:true,Status:it.Status},nil
 }
 data:=httpReq.GetRawBody()
 if len(data)==0 {data=[]byte(httpReq.GetBody())}
 if len(data)>1048576{return wafDecision{Matched:true,Status:413},nil}
 if len(data)>0 {
  it,_,err:=tx.WriteRequestBody(data)
  if err!=nil{return result,fmt.Errorf("waf body parse: %w",err)}
  if it!=nil{return wafDecision{Matched:true,Status:it.Status},nil}
 }
 it,err:=tx.ProcessRequestBody()
 if err!=nil{return result,fmt.Errorf("waf body phase: %w",err)}
 if it!=nil{return wafDecision{Matched:true,Status:it.Status},nil}
 return result,nil
}
func wafForbidden() *auth.CheckResponse{
 return &auth.CheckResponse{
  Status:&rpcstatus.Status{Code:int32(codes.PermissionDenied)},
  HttpResponse:&auth.CheckResponse_DeniedResponse{DeniedResponse:&auth.DeniedHttpResponse{
   Status:&typ.HttpStatus{Code:typ.StatusCode_Forbidden},
   Body:"Request blocked by WAF policy",
  }},
 }
}
func wafUnavailable() *auth.CheckResponse{
 return &auth.CheckResponse{
  Status:&rpcstatus.Status{Code:int32(codes.Unavailable)},
  HttpResponse:&auth.CheckResponse_DeniedResponse{DeniedResponse:&auth.DeniedHttpResponse{
   Status:&typ.HttpStatus{Code:typ.StatusCode_ServiceUnavailable},
   Body:"WAF verification temporarily unavailable",
  }},
 }
}
