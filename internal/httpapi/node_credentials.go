package httpapi
import(
 "net/http"
 "gateway-control/internal/model"
)
func (s *Server) nodeAgentCredential(w http.ResponseWriter,r *http.Request){
 if current(r).Role!=model.RoleOperator{writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"});return}
 var body struct{SHA256 string `json:"sha256"`}
 if err:=decode(w,r,&body);err!=nil{writeJSON(w,http.StatusBadRequest,map[string]string{"error":"invalid SHA256"});return}
 err:=s.store.SetNodeCredential(r.Context(),r.PathValue("name"),body.SHA256,current(r).Name)
 if err!=nil{writeError(w,err);return}
 writeJSON(w,http.StatusOK,map[string]string{"status":"configured"})
}
