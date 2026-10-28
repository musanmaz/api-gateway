package store

import (
 "testing"
 "gateway-control/internal/compiler"
 "gateway-control/internal/model"
 routepb "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
 "github.com/envoyproxy/go-control-plane/pkg/resource/v3"
)

func TestCentralGatewayRendersSeparateRoutesWithoutChangingLegacyHosts(t *testing.T) {
 listener:=model.Listener{ID:"http-1",Name:"public-http",Protocol:"http",Port:10000,FleetID:"fleet-1"}
 mk:=func(name,host,backend,path string) model.Service {
  return model.Service{
   ID:name+"-id", Name:name, ListenerID:listener.ID, Hostnames:[]string{host},
   Routes:[]model.Route{{ID:name+"-route", MatchType:model.MatchPrefix, Path:path,
    Upstream:model.Upstream{ID:name+"-pool",Name:name,Algorithm:model.AlgoRoundRobin,
      ConnectTimeoutMs:1000, RequestTimeoutMs:5000,
      Endpoints:[]model.Endpoint{{Address:backend,Port:8080,Weight:1}},
    },
   }},
  }
 }
 services:=[]model.Service{
  mk("api1","api1.mehmet.tech","backend-a","/"),
  mk("api11","api11.mehmet.tech","backend-b","/api/info"),
 }
 pkg,_,err:=buildPackage(7,"fleet-1",services,[]model.Listener{listener})
 if err!=nil {t.Fatal(err)}
 var central *model.PkgVHost
 legacy:=0
 for i:=range pkg.VirtualHosts {
  vh:= &pkg.VirtualHosts[i]
  if len(vh.Domains)!=1 {t.Fatalf("domains: %v",vh.Domains)}
  if vh.Domains[0]=="api-gateway.mehmet.tech" {central=vh} else {legacy++}
 }
 if central==nil || legacy!=2 {t.Fatalf("gateway and legacy hosts expected, got %#v",pkg.VirtualHosts)}
 desired:=map[string]string{
  "/api/api1/":"/", "/api/api1":"/", "/api/api11/api/info":"/api/info",
 }
 for _,r:=range central.Routes {
  want,ok:=desired[r.Path]
  if !ok {t.Fatalf("unexpected central path: %s",r.Path)}
  if want!=r.RewritePrefix {t.Fatalf("rewrite for %s => %s expected %s",r.Path,r.RewritePrefix,want)}
  delete(desired,r.Path)
 }
 if len(desired)!=0 {t.Fatalf("missing routes: %v",desired)}
 snapshot,err:=compiler.Build(pkg,nil,nil)
 if err!=nil {t.Fatal(err)}
 rc:=snapshot.GetResources(resource.RouteType)["edge"].(*routepb.RouteConfiguration)
 count:=0
 for _,vh:=range rc.VirtualHosts {
  if len(vh.Domains)==1 && vh.Domains[0]=="api-gateway.mehmet.tech" {
   count++
   for _,rt:=range vh.Routes {
    if rt.GetRoute().GetPrefixRewrite()=="" {t.Fatalf("missing prefix rewrite: %v",rt)}
   }
  }
 }
 if count!=1 {t.Fatalf("central virtual hosts=%d",count)}
}
