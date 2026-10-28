package store

import (
 "context"
 "errors"
 "time"

 "github.com/jackc/pgx/v5"
 "gateway-control/internal/model"
 "gateway-control/internal/validate"
)
func (s *Store) SetWAF(ctx context.Context,id,actor,mode string)(model.Service,error){
 if mode!="off" && mode!="detect" && mode!="block" {
  return model.Service{},&validate.Error{Details:[]string{"WAF mode off, detect veya block olmalı"}}
 }
 tx,err:=s.pool.Begin(ctx);if err!=nil{return model.Service{},err}
 defer tx.Rollback(ctx)
 var previous string
 err=tx.QueryRow(ctx,"SELECT waf_mode FROM services WHERE id=$1 FOR UPDATE",id).Scan(&previous)
 if errors.Is(err,pgx.ErrNoRows){return model.Service{},ErrNotFound}
 if err!=nil{return model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE services SET waf_mode=$2,version=version+1,updated_at=$3 WHERE id=$1",id,mode,time.Now().UTC())
 if err!=nil{return model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"service.waf.configure","service:"+id,
  map[string]string{"mode":previous},map[string]string{"mode":mode},"ok");err!=nil{return model.Service{},err}
 if err=tx.Commit(ctx);err!=nil{return model.Service{},err}
 return s.GetService(ctx,id)
}
