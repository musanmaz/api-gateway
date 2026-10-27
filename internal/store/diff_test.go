package store

import (
	"testing"

	"gateway-control/internal/model"
)

func TestDiffPackages(t *testing.T) {
	previous := samplePackage("demo.local", "/a", "GET", "pool-a", "backend-a")
	next := samplePackage("demo.local", "/a", "GET", "pool-a", "backend-a")
	next.VirtualHosts[0].Routes = append(next.VirtualHosts[0].Routes, model.PkgRoute{
		MatchType: "prefix", Path: "/b", Methods: []string{"POST"}, Cluster: "pool-b",
	})
	next.Clusters = append(next.Clusters, model.PkgCluster{
		Name: "pool-b", Endpoints: []model.PkgEndpoint{{Address: "backend-b", Port: 8080, Weight: 1}},
	})

	added := diffPackages(previous, next)
	if len(added) != 1 || added[0].Op != "add" || added[0].Path != "/b" || added[0].Host != "demo.local" {
		t.Fatalf("eklenen rota görülmedi: %+v", added)
	}
	removed := diffPackages(next, previous)
	if len(removed) != 1 || removed[0].Op != "remove" || removed[0].Path != "/b" {
		t.Fatalf("kalkan rota görülmedi: %+v", removed)
	}
	if len(diffPackages(previous, previous)) != 0 {
		t.Fatal("aynı paket fark üretmemeli")
	}
}

func samplePackage(host, path, method, cluster, address string) model.Package {
	return model.Package{
		VirtualHosts: []model.PkgVHost{{
			Domains: []string{host},
			Routes: []model.PkgRoute{{
				MatchType: "prefix", Path: path, Methods: []string{method}, Cluster: cluster,
			}},
		}},
		Clusters: []model.PkgCluster{{
			Name: cluster, Endpoints: []model.PkgEndpoint{{Address: address, Port: 8080, Weight: 1}},
		}},
	}
}
