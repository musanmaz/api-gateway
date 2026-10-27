package httpapi

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"gateway-control/internal/model"
	"gateway-control/internal/sign"
	"gateway-control/internal/store"
	"gateway-control/internal/validate"
)

type Tokens struct {
	Operator string
	Viewer   string
	Agent    string
}

func LoadTokens(operatorPath, viewerPath, agentPath string) (Tokens, error) {
	op, err := readSecret(operatorPath)
	if err != nil {
		return Tokens{}, err
	}
	view, err := readSecret(viewerPath)
	if err != nil {
		return Tokens{}, err
	}
	agent, err := readSecret(agentPath)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{Operator: op, Viewer: view, Agent: agent}, nil
}

func readSecret(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(raw))
	if value == "" {
		return "", errors.New("boş kimlik dosyası: " + path)
	}
	return value, nil
}

type Server struct {
	store  *store.Store
	signer sign.Signer
	tokens Tokens
	limit  *limiter
	tryURL string
}

func New(st *store.Store, signer sign.Signer, tokens Tokens) *Server {
	return &Server{store: st, signer: signer, tokens: tokens, limit: newLimiter(100, time.Second)}
}

func (s *Server) WithTryURL(raw string) *Server {
	s.tryURL = cleanTryURL(raw)
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /api/v1/bootstrap", s.bootstrap)
	mux.HandleFunc("GET /api/v1/overview", s.overview)
	mux.HandleFunc("GET /api/v1/services", s.listServices)
	mux.HandleFunc("POST /api/v1/services", s.createService)
	mux.HandleFunc("GET /api/v1/services/{id}", s.getService)
	mux.HandleFunc("PUT /api/v1/services/{id}", s.updateService)
	mux.HandleFunc("DELETE /api/v1/services/{id}", s.deleteService)
	mux.HandleFunc("POST /api/v1/services/{id}/validate", s.validateService)
	mux.HandleFunc("GET /api/v1/services/{id}/diff", s.diffService)
	mux.HandleFunc("POST /api/v1/services/{id}/publish", s.publishService)
	mux.HandleFunc("GET /api/v1/revisions", s.listRevisions)
	mux.HandleFunc("GET /api/v1/revisions/{id}", s.getRevision)
	mux.HandleFunc("POST /api/v1/revisions/{id}/rollback", s.rollbackRevision)
	mux.HandleFunc("GET /api/v1/audit", s.listAudit)
	mux.HandleFunc("GET /api/v1/agent/package", s.agentPackage)
	mux.HandleFunc("POST /api/v1/agent/heartbeat", s.agentHeartbeat)
	return s.wrap(mux)
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		principal, ok := s.authenticate(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "kimlik doğrulanamadı"})
			return
		}
		if !s.limit.allow(principal.Role + ":" + principal.Name) {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "istek sınırı aşıldı"})
			return
		}
		if principal.Role == model.RoleAgent && !strings.HasPrefix(r.URL.Path, "/api/v1/agent/") {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "bu role için kapalı"})
			return
		}
		if principal.Role == model.RoleViewer && r.Method != http.MethodGet {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "bu role için kapalı"})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/agent/") && principal.Role != model.RoleAgent && principal.Role != model.RoleOperator {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "bu role için kapalı"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
	})
}

func (s *Server) authenticate(r *http.Request) (principal, bool) {
	header := r.Header.Get("Authorization")
	token, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || strings.TrimSpace(token) == "" {
		return principal{}, false
	}
	token = strings.TrimSpace(token)
	switch {
	case matchToken(token, s.tokens.Operator):
		return principal{Name: "operator", Role: model.RoleOperator}, true
	case matchToken(token, s.tokens.Viewer):
		return principal{Name: "viewer", Role: model.RoleViewer}, true
	case matchToken(token, s.tokens.Agent):
		return principal{Name: "agent", Role: model.RoleAgent}, true
	default:
		return principal{}, false
	}
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) bootstrap(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.Bootstrap(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"environmentId": item.EnvironmentID,
		"environment":   item.Environment,
		"teamId":        item.TeamID,
		"team":          item.Team,
		"fleetId":       item.FleetID,
		"fleet":         item.Fleet,
		"listener":      item.Listener,
		"listeners":     item.Listeners,
		"protocols":     protocols(),
		"tryUrl":        s.tryURL,
	})
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.Overview(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	page, err := s.store.ListServices(r.Context(), limit, offset, strings.TrimSpace(r.URL.Query().Get("q")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	var input model.ServiceInput
	if err := decode(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "istek gövdesi okunamadı"})
		return
	}
	item, err := s.store.Create(r.Context(), input, current(r).Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.GetService(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	var input model.ServiceInput
	if err := decode(w, r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "istek gövdesi okunamadı"})
		return
	}
	item, err := s.store.Update(r.Context(), r.PathValue("id"), input, current(r).Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Delete(r.Context(), r.PathValue("id"), current(r).Name); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) validateService(w http.ResponseWriter, r *http.Request) {
	item, errs, err := s.store.Validate(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"service": item, "errors": errs, "valid": len(errs) == 0})
}

func (s *Server) diffService(w http.ResponseWriter, r *http.Request) {
	changes, err := s.store.Diff(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"changes": changes})
}

func (s *Server) publishService(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "istek gövdesi okunamadı"})
		return
	}
	var input struct {
		ExpectedVersion int    `json:"expectedVersion"`
		Node            string `json:"node"`
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "istek gövdesi okunamadı"})
			return
		}
	}
	revision, replayed, err := s.store.Publish(r.Context(), r.PathValue("id"), input.ExpectedVersion, input.Node, current(r).Name, strings.TrimSpace(r.Header.Get("Idempotency-Key")), store.RequestHash(body), s.signer)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revision": revision, "replayed": replayed})
}

func (s *Server) listRevisions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := s.store.ListRevisions(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getRevision(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "revision numarası geçersiz"})
		return
	}
	item, err := s.store.GetRevision(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) rollbackRevision(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "revision numarası geçersiz"})
		return
	}
	item, err := s.store.Rollback(r.Context(), id, current(r).Name, s.signer)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revision": item})
}

func (s *Server) listAudit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := s.store.ListAudit(r.Context(), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) agentPackage(w http.ResponseWriter, r *http.Request) {
	env, err := s.store.DesiredEnvelope(r.Context(), r.URL.Query().Get("node"))
	if errors.Is(err, store.ErrNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (s *Server) agentHeartbeat(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Node              string        `json:"node"`
		DeliveredRevision *int64        `json:"deliveredRevision"`
		AckedRevision     *int64        `json:"ackedRevision"`
		NackRevision      *int64        `json:"nackRevision"`
		NackDetail        string        `json:"nackDetail"`
		ClearNack         bool          `json:"clearNack"`
		Probes            []model.Probe `json:"probes"`
	}
	if err := decode(w, r, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "istek gövdesi okunamadı"})
		return
	}
	if err := s.store.Heartbeat(r.Context(), store.HeartbeatInput{
		Node: body.Node, DeliveredRevision: body.DeliveredRevision, AckedRevision: body.AckedRevision,
		NackRevision: body.NackRevision, NackDetail: body.NackDetail, ClearNack: body.ClearNack, Probes: body.Probes,
	}); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decode(w http.ResponseWriter, r *http.Request, dest any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dest)
}

func writeError(w http.ResponseWriter, err error) {
	var invalid *validate.Error
	switch {
	case errors.As(err, &invalid):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "doğrulama başarısız", "details": invalid.Details})
	case errors.Is(err, store.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "kayıt bulunamadı"})
	case errors.Is(err, store.ErrConflict):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "çakışma"})
	case errors.Is(err, store.ErrVersion):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "kayıt başka bir işlemle değişti"})
	case errors.Is(err, store.ErrCurrent):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "bu revision zaten istenen paket"})
	case errors.Is(err, store.ErrDraftOnly):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "yayımlanmış servis bu sürümde değiştirilemez"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "işlem tamamlanamadı"})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}

func protocols() []map[string]string {
	return []map[string]string{
		{"id": "http", "state": "available"},
		{"id": "https", "state": "available"},
		{"id": "http2", "state": "available"},
		{"id": "websocket", "state": "available"},
		{"id": "sse", "state": "available"},
		{"id": "grpc", "state": "available"},
		{"id": "tcp", "state": "available"},
		{"id": "udp", "state": "available"},
		{"id": "tls_passthrough", "state": "available"},
	}
}

func matchToken(got, want string) bool {
	if want == "" {
		return false
	}
	a := sha256.Sum256([]byte(got))
	b := sha256.Sum256([]byte(want))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

type principal struct {
	Name string
	Role string
}

type principalKey struct{}

func current(r *http.Request) principal {
	value, _ := r.Context().Value(principalKey{}).(principal)
	return value
}

type limiter struct {
	mu     sync.Mutex
	window time.Duration
	max    int
	hits   map[string][]time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{window: window, max: max, hits: map[string][]time.Time{}}
}

func (l *limiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	existing := l.hits[key]
	recent := make([]time.Time, 0, len(existing)+1)
	for _, at := range existing {
		if now.Sub(at) < l.window {
			recent = append(recent, at)
		}
	}
	if len(recent) >= l.max {
		l.hits[key] = recent
		return false
	}
	l.hits[key] = append(recent, now)
	return true
}
