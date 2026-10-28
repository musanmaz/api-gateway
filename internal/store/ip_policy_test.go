package store

import "testing"

func TestNormalizeCIDRs(t *testing.T) {
 result,err:=normalizeCIDRs([]string{"192.0.2.17/24","192.0.2.0/24","2001:db8::1"})
 if err!=nil{t.Fatal(err)}
 if len(result)!=2 || result[0]!="192.0.2.0/24" || result[1]!="2001:db8::1/128"{t.Fatalf("bad normalization %v",result)}
 _,err=normalizeCIDRs([]string{"garbage"})
 if err==nil{t.Fatal("invalid CIDR accepted")}
}
