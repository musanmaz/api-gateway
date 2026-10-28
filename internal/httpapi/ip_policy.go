package httpapi

import (
 "net/http"
 "gateway-control/internal/model"
)
func(s *Server) setServiceIPPolicy(w http.ResponseWriter,r *http.Request){
 if current(r).Role!=model.RoleOperator{writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"});return}
 var input struct{Allow []string `json:"allow"`; Deny []string `json:"deny"`}
 if err:=decode(w,r,&input);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"geçersiz CIDR listesi"});return}
 item,err:=s.store.SetServiceIPPolicy(r.Context(),r.PathValue("id"),current(r).Name,input.Allow,input.Deny)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]any{"service":item,"notice":"Politika kayıtlı; yayınlandıktan sonra etkin."})
}
