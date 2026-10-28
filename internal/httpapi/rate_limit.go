package httpapi

import(
 "net/http"
 "gateway-control/internal/model"
)

func(s *Server) setServiceRateLimit(w http.ResponseWriter,r *http.Request){
 if current(r).Role!=model.RoleOperator {writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"});return}
 var body struct{RequestsPerSecond int `json:"requestsPerSecond"`}
 if err:=decode(w,r,&body);err!=nil {writeJSON(w,http.StatusBadRequest,map[string]string{"error":"geçersiz rate limit isteği"});return}
 item,err:=s.store.SetRateLimit(r.Context(),r.PathValue("id"),current(r).Name,body.RequestsPerSecond)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]any{"service":item,"notice":"Değişikliğin etkinleşmesi için tüm node'lara yayınlayın"})
}
