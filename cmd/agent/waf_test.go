package main

import (
 "testing"
 auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
)
func TestCorazaWAFSQLInjection(t *testing.T){
 waf,err:=createCorazaWAF();if err!=nil{t.Fatalf("load CRS failed: %v",err)}
 clean:=&auth.AttributeContext_HttpRequest{
  Host:"api-gateway.mehmet.tech",Path:"/api/test/orders?id=42",Method:"GET",
  Headers:map[string]string{"content-type":"application/json","host":"api-gateway.mehmet.tech"},
 }
 result,err:=inspectWAF(waf,clean)
 if err!=nil || result.Matched {t.Fatalf("normal request flagged: %+v %v",result,err)}
 attack:=&auth.AttributeContext_HttpRequest{
  Host:"api-gateway.mehmet.tech",Path:"/api/test/orders?id=1%20UNION%20SELECT%20password%20FROM%20users",Method:"GET",
  Headers:map[string]string{"content-type":"application/json","host":"api-gateway.mehmet.tech"},
 }
 result,err=inspectWAF(waf,attack)
 if err!=nil || !result.Matched {t.Fatalf("injection request not detected: %+v %v",result,err)}
}
