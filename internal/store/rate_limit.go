package store

import (
  "context"
  "errors"
  "time"
  "github.com/jackc/pgx/v5"
  "gateway-control/internal/model"
  "gateway-control/internal/validate"
)

func (s *Store) SetRateLimit(ctx context.Context,id,actor string,rps int)(model.Service,error){
 if rps<0 || rps>10000 {return model.Service{},&validate.Error{Details:[]string{"rate limit 0-10000 istek/saniye olmalıdır"}}}
 tx,err:=s.pool.Begin(ctx);if err!=nil{return model.Service{},err};defer tx.Rollback(ctx)
 var previous int
 err=tx.QueryRow(ctx,"SELECT rate_limit_rps FROM services WHERE id=$1 FOR UPDATE",id).Scan(&previous)
 if errors.Is(err,pgx.ErrNoRows){return model.Service{},ErrNotFound}
 if err!=nil{return model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE services SET rate_limit_rps=$2,version=version+1,updated_at=$3 WHERE id=$1",id,rps,time.Now().UTC())
 if err!=nil{return model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"service.rate-limit.configure","service:"+id,
  map[string]int{"rps":previous},map[string]int{"rps":rps},"ok");err!=nil{return model.Service{},err}
 if err=tx.Commit(ctx);err!=nil{return model.Service{},err}
 return s.GetService(ctx,id)
}
