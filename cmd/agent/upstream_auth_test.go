package main

import (
 "context"
 "crypto/sha256"
 "encoding/hex"
 "testing"

 auth "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
 "gateway-control/internal/model"
 "gateway-control/internal/secretbox"
)
func TestUpstreamCredentialsInLocalAuthorizer(t *testing.T){
 key:=make([]byte,32);for i:=range key {key[i]=byte(i+1)}
 password:="test-only-s3cr3t!"
 ciphertext,err:=secretbox.Encrypt(key,password);if err!=nil{t.Fatal(err)}
 sum:=sha256.Sum256([]byte("test-client-api-key"))
 state:=&nodeState{pkg:&model.Package{VirtualHosts:[]model.PkgVHost{{
  Domains:[]string{"api-gateway.mehmet.tech"},
  Routes:[]model.PkgRoute{{Path:"/api/probe/",MatchType:model.MatchPrefix,Methods:[]string{"POST"},AuthKeyHash:hex.EncodeToString(sum[:])}},
 }}}}
 checker:=&localAuthorizer{state:state,encryptionKey:key}
 attrs:=map[string]string{
  "gateway-required-header":"x-client-id", "gateway-upstream-mode":"basic",
  "gateway-upstream-username":"backend-user","gateway-upstream-cipher":ciphertext,
  "gateway-key-sha256":hex.EncodeToString(sum[:]),
 }
 makeReq:=func(headers map[string]string)*auth.CheckRequest{
  return &auth.CheckRequest{Attributes:&auth.AttributeContext{
   Request:&auth.AttributeContext_Request{Http:&auth.AttributeContext_HttpRequest{
     Host:"api-gateway.mehmet.tech",Path:"/api/probe/submit",Method:"POST",Headers:headers,
   }},
   ContextExtensions:attrs,
  }}
 }
 missing,_:=checker.Check(context.Background(),makeReq(map[string]string{"x-gateway-key":"test-client-api-key"}))
 if missing.GetDeniedResponse()==nil {t.Fatal("missing required header should be denied")}
 wrong,_:=checker.Check(context.Background(),makeReq(map[string]string{"x-client-id":"client-1","x-gateway-key":"bad"}))
 if wrong.GetDeniedResponse()==nil {t.Fatal("invalid key should be denied")}
 valid,_:=checker.Check(context.Background(),makeReq(map[string]string{"x-client-id":"client-1","x-gateway-key":"test-client-api-key"}))
 if valid.GetOkResponse()==nil {t.Fatal("valid request should be allowed")}
 headers:=valid.GetOkResponse().GetHeaders()
 if len(headers)!=1 || headers[0].GetHeader().GetKey()!="authorization" {t.Fatalf("no injected authorization header")}
 // No secret is returned from management APIs, but authz injects it into the private upstream request.
 if headers[0].GetHeader().GetValue()!="Basic YmFja2VuZC11c2VyOnRlc3Qtb25seS1zM2NyM3Qh" {t.Fatal("injected Basic Auth value mismatches")}
 attrs["gateway-upstream-mode"]="header";attrs["gateway-upstream-header"]="x-api-key"
 valid,_=checker.Check(context.Background(),makeReq(map[string]string{"x-client-id":"client-1","x-gateway-key":"test-client-api-key"}))
 if valid.GetOkResponse().GetHeaders()[0].GetHeader().GetKey()!="x-api-key" ||
    valid.GetOkResponse().GetHeaders()[0].GetHeader().GetValue()!=password {
  t.Fatal("custom upstream header mismatch")
 }
 attrs["gateway-upstream-mode"]="bearer"
 valid,_=checker.Check(context.Background(),makeReq(map[string]string{"x-client-id":"client-1","x-gateway-key":"test-client-api-key"}))
 if valid.GetOkResponse().GetHeaders()[0].GetHeader().GetValue()!="Bearer "+password {
  t.Fatal("bearer upstream header mismatch")
 }
}
