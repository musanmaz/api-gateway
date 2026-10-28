package httpapi

import (
    "net/http"
    "gateway-control/internal/model"
)

func (s *Server) rotateServiceKey(w http.ResponseWriter,r *http.Request) {
    if current(r).Role != model.RoleOperator {
        writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"})
        return
    }
    secret, service,err:=s.store.RotateServiceAPIKey(r.Context(),r.PathValue("id"),current(r).Name)
    if err!=nil {writeError(w,err);return}
    writeJSON(w,http.StatusOK,map[string]any{"key":secret,"service":service,"notice":"Anahtar yalnızca bir kez gösterilir. Korumanın etkinleşmesi için tüm node'lara yayınlayın."})
}

func (s *Server) disableServiceKey(w http.ResponseWriter,r *http.Request) {
    if current(r).Role != model.RoleOperator {
        writeJSON(w,http.StatusForbidden,map[string]string{"error":"operator gerekli"})
        return
    }
    service,err:=s.store.DisableServiceAPIKey(r.Context(),r.PathValue("id"),current(r).Name)
    if err!=nil {writeError(w,err);return}
    writeJSON(w,http.StatusOK,map[string]any{"service":service,"notice":"Public modun geçerli olması için tüm node'lara yayınlayın."})
}
