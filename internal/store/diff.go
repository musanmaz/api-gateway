package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"gateway-control/internal/model"
)

func (s *Store) Diff(ctx context.Context, id string) ([]model.DiffChange, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	svc, err := s.getServiceTx(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	services, err := s.servicesForPublish(ctx, tx, svc.FleetID, id)
	if err != nil {
		return nil, err
	}
	listeners, err := s.listeners(ctx, tx, svc.FleetID)
	if err != nil {
		return nil, err
	}
	next, _, err := buildPackage(0, svc.FleetID, services, listeners)
	if err != nil {
		return nil, err
	}
	var payload string
	err = tx.QueryRow(ctx, `SELECT payload FROM revisions WHERE fleet_id=$1 ORDER BY id DESC LIMIT 1`, svc.FleetID).Scan(&payload)
	var previous model.Package
	if err == nil {
		if err := json.Unmarshal([]byte(payload), &previous); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	return diffPackages(previous, next), nil
}

func diffPackages(previous, next model.Package) []model.DiffChange {
	before := fingerprints(previous)
	after := fingerprints(next)
	keys := map[string]struct{}{}
	for key := range before {
		keys[key] = struct{}{}
	}
	for key := range after {
		keys[key] = struct{}{}
	}
	ordered := make([]string, 0, len(keys))
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	out := []model.DiffChange{}
	for _, key := range ordered {
		parts := strings.SplitN(key, "\n", 3)
		host, path := parts[0], parts[1]
		left, hasLeft := before[key]
		right, hasRight := after[key]
		switch {
		case hasLeft && hasRight && left == right:
			continue
		case hasLeft && hasRight:
			out = append(out, model.DiffChange{Op: "change", Host: host, Path: path, Before: left, After: right})
		case hasRight:
			out = append(out, model.DiffChange{Op: "add", Host: host, Path: path, After: right})
		default:
			out = append(out, model.DiffChange{Op: "remove", Host: host, Path: path, Before: left})
		}
	}
	return out
}

func fingerprints(pkg model.Package) map[string]string {
	clusters := map[string]string{}
	for _, cluster := range pkg.Clusters {
		parts := make([]string, 0, len(cluster.Endpoints))
		for _, endpoint := range cluster.Endpoints {
			parts = append(parts, fmt.Sprintf("%s:%d", endpoint.Address, endpoint.Port))
		}
		sort.Strings(parts)
		clusters[cluster.Name] = strings.Join(parts, ",")
	}
	out := map[string]string{}
	for _, host := range pkg.VirtualHosts {
		domains := append([]string(nil), host.Domains...)
		sort.Strings(domains)
		for _, route := range host.Routes {
			methods := append([]string(nil), route.Methods...)
			sort.Strings(methods)
			methodText := strings.Join(methods, ",")
			if methodText == "" {
				methodText = "tüm metodlar"
			}
			detail := fmt.Sprintf("%s %s → %s", route.MatchType, methodText, clusters[route.Cluster])
			for _, domain := range domains {
				out[domain+"\n"+route.Path+"\n"+route.MatchType] = detail
			}
		}
	}
	return out
}
