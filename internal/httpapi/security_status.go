package httpapi

import "net/http"

func(s *Server) serviceSecurityStatus(w http.ResponseWriter,r *http.Request){
 report,err:=s.store.GetServiceSecurityStatus(r.Context(),r.PathValue("id"))
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,report)
}
