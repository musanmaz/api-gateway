package store

import (
 "context"
 "crypto/rand"
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "time"

 "github.com/jackc/pgx/v5"
 "gateway-control/internal/model"
)

// RotateServiceAPIKey creates a cryptographically random key, storing only its SHA-256 digest.
// The plaintext is returned only in this response; never stored or logged.
func (s *Store) RotateServiceAPIKey(ctx context.Context, id, actor string) (string, model.Service, error) {
 raw := make([]byte, 32)
 if _,err:=rand.Read(raw);err!=nil{return "",model.Service{},err}
 secret:="gw_"+hex.EncodeToString(raw)
 sum:=sha256.Sum256([]byte(secret))
 tx,err:=s.pool.Begin(ctx)
 if err!=nil{return "",model.Service{},err}
 defer tx.Rollback(ctx)
 var oldMode string
 err=tx.QueryRow(ctx,"SELECT auth_mode FROM services WHERE id=$1 FOR UPDATE",id).Scan(&oldMode)
 if errors.Is(err,pgx.ErrNoRows){return "",model.Service{},ErrNotFound}
 if err!=nil{return "",model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE services SET auth_mode=$2, auth_key_hash=$3, version=version+1, updated_at=$4 WHERE id=$1",
   id,model.AuthAPIKey,hex.EncodeToString(sum[:]),time.Now().UTC())
 if err!=nil{return "",model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"service.auth.rotate","service:"+id,
   map[string]string{"mode":oldMode},map[string]string{"mode":model.AuthAPIKey},"ok");err!=nil{return "",model.Service{},err}
 if err=tx.Commit(ctx);err!=nil{return "",model.Service{},err}
 service,err:=s.GetService(ctx,id)
 return secret,service,err
}

func (s *Store) DisableServiceAPIKey(ctx context.Context,id,actor string)(model.Service,error) {
 tx,err:=s.pool.Begin(ctx)
 if err!=nil{return model.Service{},err}
 defer tx.Rollback(ctx)
 var oldMode string
 err=tx.QueryRow(ctx,"SELECT auth_mode FROM services WHERE id=$1 FOR UPDATE",id).Scan(&oldMode)
 if errors.Is(err,pgx.ErrNoRows){return model.Service{},ErrNotFound}
 if err!=nil{return model.Service{},err}
 _,err=tx.Exec(ctx,"UPDATE services SET auth_mode='public',auth_key_hash='',version=version+1,updated_at=$2 WHERE id=$1",id,time.Now().UTC())
 if err!=nil{return model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"service.auth.disable","service:"+id,
   map[string]string{"mode":oldMode},map[string]string{"mode":model.AuthPublic},"ok");err!=nil{return model.Service{},err}
 if err=tx.Commit(ctx);err!=nil{return model.Service{},err}
 return s.GetService(ctx,id)
}
