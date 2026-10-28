package httpapi

import (
 "net/http"
 "strconv"
)
func (s *Server) serviceTraffic(w http.ResponseWriter,r *http.Request){
 minutes,_:=strconv.Atoi(r.URL.Query().Get("minutes"))
 report,err:=s.store.ServiceTraffic(r.Context(),r.PathValue("id"),minutes)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,report)
}
