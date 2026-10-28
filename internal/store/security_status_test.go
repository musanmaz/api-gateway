package store

import (
 "encoding/json"
 "testing"

 "gateway-control/internal/model"
)

func TestAppliedSecuritySanitizesSecrets(t *testing.T){
 pkg:=model.Package{VirtualHosts:[]model.PkgVHost{{
  Name:"service-one",
  Routes:[]model.PkgRoute{{Path:"/orders",MatchType:"exact",Methods:[]string{"POST"},
   RateLimitRPS:12,WafMode:"block",AuthKeyHash:"a-secret-hash",
   UpstreamAuthMode:"basic",UpstreamAuthCipher:"encrypted-stuff",
   IPAllowlist:[]string{"192.0.2.0/24"},IPDenylist:[]string{"203.0.113.0/24"},
  }},
 }}}
 data,err:=json.Marshal(pkg)
 if err!=nil{t.Fatal(err)}
 policy,err:=appliedSecurity(string(data),"service-one")
 if err!=nil{t.Fatal(err)}
 if policy==nil || policy.WAFMode!="block" || policy.AuthMode!="api_key" ||
    policy.UpstreamAuthMode!="basic" || !policy.UpstreamAuthConfigured || policy.Routes[0].RateLimitRPS!=12{
  t.Fatalf("bad policy: %+v",policy)
 }
 raw,err:=json.Marshal(policy)
 if err!=nil{t.Fatal(err)}
 for _,name:=range []string{"a-secret-hash","encrypted-stuff","authKeyHash","upstreamAuthCipher"}{
  if stringContains(string(raw),name){t.Fatalf("leaked %s",name)}
 }
}
func stringContains(haystack,needle string) bool{
 for i:=0;i+len(needle)<=len(haystack);i++{if haystack[i:i+len(needle)]==needle{return true}}
 return false
}
