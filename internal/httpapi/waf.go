package httpapi

import (
 "net/http"
 "gateway-control/internal/model"
)
func (s *Server) setServiceWAF(w http.ResponseWriter,r *http.Request){
 if current(r).Role!=model.RoleOperator {writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"});return}
 var config struct{Mode string `json:"mode"`}
 if err:=decode(w,r,&config);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"geçersiz WAF modu"});return}
 item,err:=s.store.SetWAF(r.Context(),r.PathValue("id"),current(r).Name,config.Mode)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]any{"service":item,"notice":"Yeni WAF politikası yayınlanmadan aktif değildir"})
}
