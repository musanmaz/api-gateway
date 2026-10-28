package main

import (
 "context"
 "encoding/json"
 "io"
 "net"
 "net/http"
 "time"
 "gateway-control/internal/model"
)

func readEnvoyStats(socket string) model.NodeTelemetry{
 result:=model.NodeTelemetry{}
 client:=&http.Client{
  Timeout:800*time.Millisecond,
  Transport:&http.Transport{
   DialContext:func(ctx context.Context,_,_ string)(net.Conn,error){
    return (&net.Dialer{Timeout:500*time.Millisecond}).DialContext(ctx,"unix",socket)
   },
  },
 }
 resp,err:=client.Get("http://localhost/stats?format=json&usedonly")
 if err!=nil{return result}
 defer resp.Body.Close()
 if resp.StatusCode!=http.StatusOK{return result}
 var data struct{Stats []struct{Name string `json:"name"`;Value json.RawMessage `json:"value"`} `json:"stats"`}
 decoder:=json.NewDecoder(io.LimitReader(resp.Body,2<<20))
 if err=decoder.Decode(&data);err!=nil{return result}
 for _,s:=range data.Stats{
  var number uint64
  if json.Unmarshal(s.Value,&number)!=nil {continue}
  switch s.Name{
  case "http.ingress.downstream_rq_total": result.HTTPRequests=number
  case "http.ingress.downstream_rq_2xx": result.HTTP2xx=number
  case "http.ingress.downstream_rq_4xx": result.HTTP4xx=number
  case "http.ingress.downstream_rq_5xx": result.HTTP5xx=number
  }
 }
 return result
}
