package main

import (
  "crypto/rand"
 "encoding/hex"
 "errors"
 "io"
 "math"
 "strings"
 "sync"
 "time"

 logsvc "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v3"
 logs "github.com/envoyproxy/go-control-plane/envoy/data/accesslog/v3"
 "google.golang.org/grpc"

 "gateway-control/internal/model"
)

type trafficCollector struct{
 logsvc.UnimplementedAccessLogServiceServer
 mu sync.Mutex
 bootID string
 buckets map[string]map[int64]*model.ServiceTrafficBucket
 state *nodeState
}

func newTrafficCollector(state *nodeState)*trafficCollector{
 var seed [16]byte
 if _,err:=rand.Read(seed[:]);err!=nil{panic(err)}
 return &trafficCollector{bootID:hex.EncodeToString(seed[:]),buckets:map[string]map[int64]*model.ServiceTrafficBucket{},state:state}
}
func (c *trafficCollector) put(serviceID string,ts time.Time,write func(*model.ServiceTrafficBucket)){
 if serviceID==""{return}
 minute:=ts.UTC().Unix()/60
 c.mu.Lock();defer c.mu.Unlock()
 rows:=c.buckets[serviceID]
 if rows==nil{rows=map[int64]*model.ServiceTrafficBucket{};c.buckets[serviceID]=rows}
 row:=rows[minute]
 if row==nil{row=&model.ServiceTrafficBucket{ServiceID:serviceID,Minute:minute};rows[minute]=row}
 write(row)
}
func (c *trafficCollector) snapshot()[]model.ServiceTrafficBucket{
 now:=time.Now().UTC().Unix()/60
 c.mu.Lock();defer c.mu.Unlock()
 results:=make([]model.ServiceTrafficBucket,0,128)
 for id,mins:=range c.buckets{
  for minute,b:=range mins{
   if minute<now-4||minute>now+1{delete(mins,minute);continue}
   results=append(results,*b)
  }
  if len(mins)==0{delete(c.buckets,id)}
 }
 return results
}
func durationBin(ms int64)int{
 thresholds:=[9]int64{10,25,50,100,250,500,1000,2500,10000}
 for i,edge:=range thresholds{if ms<=edge{return i}}
 return 9
}
func (c *trafficCollector) serviceForRoute(name string)string{
 c.state.mu.Lock();defer c.state.mu.Unlock()
 if c.state.pkg==nil{return ""}
 routeName:=strings.TrimPrefix(name,"central-")
 for _,host:=range c.state.pkg.VirtualHosts{
  for _,route:=range host.Routes{
   if route.ServiceID==""{continue}
   if routeName==route.Name||strings.HasPrefix(routeName,route.Name+"-"){return route.ServiceID}
  }
 }
 return ""
}
func (c *trafficCollector) observe(entry *logs.HTTPAccessLogEntry){
 common:=entry.GetCommonProperties()
 if common==nil{return}
 sid:=c.serviceForRoute(common.GetRouteName())
 if sid==""{return}
 code:=int(entry.GetResponse().GetResponseCode().GetValue())
 duration:=int64(0)
 if common.GetDuration()!=nil{duration=common.GetDuration().AsDuration().Milliseconds()}
 if duration<0{duration=0}
 stamp:=time.Now()
 if common.GetStartTime()!=nil{stamp=common.GetStartTime().AsTime()}
 if math.Abs(time.Since(stamp).Minutes())>6{stamp=time.Now()}
 detail:=entry.GetResponse().GetResponseCodeDetails()
 c.put(sid,stamp,func(m *model.ServiceTrafficBucket){
  m.Requests++
  switch {
  case code>=200&&code<300:m.Status2xx++
  case code>=300&&code<400:m.Status3xx++
  case code>=400&&code<500:m.Status4xx++
  case code>=500&&code<600:m.Status5xx++
  }
  if code==429 && (strings.Contains(detail,"rate_limit")||strings.Contains(detail,"local_rate")){m.Limited++}
  m.Durations[durationBin(duration)]++
 })
}
func (c *trafficCollector) observeWAF(service string,blocked bool){
 c.put(service,time.Now(),func(m *model.ServiceTrafficBucket){
  m.WAFMatches++
  if blocked{m.WAFBlocked++}
 })
}
func (c *trafficCollector) StreamAccessLogs(stream grpc.ClientStreamingServer[logsvc.StreamAccessLogsMessage,logsvc.StreamAccessLogsResponse])error{
 for {
  msg,err:=stream.Recv()
  if errors.Is(err,io.EOF){return stream.SendAndClose(&logsvc.StreamAccessLogsResponse{})}
  if err!=nil{return err}
  if entries:=msg.GetHttpLogs();entries!=nil{
   for _,e:=range entries.GetLogEntry(){c.observe(e)}
  }
 }
}
