package main

import (
 "testing"
 "time"

 log "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v3"
 "google.golang.org/protobuf/types/known/durationpb"
 "google.golang.org/protobuf/types/known/wrapperspb"
 "google.golang.org/protobuf/types/known/timestamppb"
 "gateway-control/internal/model"
)
func TestServiceTrafficAttribution(t *testing.T){
 id:="12345678-1234-1234-1234-123456789abc"
 state:=&nodeState{pkg:&model.Package{VirtualHosts:[]model.PkgVHost{
  {Name:"gateway-central-http",Domains:[]string{"api-gateway.mehmet.tech"},
   Routes:[]model.PkgRoute{{ServiceID:id,Name:"central-route-one",Path:"/api/first"}}},
  {Name:id,Domains:[]string{"api1.mehmet.tech"},
   Routes:[]model.PkgRoute{{ServiceID:id,Name:"route-one",Path:"/"}}},
 }}}
 collector:=newTrafficCollector(state)
 for _,route:=range []string{"route-one","central-route-one"}{
  entry:=&log.HTTPAccessLogEntry{
   CommonProperties:&log.AccessLogCommon{RouteName:route,Duration:durationpb.New(12*time.Millisecond),StartTime:timestamppb.Now()},
   Response:&log.HTTPResponseProperties{ResponseCode:wrapperspb.UInt32(200)},
  }
  collector.observe(entry)
 }
 collector.observeWAF(id,true)
 rows:=collector.snapshot()
 if len(rows)!=1{t.Fatalf("expected one minute bucket, got %d",len(rows))}
 b:=rows[0]
 if b.Requests!=2 || b.Status2xx!=2 || b.WAFMatches!=1 || b.WAFBlocked!=1 || b.Durations[1]!=2 {
  t.Fatalf("wrong attribution %+v",b)
 }
 collector.observe(&log.HTTPAccessLogEntry{
  CommonProperties:&log.AccessLogCommon{RouteName:"unknown",Duration:durationpb.New(time.Millisecond),StartTime:timestamppb.Now()},
  Response:&log.HTTPResponseProperties{ResponseCode:wrapperspb.UInt32(500)},
 })
 if len(collector.snapshot())!=1 || collector.snapshot()[0].Requests!=2{t.Fatal("unknown route was counted")}
}
