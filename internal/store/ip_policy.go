package store

import (
 "context"
 "errors"
 "net/netip"
 "sort"
 "time"
 "github.com/jackc/pgx/v5"
 "gateway-control/internal/model"
 "gateway-control/internal/validate"
)
func normalizeCIDRs(items []string)([]string,error){
 if len(items)>100 {return nil,&validate.Error{Details:[]string{"En fazla 100 CIDR girilebilir"}}}
 out:=make([]string,0,len(items));unique:=map[string]bool{}
 for _,raw:=range items {
  prefix,err:=netip.ParsePrefix(raw)
  if err!=nil {
   ip,iperr:=netip.ParseAddr(raw)
   if iperr!=nil{return nil,&validate.Error{Details:[]string{"Geçersiz IP/CIDR: "+raw}}}
   prefix=netip.PrefixFrom(ip,ip.BitLen())
  }
  k:=prefix.Masked().String()
  if !unique[k]{unique[k]=true;out=append(out,k)}
 }
 sort.Strings(out);return out,nil
}
func (s *Store) SetServiceIPPolicy(ctx context.Context,id,actor string,allow,deny []string)(model.Service,error){
 a,err:=normalizeCIDRs(allow);if err!=nil{return model.Service{},err}
 d,err:=normalizeCIDRs(deny);if err!=nil{return model.Service{},err}
 tx,err:=s.pool.Begin(ctx);if err!=nil{return model.Service{},err};defer tx.Rollback(ctx)
 var oldAllow,oldDeny []string
 err=tx.QueryRow(ctx,"SELECT ip_allowlist,ip_denylist FROM services WHERE id=$1 FOR UPDATE",id).Scan(&oldAllow,&oldDeny)
 if errors.Is(err,pgx.ErrNoRows){return model.Service{},ErrNotFound}
 if err!=nil{return model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE services SET ip_allowlist=$2,ip_denylist=$3,version=version+1,updated_at=$4 WHERE id=$1",id,a,d,time.Now().UTC())
 if err!=nil{return model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"service.ip-policy.configure","service:"+id,
  map[string]any{"allow":oldAllow,"deny":oldDeny},map[string]any{"allow":a,"deny":d},"ok");err!=nil{return model.Service{},err}
 if err=tx.Commit(ctx);err!=nil{return model.Service{},err}
 return s.GetService(ctx,id)
}
