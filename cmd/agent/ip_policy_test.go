package main

import (
 "context"
 "testing"
 auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
 "gateway-control/internal/model"
)
func TestIPPolicyLocalAuthorizer(t *testing.T){
 state:=&nodeState{pkg:&model.Package{VirtualHosts:[]model.PkgVHost{{
   Domains:[]string{"api-gateway.mehmet.tech"},
   Routes:[]model.PkgRoute{{MatchType:model.MatchPrefix,Path:"/api/test/"}},
 }}}}
 a:=&localAuthorizer{state:state}
 attrs:=map[string]string{"gateway-ip-allowlist":"192.0.2.0/24","gateway-ip-denylist":"192.0.2.34/32"}
 probe:=func(ip string)(bool){
  req:=&auth.CheckRequest{Attributes:&auth.AttributeContext{
   Request:&auth.AttributeContext_Request{Http:&auth.AttributeContext_HttpRequest{
    Host:"api-gateway.mehmet.tech",Path:"/api/test/info",Method:"GET",
    Headers:map[string]string{"x-envoy-external-address":ip,"x-forwarded-for":"192.0.2.8"},
   }},
   ContextExtensions:attrs,
  }}
  result,err:=a.Check(context.Background(),req)
  if err!=nil{t.Fatal(err)}
  return result.GetOkResponse()!=nil
 }
 if !probe("192.0.2.9"){t.Fatal("allowed IP rejected")}
 if probe("192.0.2.34"){t.Fatal("explicit deny did not take precedence")}
 if probe("198.51.100.9"){t.Fatal("outside CIDR allowlist accepted")}
 if probe(""){t.Fatal("missing verified client IP accepted")}
 if state.authChecks.Load()!=4 || state.authDenied.Load()!=3 {t.Fatal("incorrect traffic auth counters")}
}
