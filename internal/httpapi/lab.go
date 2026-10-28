package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type LabTarget struct {
	Host             string `json:"host"`
	ExpectedInstance string `json:"expectedInstance"`
	URL              string `json:"url"`
}
type LabResult struct {
	LabTarget
	Status    int    `json:"status"`
	LatencyMS int64  `json:"latencyMs"`
	OK        bool   `json:"ok"`
	Body      string `json:"body"`
	Error     string `json:"error,omitempty"`
	CheckedAt string `json:"checkedAt"`
}

var labHost = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

// Targets come only from the trusted deployment configuration, never request input.
// A fixed HTTPS path and no redirects prevent this from becoming a generic URL proxy.
func (s *Server) WithLabTargets(raw string) *Server {
	s.labTargets = nil
	seen := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		host, expected, ok := strings.Cut(strings.TrimSpace(entry), "=")
		if !ok || !labHost.MatchString(host) || len(host) > 253 || expected == "" || seen[host] || len(s.labTargets) >= 4 {
			continue
		}
		seen[host] = true
		s.labTargets = append(s.labTargets, LabTarget{Host: host, ExpectedInstance: expected, URL: "https://" + host + "/api/info"})
	}
	return s
}
func (s *Server) labConfig(w http.ResponseWriter, r *http.Request) {
	targets := s.labTargets
	if targets == nil {
		targets = []LabTarget{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"targets": targets, "source": "control-plane", "tlsTermination": "Caddy"})
}
func (s *Server) labProbe(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	results := make([]LabResult, len(s.labTargets))
	var wg sync.WaitGroup
	for i, target := range s.labTargets {
		wg.Add(1)
		go func(i int, target LabTarget) { defer wg.Done(); results[i] = probeTarget(r, client, target) }(i, target)
	}
	wg.Wait()
	writeJSON(w, http.StatusOK, map[string]any{"results": results, "source": "control-plane"})
}
func probeTarget(parent *http.Request, client *http.Client, target LabTarget) LabResult {
	result := LabResult{LabTarget: target, CheckedAt: time.Now().UTC().Format(time.RFC3339)}
	start := time.Now()
	req, err := http.NewRequestWithContext(parent.Context(), http.MethodGet, target.URL, nil)
	if err != nil {
		result.Error = "Test adresi geçersiz"
		return result
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Gateway-Control-Lab/1.0")
	response, err := client.Do(req)
	result.LatencyMS = time.Since(start).Milliseconds()
	if err != nil {
		result.Error = "HTTPS bağlantısı kurulamadı; DNS, TLS ve gateway durumunu kontrol edin"
		return result
	}
	defer response.Body.Close()
	result.Status = response.StatusCode
	body, err := io.ReadAll(io.LimitReader(response.Body, 4097))
	if err != nil || len(body) > 4096 {
		result.Error = "Yanıt okunamadı veya 4 KiB sınırını aştı"
		return result
	}
	result.Body = string(body)
	var info struct {
		Instance string `json:"instance"`
	}
	if err = json.Unmarshal(body, &info); err != nil {
		result.Error = "Beklenen JSON yanıtı alınamadı"
		return result
	}
	result.OK = response.StatusCode == http.StatusOK && info.Instance == target.ExpectedInstance
	if !result.OK {
		result.Error = "HTTP durumu veya cevap veren backend beklenenle eşleşmiyor"
	}
	return result
}
