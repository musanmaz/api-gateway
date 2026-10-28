package store

import (
 "context"
 "errors"
 "regexp"
 "strings"
 "time"

 "github.com/jackc/pgx/v5"
 "gateway-control/internal/model"
 "gateway-control/internal/validate"
)

type UpstreamConfig struct {
 Mode string `json:"mode"`
 Username string `json:"username"`
 HeaderName string `json:"headerName"`
 Secret string `json:"secret"`
 RequiredClientHeader string `json:"requiredClientHeader"`
}

var validHeaderName=regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,79}$`)

func validGatewayHeader(raw string) bool {
 n:=strings.ToLower(raw)
 switch n {
 case "host","authorization","connection","content-length","transfer-encoding","upgrade","cookie",
      "proxy-authorization","te","trailer","expect","x-gateway-key","x-forwarded-for","x-forwarded-host","x-forwarded-proto":
  return false
 }
 return validHeaderName.MatchString(raw)
}

func (s *Store) ConfigureUpstreamAuth(ctx context.Context,id,actor string,in UpstreamConfig,ciphertext string) (model.Service,error) {
 in.Mode=strings.TrimSpace(strings.ToLower(in.Mode))
 in.HeaderName=strings.TrimSpace(in.HeaderName)
 in.RequiredClientHeader=strings.TrimSpace(in.RequiredClientHeader)
 in.Username=strings.TrimSpace(in.Username)
 if in.Mode=="" {in.Mode="none"}
 var details []string
 if len(in.RequiredClientHeader)>0 && !validGatewayHeader(in.RequiredClientHeader) {
  details=append(details,"zorunlu istemci header adı geçersiz veya yasaklı")
 }
 switch in.Mode {
 case "none":
  if in.HeaderName!="" || in.Username!="" || ciphertext!="" {details=append(details,"auth kapalıyken diğer kimlik alanları boş olmalıdır")}
 case "basic":
  if len(in.Username)==0 || len(in.Username)>128 || strings.ContainsAny(in.Username,":\r\n") {
   details=append(details,"Basic Auth kullanıcı adı zorunlu; iki nokta ve satır sonu içeremez")
  }
 case "bearer":
  if in.Username!="" || in.HeaderName!="" {details=append(details,"Bearer yalnız token kabul eder")}
 case "header":
  if !validGatewayHeader(in.HeaderName) {details=append(details,"custom header adı geçersiz veya yasaklı")}
  if in.Username!="" {details=append(details,"custom header için kullanıcı adı kullanılmaz")}
 default:
  details=append(details,"auth türü none, basic, bearer veya header olmalı")
 }
 if len(details)>0 {return model.Service{},&validate.Error{Details:details}}
 tx,err:=s.pool.Begin(ctx)
 if err!=nil{return model.Service{},err}
 defer tx.Rollback(ctx)
 var currentMode,currentCipher, currentListener string
 err=tx.QueryRow(ctx,`SELECT s.upstream_auth_mode,s.upstream_auth_cipher,l.protocol
      FROM services s JOIN listeners l ON l.id=s.listener_id WHERE s.id=$1 FOR UPDATE OF s`,id).
  Scan(&currentMode,&currentCipher,&currentListener)
 if errors.Is(err,pgx.ErrNoRows){return model.Service{},ErrNotFound}
 if err!=nil{return model.Service{},err}
 if currentListener!="http" {return model.Service{},&validate.Error{Details:[]string{"backend kimlik bilgileri şu anda yalnız HTTP servislerde destekleniyor"}}}
 if in.Mode=="none" {ciphertext=""} else if ciphertext=="" {
  if currentMode!=in.Mode || currentCipher=="" {
   return model.Service{},&validate.Error{Details:[]string{"bu auth türü için parola/token/header değeri zorunlu"}}
  }
  ciphertext=currentCipher
 }
 _,err=tx.Exec(ctx,`UPDATE services SET upstream_auth_mode=$2,upstream_auth_username=$3,
  upstream_auth_header=$4,upstream_auth_cipher=$5,required_client_header=$6,
  version=version+1,updated_at=$7 WHERE id=$1`,
  id,in.Mode,in.Username,in.HeaderName,ciphertext,strings.ToLower(in.RequiredClientHeader),time.Now().UTC())
 if err!=nil{return model.Service{},err}
 if err=insertAudit(ctx,tx,actor,"service.upstream-auth.configure","service:"+id,
  map[string]string{"mode":currentMode},map[string]string{"mode":in.Mode,"requiredHeader":strings.ToLower(in.RequiredClientHeader)},"ok");err!=nil {
  return model.Service{},err
 }
 if err=tx.Commit(ctx);err!=nil{return model.Service{},err}
 return s.GetService(ctx,id)
}
