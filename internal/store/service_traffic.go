package store

import (
 "context"
 "encoding/hex"
 "encoding/json"
 "fmt"
 "sort"
 "time"

 "github.com/jackc/pgx/v5"

 "gateway-control/internal/model"
 "gateway-control/internal/validate"
)
type TrafficPoint struct {
 Minute int64 `json:"minute"`
 Requests uint64 `json:"requests"`
 Status2xx uint64 `json:"status2xx"`
 Status3xx uint64 `json:"status3xx"`
 Status4xx uint64 `json:"status4xx"`
 Status5xx uint64 `json:"status5xx"`
 Limited uint64 `json:"limited"`
 WAFMatches uint64 `json:"wafMatches"`
 WAFBlocked uint64 `json:"wafBlocked"`
 Durations [10]uint64 `json:"-"`
 P95Ms int64 `json:"p95Ms"`
 P99Ms int64 `json:"p99Ms"`
}
type ServiceTrafficReport struct{
 ServiceID string `json:"serviceId"`
 Minutes int `json:"minutes"`
 From time.Time `json:"from"`
 To time.Time `json:"to"`
 HasData bool `json:"hasData"`
 Summary TrafficPoint `json:"summary"`
 Series []TrafficPoint `json:"series"`
}
func (s *Store) saveTraffic(ctx context.Context,in HeartbeatInput)error{
 if len(in.Traffic)==0{return nil}
 if len(in.Traffic)>1200{return &validate.Error{Details:[]string{"too many service traffic buckets"}}}
 if len(in.TrafficBootID)!=32{return &validate.Error{Details:[]string{"invalid traffic boot id"}}}
 if _,err:=hex.DecodeString(in.TrafficBootID);err!=nil{return &validate.Error{Details:[]string{"invalid traffic boot id"}}}
 now:=time.Now().UTC()
 tx,err:=s.pool.Begin(ctx);if err!=nil{return err}
 defer tx.Rollback(ctx)
 for _,bucket:=range in.Traffic{
  if bucket.ServiceID=="" || bucket.Minute<now.Add(-6*time.Minute).Unix()/60 || bucket.Minute>now.Add(time.Minute).Unix()/60{
   return &validate.Error{Details:[]string{"invalid service traffic bucket timestamp or service"}}
  }
  if bucket.Requests<bucket.Status2xx+bucket.Status3xx+bucket.Status4xx+bucket.Status5xx ||
    bucket.Limited>bucket.Requests || bucket.WAFBlocked>bucket.WAFMatches {
   return &validate.Error{Details:[]string{"inconsistent service traffic counters"}}
  }
  raw,err:=json.Marshal(bucket);if err!=nil{return err}
  _,err=tx.Exec(ctx,`
   INSERT INTO service_traffic_minutes (node_name,boot_id,service_id,bucket_start,metrics)
   SELECT $1,$2,s.id,to_timestamp($4),$5::jsonb
   FROM services s WHERE s.id=$3::uuid
   ON CONFLICT (node_name,boot_id,service_id,bucket_start)
   DO UPDATE SET metrics=EXCLUDED.metrics,received_at=now()`,
   in.Node,in.TrafficBootID,bucket.ServiceID,bucket.Minute*60,raw)
  if err!=nil{return err}
 }
 return tx.Commit(ctx)
}
func addTraffic(out *TrafficPoint,in model.ServiceTrafficBucket){
 out.Requests+=in.Requests
 out.Status2xx+=in.Status2xx
 out.Status3xx+=in.Status3xx
 out.Status4xx+=in.Status4xx
 out.Status5xx+=in.Status5xx
 out.Limited+=in.Limited
 out.WAFMatches+=in.WAFMatches
 out.WAFBlocked+=in.WAFBlocked
 for i,n:=range in.Durations{out.Durations[i]+=n}
}
func percentile(hist [10]uint64,pct uint64)int64{
 total:=uint64(0)
 for _,n:=range hist{total+=n}
 if total==0{return 0}
 target:=(total*pct+99)/100
 if target==0{target=1}
 cuts:=[10]int64{10,25,50,100,250,500,1000,2500,10000,10000}
 seen:=uint64(0)
 for i,n:=range hist{
  seen+=n
  if seen>=target{return cuts[i]}
 }
 return cuts[9]
}
func (s *Store) ServiceTraffic(ctx context.Context,id string,minutes int)(ServiceTrafficReport,error){
 if minutes!=60 && minutes!=1440{minutes=60}
 if _,err:=s.GetService(ctx,id);err!=nil{return ServiceTrafficReport{},err}
 now:=time.Now().UTC()
 start:=now.Truncate(time.Minute).Add(-time.Duration(minutes-1)*time.Minute)
 rows,err:=s.pool.Query(ctx,`
 SELECT metrics FROM service_traffic_minutes
 WHERE service_id=$1::uuid AND bucket_start >=$2 AND bucket_start <=$3
 ORDER BY bucket_start`,id,start,now)
 if err!=nil{return ServiceTrafficReport{},err}
 defer rows.Close()
 byMinute:=map[int64]*TrafficPoint{}
 out:=ServiceTrafficReport{ServiceID:id,Minutes:minutes,From:start,To:now,Series:[]TrafficPoint{}}
 for rows.Next(){
  var raw []byte
  if err:=rows.Scan(&raw);err!=nil{return ServiceTrafficReport{},err}
  var m model.ServiceTrafficBucket
  if err:=json.Unmarshal(raw,&m);err!=nil{return ServiceTrafficReport{},err}
  b:=byMinute[m.Minute]
  if b==nil{b=&TrafficPoint{Minute:m.Minute};byMinute[m.Minute]=b}
  addTraffic(b,m)
  addTraffic(&out.Summary,m)
  out.HasData=true
 }
 if err:=rows.Err();err!=nil{return ServiceTrafficReport{},err}
 keys:=make([]int64,0,len(byMinute))
 for k:=range byMinute{keys=append(keys,k)}
 sort.Slice(keys,func(i,j int)bool{return keys[i]<keys[j]})
 for _,minute:=range keys{
  point:=byMinute[minute]
  point.P95Ms=percentile(point.Durations,95)
  point.P99Ms=percentile(point.Durations,99)
  out.Series=append(out.Series,*point)
 }
 out.Summary.P95Ms=percentile(out.Summary.Durations,95)
 out.Summary.P99Ms=percentile(out.Summary.Durations,99)
 return out,nil
}
var _ = fmt.Sprintf
var _ pgx.Tx
