package httpapi

import (
 "net/http"
 "gateway-control/internal/model"
)

func (s *Server) setRouteRateLimit(w http.ResponseWriter,r *http.Request){
 if current(r).Role!=model.RoleOperator{writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"});return}
 var in struct{RequestsPerSecond int `json:"requestsPerSecond"`}
 if err:=decode(w,r,&in);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"geçersiz endpoint limit"});return}
 item,err:=s.store.SetRouteRateLimit(r.Context(),r.PathValue("id"),r.PathValue("routeId"),current(r).Name,in.RequestsPerSecond)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]any{"service":item,"notice":"Endpoint politikası kaydedildi. Yayınlayarak etkinleştirin."})
}
