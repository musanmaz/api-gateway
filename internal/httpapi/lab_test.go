package httpapi

import (
	"gateway-control/internal/sign"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type labTransport func(*http.Request) (*http.Response, error)

func (f labTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestLabTargetsAndAuthentication(t *testing.T) {
	s := New(nil, sign.Signer{}, Tokens{Operator: "test-operator"}).WithLabTargets("api1.example.com=backend-a,api2.example.com=backend-b,api1.example.com=other,https://evil.com=x,127.0.0.1:22=x,bad/path=x")
	if len(s.labTargets) != 2 {
		t.Fatalf("unexpected targets: %#v", s.labTargets)
	}
	r := httptest.NewRequest("GET", "/api/v1/lab", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthed request: %d", w.Code)
	}
	r.Header.Set("Authorization", "Bearer test-operator")
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "api1.example.com") {
		t.Fatalf("authorized config: %d", w.Code)
	}
}
func TestProbeRequiresExpectedBackendAndNeverForwardsToken(t *testing.T) {
	parent := httptest.NewRequest("GET", "/api/v1/lab/probe?url=http://evil.example", nil)
	parent.Header.Set("Authorization", "Bearer private-token")
	target := LabTarget{Host: "api1.example.com", URL: "https://api1.example.com/api/info", ExpectedInstance: "backend-a"}
	for _, body := range []string{`{"instance":"backend-a"}`, `{"instance":"backend-b"}`} {
		client := &http.Client{Transport: labTransport(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != target.URL || r.Header.Get("Authorization") != "" {
				t.Fatal("unexpected target or forwarded secret")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
		})}
		got := probeTarget(parent, client, target)
		if got.OK != (body == `{"instance":"backend-a"}`) {
			t.Fatalf("wrong success result: %#v", got)
		}
	}
}
