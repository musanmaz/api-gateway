package validate

import (
	"testing"

	"gateway-control/internal/model"
)

func TestNormalizeRejectsWildcardAndLoopback(t *testing.T) {
	_, errs := Normalize(model.ServiceInput{
		Name:          "demo",
		Owner:         "platform",
		EnvironmentID: "env",
		TeamID:        "team",
		FleetID:       "fleet",
		Hostnames:     []string{"*.example.com"},
		Routes: []model.RouteInput{{
			Path: "/",
			Upstream: model.UpstreamInput{
				Name: "backend-a",
				Endpoints: []model.EndpointInput{{
					Address: "127.0.0.1",
					Port:    8080,
				}},
			},
		}},
	})
	if len(errs) < 2 {
		t.Fatalf("expected hostname and address errors, got %v", errs)
	}
}

func TestNormalizeAllowsComposeBackend(t *testing.T) {
	got, errs := Normalize(sample())
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	if got.Routes[0].Upstream.Algorithm != model.AlgoRoundRobin {
		t.Fatalf("algorithm = %s", got.Routes[0].Upstream.Algorithm)
	}
	if got.Routes[0].Upstream.ConnectTimeoutMs != 2000 {
		t.Fatalf("connect timeout = %d", got.Routes[0].Upstream.ConnectTimeoutMs)
	}
}

func TestConflictsOnHostname(t *testing.T) {
	errs := Conflicts("svc-2", []string{"demo.local"}, sample().Routes, []OwnedRoute{{
		ServiceID: "svc-1",
		Hostname:  "demo.local",
		MatchType: model.MatchPrefix,
		Path:      "/other",
	}})
	if len(errs) != 1 {
		t.Fatalf("expected one conflict, got %v", errs)
	}
}

func TestMetadataAddressBlocked(t *testing.T) {
	if err := CheckAddress("169.254.169.254"); err == nil {
		t.Fatal("expected metadata address to be blocked")
	}
	if err := CheckAddress("10.0.0.8"); err != nil {
		t.Fatal(err)
	}
}

func sample() model.ServiceInput {
	return model.ServiceInput{
		Name:          "demo",
		Owner:         "platform",
		EnvironmentID: "env",
		TeamID:        "team",
		FleetID:       "fleet",
		Hostnames:     []string{"Demo.Local"},
		Routes: []model.RouteInput{{
			Path: "/a",
			Upstream: model.UpstreamInput{
				Name: "backend-a",
				Endpoints: []model.EndpointInput{{
					Address: "backend-a",
					Port:    8080,
				}},
			},
		}},
	}
}
