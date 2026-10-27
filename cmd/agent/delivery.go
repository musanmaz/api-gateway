package main

import (
	"strconv"
	"sync"

	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"

	"gateway-control/internal/model"
)

type pendingPush struct {
	env      model.Envelope
	revision int64
	snapshot *cache.Snapshot
	need     map[string]struct{}
	got      map[string]struct{}
}

type nodeState struct {
	mu           sync.Mutex
	node         string
	pkg          *model.Package
	restorePkg   *model.Package
	restore      *cache.Snapshot
	pending      *pendingPush
	delivered    int64
	ackedVersion string
	nackRevision int64
	nackDetail   string
	clearNack    bool
}

type beat struct {
	Delivered *int64
	Acked     *int64
	Nack      *int64
	Detail    string
	Clear     bool
	Pkg       *model.Package
}

func (s *nodeState) rememberGood(pkg model.Package, snapshot *cache.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := pkg
	s.restorePkg = &copied
	s.pkg = s.restorePkg
	s.restore = snapshot
	s.delivered = pkg.Revision
}

func (s *nodeState) begin(env model.Envelope, pkg model.Package, snapshot *cache.Snapshot) {
	s.mu.Lock()
	defer s.mu.Unlock()
	copied := pkg
	s.pkg = &copied
	s.delivered = pkg.Revision
	s.pending = &pendingPush{
		env: env, revision: pkg.Revision, snapshot: snapshot,
		need: ackTypes(snapshot), got: map[string]struct{}{},
	}
}

func (s *nodeState) shouldSkip(revision int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if revision == 0 {
		return true
	}
	if s.pending != nil && s.pending.revision == revision {
		return true
	}
	if revision == s.nackRevision {
		return true
	}
	return s.pending == nil && revision <= s.delivered
}

func (s *nodeState) observeAck(typeURL, version string, save func(model.Envelope) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		if version != "" {
			s.ackedVersion = version
		}
		return nil
	}
	if version != strconv.FormatInt(s.pending.revision, 10) {
		return nil
	}
	if _, ok := s.pending.need[typeURL]; ok {
		s.pending.got[typeURL] = struct{}{}
	}
	if len(s.pending.need) > 0 && len(s.pending.got) < len(s.pending.need) {
		return nil
	}
	if err := save(s.pending.env); err != nil {
		return err
	}
	s.restore = s.pending.snapshot
	copied := model.Package{}
	if s.pkg != nil {
		copied = *s.pkg
	}
	s.restorePkg = &copied
	s.ackedVersion = version
	s.nackRevision = 0
	s.nackDetail = ""
	s.clearNack = true
	s.pending = nil
	return nil
}

func (s *nodeState) noteNack(detail string) (*cache.Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return nil, false
	}
	s.nackRevision = s.pending.revision
	s.nackDetail = model.NackDetail(detail)
	s.delivered = s.pending.revision
	s.clearNack = false
	if s.restorePkg != nil {
		copied := *s.restorePkg
		s.pkg = &copied
	}
	restore := s.restore
	s.pending = nil
	return restore, true
}

func (s *nodeState) beat() beat {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := beat{Pkg: s.pkg, Detail: s.nackDetail, Clear: s.clearNack && s.nackRevision == 0}
	if s.delivered > 0 {
		value := s.delivered
		out.Delivered = &value
	}
	if value, err := strconv.ParseInt(s.ackedVersion, 10, 64); err == nil && value > 0 {
		out.Acked = &value
	}
	if s.nackRevision > 0 {
		value := s.nackRevision
		out.Nack = &value
		out.Clear = false
	}
	return out
}

func (s *nodeState) nackCleared() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.nackRevision == 0 {
		s.clearNack = false
	}
}

func ackTypes(snapshot *cache.Snapshot) map[string]struct{} {
	need := map[string]struct{}{}
	if snapshot == nil {
		return need
	}
	for _, typ := range []string{resource.ListenerType, resource.RouteType, resource.ClusterType, resource.SecretType} {
		if len(snapshot.GetResources(typ)) > 0 {
			need[typ] = struct{}{}
		}
	}
	return need
}
