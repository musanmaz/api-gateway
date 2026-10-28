package store

import (
 "context"
 "errors"
 "time"

 "github.com/jackc/pgx/v5"
 "gateway-control/internal/model"
 "gateway-control/internal/validate"
)

func (s *Store) SetRouteRateLimit(ctx context.Context,serviceID,routeID,actor string,rps int)(model.Service,error){
 if rps<0 || rps>10000 {return model.Service{},&validate.Error{Details:[]string{"endpoint rate limit 0-10000 istek/saniye olmalı"}}}
 tx,err:=s.pool.Begin(ctx);if err!=nil{return model.Service{},err};defer tx.Rollback(ctx)
 var previous int
 err=tx.QueryRow(ctx,"SELECT rate_limit_rps FROM routes WHERE id=$1 AND service_id=$2 FOR UPDATE",routeID,serviceID).Scan(&previous)
 if errors.Is(err,pgx.ErrNoRows){return model.Service{},ErrNotFound}
 if err!=nil{return model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE routes SET rate_limit_rps=$2 WHERE id=$1",routeID,rps);if err!=nil{return model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE services SET version=version+1, updated_at=$2 WHERE id=$1",serviceID,time.Now().UTC());if err!=nil{return model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"route.rate-limit.configure","route:"+routeID,
  map[string]int{"rps":previous},map[string]int{"rps":rps},"ok");err!=nil{return model.Service{},err}
 if err=tx.Commit(ctx);err!=nil{return model.Service{},err}
 return s.GetService(ctx,serviceID)
}
