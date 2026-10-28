package httpapi

import (
 "net/http"
 "os"
 "strings"

 "gateway-control/internal/model"
 "gateway-control/internal/secretbox"
 "gateway-control/internal/store"
)

func (s *Server) configureUpstreamAuth(w http.ResponseWriter,r *http.Request){
 if current(r).Role!=model.RoleOperator {
  writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"});return
 }
 var input store.UpstreamConfig
 if err:=decode(w,r,&input);err!=nil {
  writeJSON(w,http.StatusBadRequest,map[string]string{"error":"güvenlik ayarı okunamadı"});return
 }
 // Never write secrets to logs, error responses, audit events or the service JSON API.
 var ciphertext string
 if input.Secret!="" {
  if len(input.Secret)>4096 || strings.ContainsAny(input.Secret,"\r\n") {
   writeJSON(w,http.StatusUnprocessableEntity,map[string]string{"error":"gizli değer uzunluğu veya biçimi geçersiz"});return
  }
  key,err:=secretbox.ReadKey(os.Getenv("GATEWAY_CREDENTIAL_KEY_FILE"))
  if err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"gizli anahtar kasasına ulaşılamıyor"});return}
  ciphertext,err=secretbox.Encrypt(key,input.Secret)
  if err!=nil{writeJSON(w,http.StatusInternalServerError,map[string]string{"error":"gizli değer kaydedilemedi"});return}
 }
 // Prevent incidental downstream leakage of the plaintext after this request.
 input.Secret=""
 item,err:=s.store.ConfigureUpstreamAuth(r.Context(),r.PathValue("id"),current(r).Name,input,ciphertext)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]any{"service":item,"notice":"Kaydedildi; etkinleşmesi için iki gateway node'una yayınlanmalı"})
}
