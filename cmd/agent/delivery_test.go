package main

import (
	"testing"

	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"gateway-control/internal/compiler"
	"gateway-control/internal/model"
)

func TestAckWaitsUntilEveryResourceAndNackRestores(t *testing.T) {
	pkg := samplePush(7)
	snapshot, err := compiler.Build(pkg, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	state := &nodeState{node: "gw-local-1"}
	state.rememberGood(pkg, snapshot)
	next := samplePush(8)
	nextSnap, err := compiler.Build(next, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	state.begin(model.Envelope{Revision: 8, Payload: []byte(`{"revision":8}`)}, next, nextSnap)
	saved := 0
	if err := state.observeAck(resource.ListenerType, "8", func(model.Envelope) error {
		saved++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if saved != 0 {
		t.Fatal("listener ack saved the package")
	}
	if err := state.observeAck(resource.RouteType, "8", func(model.Envelope) error {
		saved++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := state.observeAck(resource.ClusterType, "8", func(model.Envelope) error {
		saved++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if saved != 1 {
		t.Fatalf("saves %d", saved)
	}
	if state.shouldSkip(8) == false {
		t.Fatal("acked revision was offered again")
	}
	rejected := samplePush(9)
	rejectedSnap, err := compiler.Build(rejected, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	state.begin(model.Envelope{Revision: 9}, rejected, rejectedSnap)
	restore, rejectedPush := state.noteNack("envoy reddetti")
	if !rejectedPush || restore != nextSnap {
		t.Fatal("nack did not return the last acked snapshot")
	}
	if !state.shouldSkip(9) {
		t.Fatal("nacked revision would be pushed again")
	}
	report := state.beat()
	if report.Nack == nil || *report.Nack != 9 || report.Clear {
		t.Fatalf("beat %+v", report)
	}
	if report.Acked == nil || *report.Acked != 8 {
		t.Fatalf("ack %v", report.Acked)
	}
}

func TestShouldSkipIgnoresOlderRevision(t *testing.T) {
	state := &nodeState{delivered: 4}
	if !state.shouldSkip(4) || state.shouldSkip(5) {
		t.Fatal("skip boundaries")
	}
}

func samplePush(revision int64) model.Package {
	return model.Package{
		Revision: revision, FleetID: "fleet", IssuedAt: "2026-10-26T00:00:00Z",
		Listeners: []model.PkgListener{{
			Name: "public-http", Protocol: "http", Port: 10000, RouteConfigName: "edge",
		}},
		VirtualHosts: []model.PkgVHost{{
			Name: "demo", Domains: []string{"demo.local"},
			Routes: []model.PkgRoute{{
				Name: "a", MatchType: model.MatchPrefix, Path: "/a", Cluster: "pool-a", TimeoutMs: 1000,
			}},
		}},
		Clusters: []model.PkgCluster{{
			Name: "pool-a", Algorithm: model.AlgoRoundRobin, ConnectTimeoutMs: 1000,
			Endpoints: []model.PkgEndpoint{{Address: "backend-a", Port: 8080, Weight: 1}},
		}},
	}
}
