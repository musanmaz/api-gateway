package compiler

import (
	"testing"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"gateway-control/internal/model"
)

func TestBuildHasNoWildcardOrRetry(t *testing.T) {
	snapshot, err := Build(samplePackage(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	routes := snapshot.GetResources(resource.RouteType)
	config, ok := routes["edge"].(*route.RouteConfiguration)
	if !ok {
		t.Fatalf("route resource type %T", routes["edge"])
	}
	if len(config.VirtualHosts) != 1 {
		t.Fatalf("virtual hosts = %d", len(config.VirtualHosts))
	}
	for _, domain := range config.VirtualHosts[0].Domains {
		if domain == "*" {
			t.Fatal("wildcard domain was emitted")
		}
	}
	for _, item := range config.VirtualHosts[0].Routes {
		if item.GetRoute().GetRetryPolicy() != nil {
			t.Fatal("retry policy was emitted")
		}
	}
	clusters := snapshot.GetResources(resource.ClusterType)
	backend, ok := clusters["pool-a"].(*cluster.Cluster)
	if !ok {
		t.Fatalf("cluster type %T", clusters["pool-a"])
	}
	if backend.GetType() != cluster.Cluster_STRICT_DNS {
		t.Fatalf("discovery = %s", backend.GetType())
	}
	if backend.GetCircuitBreakers().GetThresholds()[0].GetMaxRetries().GetValue() != 0 {
		t.Fatal("retry budget was not zero")
	}
}

func TestBuildRejectsWildcard(t *testing.T) {
	pkg := samplePackage()
	pkg.VirtualHosts[0].Domains = []string{"*"}
	if _, err := Build(pkg, nil, nil); err == nil {
		t.Fatal("wildcard package was accepted")
	}
}

func TestEmptyPackageHasNoListener(t *testing.T) {
	snapshot, err := Build(model.Package{Revision: 1, FleetID: "fleet", IssuedAt: "2026-10-22T00:00:00Z"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.GetResources(resource.ListenerType)) != 0 {
		t.Fatal("empty package opened a listener")
	}
}

func samplePackage() model.Package {
	return model.Package{
		Revision: 7,
		FleetID:  "fleet",
		IssuedAt: "2026-10-22T00:00:00Z",
		Listeners: []model.PkgListener{{
			Name: "public-http", Protocol: "http", Port: 10000, RouteConfigName: "edge",
		}},
		VirtualHosts: []model.PkgVHost{{
			Name: "demo", Domains: []string{"demo.local"},
			Routes: []model.PkgRoute{{
				Name: "a", MatchType: model.MatchPrefix, Path: "/a", Cluster: "pool-a", TimeoutMs: 15000,
				Methods: []string{"POST"},
			}},
		}},
		Clusters: []model.PkgCluster{{
			Name: "pool-a", Algorithm: model.AlgoRoundRobin, ConnectTimeoutMs: 2000,
			Endpoints: []model.PkgEndpoint{{Address: "backend-a", Port: 8080, Weight: 1}},
		}},
	}
}
