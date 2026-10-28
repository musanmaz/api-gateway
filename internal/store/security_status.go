package store

import (
 "context"
 "encoding/json"
 "time"

 "gateway-control/internal/model"
)

// Security status is always derived from the revision each node actually ACKed.
// Never expose revision payloads: they include encrypted upstream credentials
// and key hashes that have no place in the browser.
type RouteSecurityPolicy struct {
 Path string `json:"path"`
 Methods []string `json:"methods"`
 MatchType string `json:"matchType"`
 RateLimitRPS int `json:"rateLimitRps"`
}
type ServiceSecurityPolicy struct {
 WAFMode string `json:"wafMode"`
 AuthMode string `json:"authMode"`
 UpstreamAuthMode string `json:"upstreamAuthMode"`
 UpstreamAuthConfigured bool `json:"upstreamAuthConfigured"`
 RequiredClientHeader string `json:"requiredClientHeader"`
 RateLimitRPS int `json:"rateLimitRps"`
 IPAllowlist []string `json:"ipAllowlist"`
 IPDenylist []string `json:"ipDenylist"`
 Routes []RouteSecurityPolicy `json:"routes"`
}
type NodeSecurityStatus struct {
 Name string `json:"name"`
 Location string `json:"location"`
 Heartbeat string `json:"heartbeat"`
 DesiredRevision *int64 `json:"desiredRevision"`
 AckedRevision *int64 `json:"ackedRevision"`
 State string `json:"state"`
 Applied *ServiceSecurityPolicy `json:"applied"`
}
type ServiceSecurityStatus struct {
 ServiceID string `json:"serviceId"`
 Saved ServiceSecurityPolicy `json:"saved"`
 Pending bool `json:"pending"`
 Status string `json:"status"`
 Nodes []NodeSecurityStatus `json:"nodes"`
}

func savedSecurity(s model.Service) ServiceSecurityPolicy {
 p:=ServiceSecurityPolicy{
  WAFMode:s.WafMode,AuthMode:s.AuthMode,UpstreamAuthMode:s.UpstreamAuthMode,
  UpstreamAuthConfigured:s.UpstreamAuthConfigured,RequiredClientHeader:s.RequiredClientHeader,
  RateLimitRPS:s.RateLimitRPS,
  IPAllowlist:append([]string{},s.IPAllowlist...),IPDenylist:append([]string{},s.IPDenylist...),
  Routes:[]RouteSecurityPolicy{},
 }
 if p.WAFMode==""{p.WAFMode="off"}
 if p.UpstreamAuthMode==""{p.UpstreamAuthMode="none"}
 if p.AuthMode==""{p.AuthMode="public"}
 for _,r:=range s.Routes{
  eff:=r.RateLimitRPS
  if eff==0{eff=s.RateLimitRPS}
  p.Routes=append(p.Routes,RouteSecurityPolicy{
   Path:r.Path,Methods:append([]string{},r.Methods...),MatchType:r.MatchType,RateLimitRPS:eff,
  })
 }
 return p
}
func appliedSecurity(payload string,serviceID string) (*ServiceSecurityPolicy,error){
 var pkg model.Package
 if err:=json.Unmarshal([]byte(payload),&pkg);err!=nil{return nil,err}
 for _,vhost:=range pkg.VirtualHosts{
  if vhost.Name!=serviceID || len(vhost.Routes)==0{continue}
  r:=vhost.Routes[0]
  p:=&ServiceSecurityPolicy{
   WAFMode:r.WafMode,AuthMode:"public",UpstreamAuthMode:r.UpstreamAuthMode,
   UpstreamAuthConfigured:r.UpstreamAuthMode!="" && r.UpstreamAuthMode!="none" && r.UpstreamAuthCipher!="",
   RequiredClientHeader:r.RequiredClientHeader,
   IPAllowlist:append([]string{},r.IPAllowlist...),IPDenylist:append([]string{},r.IPDenylist...),
   Routes:[]RouteSecurityPolicy{},
  }
  if p.WAFMode==""{p.WAFMode="off"}
  if p.UpstreamAuthMode==""{p.UpstreamAuthMode="none"}
  if r.AuthKeyHash!=""{p.AuthMode="api_key"}
  for _,route:=range vhost.Routes{
   p.Routes=append(p.Routes,RouteSecurityPolicy{
    Path:route.Path,Methods:append([]string{},route.Methods...),MatchType:route.MatchType,
    RateLimitRPS:route.RateLimitRPS,
   })
  }
  // Older revisions preserve only the effective limit per route; when all
  // routes share a limit, this is displayed as an inferred common limit.
  same:=true
  for _,route:=range p.Routes{if route.RateLimitRPS!=p.Routes[0].RateLimitRPS{same=false;break}}
  if same {p.RateLimitRPS=p.Routes[0].RateLimitRPS}
  return p,nil
 }
 return nil,nil
}

func (s *Store) GetServiceSecurityStatus(ctx context.Context,id string)(ServiceSecurityStatus,error){
 svc,err:=s.GetService(ctx,id)
 if err!=nil{return ServiceSecurityStatus{},err}
 out:=ServiceSecurityStatus{
  ServiceID:id,Saved:savedSecurity(svc),Pending:svc.Status=="active" && svc.Version!=svc.PublishedVersion,
  Status:svc.Status,Nodes:[]NodeSecurityStatus{},
 }
 rows,err:=s.pool.Query(ctx,`
 SELECT n.name,n.location,n.desired_revision,n.acked_revision,n.last_heartbeat_at,
        COALESCE(r.payload,'')
 FROM nodes n LEFT JOIN revisions r ON r.id=n.acked_revision
 WHERE n.fleet_id=$1 ORDER BY n.name`,svc.FleetID)
 if err!=nil{return ServiceSecurityStatus{},err}
 defer rows.Close()
 now:=time.Now().UTC()
 for rows.Next(){
  var item NodeSecurityStatus
  var last *time.Time
  var payload string
  if err:=rows.Scan(&item.Name,&item.Location,&item.DesiredRevision,&item.AckedRevision,&last,&payload);err!=nil{
   return ServiceSecurityStatus{},err
  }
  item.Heartbeat="unknown"
  if last!=nil{
   if now.Sub(last.UTC())>20*time.Second{item.Heartbeat="stale"}else{item.Heartbeat="fresh"}
  }
  item.State="not_deployed"
  if item.AckedRevision!=nil && payload!=""{
   policy,err:=appliedSecurity(payload,id)
   if err!=nil{return ServiceSecurityStatus{},err}
   item.Applied=policy
   if policy!=nil{
    item.State="outdated"
    if item.DesiredRevision!=nil && *item.DesiredRevision==*item.AckedRevision && item.Heartbeat=="fresh"{
     item.State="acknowledged"
    }
   }
  }
  if item.Heartbeat!="fresh" && item.State!="not_deployed"{item.State="offline"}
  out.Nodes=append(out.Nodes,item)
 }
 return out,rows.Err()
}
