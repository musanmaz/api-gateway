package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"gateway-control/internal/canonical"
	"gateway-control/internal/model"
	"gateway-control/internal/sign"
	"gateway-control/internal/validate"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrDraftOnly = errors.New("draft only")
	ErrVersion   = errors.New("version")
	ErrCurrent   = errors.New("current")
)

type Store struct {
	pool *pgxpool.Pool
}

type Page struct {
	Items  []model.Service `json:"items"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

type Overview struct {
	Services struct {
		Total  int `json:"total"`
		Active int `json:"active"`
		Draft  int `json:"draft"`
	} `json:"services"`
	Nodes          []model.Node `json:"nodes"`
	Telemetry      string       `json:"telemetry"`
	ExternalProbe  string       `json:"externalProbe"`
	PendingDrafts  int          `json:"pendingDrafts"`
	ActiveRevision *int64       `json:"activeRevision"`
}

type Bootstrap struct {
	EnvironmentID string           `json:"environmentId"`
	Environment   string           `json:"environment"`
	TeamID        string           `json:"teamId"`
	Team          string           `json:"team"`
	FleetID       string           `json:"fleetId"`
	Fleet         string           `json:"fleet"`
	Listener      model.Listener   `json:"listener"`
	Listeners     []model.Listener `json:"listeners"`
}

func Open(ctx context.Context, dsn string) (*Store, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, "SET TIME ZONE 'UTC'")
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) Bootstrap(ctx context.Context) (Bootstrap, error) {
	var out Bootstrap
	err := s.pool.QueryRow(ctx, `
		SELECT e.id::text, e.name, t.id::text, t.name, f.id::text, f.name,
		       l.id::text, l.fleet_id::text, l.name, l.protocol, l.port
		FROM listeners l
		JOIN fleets f ON f.id = l.fleet_id
		JOIN environments e ON e.id = f.environment_id
		JOIN teams t ON t.name = 'platform'
		WHERE l.protocol = 'http'
		ORDER BY l.port
		LIMIT 1`).Scan(
		&out.EnvironmentID, &out.Environment, &out.TeamID, &out.Team, &out.FleetID, &out.Fleet,
		&out.Listener.ID, &out.Listener.FleetID, &out.Listener.Name, &out.Listener.Protocol, &out.Listener.Port,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bootstrap{}, ErrNotFound
	}
	if err != nil {
		return Bootstrap{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, fleet_id::text, name, protocol, port
		FROM listeners WHERE fleet_id=$1 ORDER BY port, protocol`, out.FleetID)
	if err != nil {
		return Bootstrap{}, err
	}
	defer rows.Close()
	out.Listeners = []model.Listener{}
	for rows.Next() {
		var item model.Listener
		if err := rows.Scan(&item.ID, &item.FleetID, &item.Name, &item.Protocol, &item.Port); err != nil {
			return Bootstrap{}, err
		}
		out.Listeners = append(out.Listeners, item)
	}
	return out, rows.Err()
}

func (s *Store) Overview(ctx context.Context) (Overview, error) {
	var out Overview
	out.Telemetry = "disconnected"
	out.ExternalProbe = "not_configured"
	err := s.pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE status = 'active'),
		       count(*) FILTER (WHERE status = 'draft')
		FROM services`).Scan(&out.Services.Total, &out.Services.Active, &out.Services.Draft)
	if err != nil {
		return Overview{}, err
	}
	out.PendingDrafts = out.Services.Draft
	err = s.pool.QueryRow(ctx, `SELECT max(id) FROM revisions`).Scan(&out.ActiveRevision)
	if err != nil {
		return Overview{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, fleet_id::text, name, location, desired_revision, delivered_revision,
		       acked_revision, nack_revision, nack_detail, last_heartbeat_at, probe
		FROM nodes ORDER BY name`)
	if err != nil {
		return Overview{}, err
	}
	defer rows.Close()
	now := time.Now().UTC()
	for rows.Next() {
		node, err := scanNode(rows, now)
		if err != nil {
			return Overview{}, err
		}
		out.Nodes = append(out.Nodes, node)
	}
	if out.Nodes == nil {
		out.Nodes = []model.Node{}
	}
	return out, rows.Err()
}

func (s *Store) ListServices(ctx context.Context, limit, offset int, query string) (Page, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	page := Page{Limit: limit, Offset: offset, Items: []model.Service{}}
	pattern := likePattern(query)
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM services s
		WHERE $1 = '' OR s.name ILIKE $2 ESCAPE '\' OR EXISTS (
			SELECT 1 FROM hostnames h WHERE h.service_id = s.id AND h.hostname ILIKE $2 ESCAPE '\'
		)`, query, pattern).Scan(&page.Total); err != nil {
		return Page{}, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text FROM services s
		WHERE $1 = '' OR s.name ILIKE $2 ESCAPE '\' OR EXISTS (
			SELECT 1 FROM hostnames h WHERE h.service_id = s.id AND h.hostname ILIKE $2 ESCAPE '\'
		)
		ORDER BY s.updated_at DESC
		LIMIT $3 OFFSET $4`, query, pattern, limit, offset)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return Page{}, err
		}
		item, err := s.GetService(ctx, id)
		if err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, item)
	}
	return page, rows.Err()
}

func (s *Store) GetService(ctx context.Context, id string) (model.Service, error) {
	return s.getService(ctx, s.pool, id)
}

func (s *Store) Create(ctx context.Context, in model.ServiceInput, actor string) (model.Service, error) {
	normalized, errs := validate.Normalize(in)
	if len(errs) > 0 {
		return model.Service{}, &validate.Error{Details: errs}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Service{}, err
	}
	defer tx.Rollback(ctx)
	if err := s.ensureScope(ctx, tx, normalized); err != nil {
		return model.Service{}, err
	}
	listener, err := s.listener(ctx, tx, normalized)
	if err != nil {
		return model.Service{}, err
	}
	existing, err := s.ownedRoutes(ctx, tx, normalized.FleetID, "")
	if err != nil {
		return model.Service{}, err
	}
	if errs := validate.Conflicts("", normalized.Hostnames, normalized.Routes, existing); len(errs) > 0 {
		return model.Service{}, &validate.Error{Details: errs}
	}
	id := newID()
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO services (
			id, team_id, environment_id, fleet_id, listener_id, name, description, owner,
			importance, status, version, published_version, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'draft',1,0,$10,$10)`,
		id, normalized.TeamID, normalized.EnvironmentID, normalized.FleetID, listener.ID,
		normalized.Name, normalized.Description, normalized.Owner, normalized.Importance, now)
	if isUnique(err) {
		return model.Service{}, ErrConflict
	}
	if err != nil {
		return model.Service{}, err
	}
	if err := insertGraph(ctx, tx, id, normalized.FleetID, normalized); err != nil {
		if isUnique(err) {
			return model.Service{}, ErrConflict
		}
		return model.Service{}, err
	}
	if err := insertAudit(ctx, tx, actor, "service.create", "service:"+id, nil, map[string]any{
		"name": normalized.Name, "hostnames": normalized.Hostnames,
	}, "ok"); err != nil {
		return model.Service{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Service{}, err
	}
	return s.GetService(ctx, id)
}

func (s *Store) Update(ctx context.Context, id string, in model.ServiceInput, actor string) (model.Service, error) {
	normalized, errs := validate.Normalize(in)
	if len(errs) > 0 {
		return model.Service{}, &validate.Error{Details: errs}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Service{}, err
	}
	defer tx.Rollback(ctx)
	current, err := s.getServiceTx(ctx, tx, id)
	if err != nil {
		return model.Service{}, err
	}
	if in.ExpectedVersion != 0 && in.ExpectedVersion != current.Version {
		return model.Service{}, ErrVersion
	}
	normalized.FleetID = current.FleetID
	normalized.EnvironmentID = current.EnvironmentID
	normalized.TeamID = current.TeamID
	listener, err := s.listener(ctx, tx, normalized)
	if err != nil {
		return model.Service{}, err
	}
	existing, err := s.ownedRoutes(ctx, tx, current.FleetID, id)
	if err != nil {
		return model.Service{}, err
	}
	if errs := validate.Conflicts(id, normalized.Hostnames, normalized.Routes, existing); len(errs) > 0 {
		return model.Service{}, &validate.Error{Details: errs}
	}
	tag, err := tx.Exec(ctx, `
		UPDATE services
		SET name=$2, description=$3, owner=$4, importance=$5, listener_id=$6,
		    version=version+1, updated_at=$7
		WHERE id=$1 AND version=$8`,
		id, normalized.Name, normalized.Description, normalized.Owner, normalized.Importance, listener.ID, time.Now().UTC(), current.Version)
	if err != nil {
		if isUnique(err) {
			return model.Service{}, ErrConflict
		}
		return model.Service{}, err
	}
	if tag.RowsAffected() != 1 {
		return model.Service{}, ErrVersion
	}
	if _, err := tx.Exec(ctx, `DELETE FROM routes WHERE service_id=$1`, id); err != nil {
		return model.Service{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM upstream_pools WHERE service_id=$1`, id); err != nil {
		return model.Service{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM hostnames WHERE service_id=$1`, id); err != nil {
		return model.Service{}, err
	}
	if err := insertGraph(ctx, tx, id, current.FleetID, normalized); err != nil {
		if isUnique(err) {
			return model.Service{}, ErrConflict
		}
		return model.Service{}, err
	}
	if err := insertAudit(ctx, tx, actor, "service.update", "service:"+id, map[string]any{"version": current.Version}, map[string]any{"name": normalized.Name}, "ok"); err != nil {
		return model.Service{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Service{}, err
	}
	return s.GetService(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id, actor string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var status, name string
	err = tx.QueryRow(ctx, `SELECT status, name FROM services WHERE id=$1 FOR UPDATE`, id).Scan(&status, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != model.StatusDraft {
		return ErrDraftOnly
	}
	if _, err := tx.Exec(ctx, `DELETE FROM services WHERE id=$1`, id); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, actor, "service.delete", "service:"+id, map[string]any{"name": name}, nil, "ok"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Validate(ctx context.Context, id string) (model.Service, []string, error) {
	svc, err := s.GetService(ctx, id)
	if err != nil {
		return model.Service{}, nil, err
	}
	input := inputFromService(svc)
	normalized, errs := validate.Normalize(input)
	if len(errs) > 0 {
		return svc, errs, nil
	}
	existing, err := s.ownedRoutes(ctx, s.pool, svc.FleetID, id)
	if err != nil {
		return model.Service{}, nil, err
	}
	errs = append(errs, validate.Conflicts(id, normalized.Hostnames, normalized.Routes, existing)...)
	return svc, errs, nil
}

func (s *Store) Publish(ctx context.Context, id string, expected int, node, actor, idemKey, requestHash string, signer sign.Signer) (model.Revision, bool, error) {
	node, err := normalizePublishNode(node)
	if err != nil {
		return model.Revision{}, false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Revision{}, false, err
	}
	defer tx.Rollback(ctx)
	if idemKey != "" {
		var storedHash string
		var raw []byte
		err := tx.QueryRow(ctx, `SELECT request_hash, response FROM idempotency_keys WHERE key=$1`, idemKey).Scan(&storedHash, &raw)
		if err == nil {
			if storedHash != requestHash {
				return model.Revision{}, false, ErrConflict
			}
			var revision model.Revision
			if err := json.Unmarshal(raw, &revision); err != nil {
				return model.Revision{}, false, err
			}
			return revision, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return model.Revision{}, false, err
		}
	}
	var fleetID, status string
	var version int
	err = tx.QueryRow(ctx, `SELECT fleet_id::text, status, version FROM services WHERE id=$1 FOR UPDATE`, id).Scan(&fleetID, &status, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Revision{}, false, ErrNotFound
	}
	if err != nil {
		return model.Revision{}, false, err
	}
	var lockedFleet string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM fleets WHERE id=$1 FOR UPDATE`, fleetID).Scan(&lockedFleet); err != nil {
		return model.Revision{}, false, err
	}
	if node != "" {
		var nodeFleet string
		err := tx.QueryRow(ctx, `SELECT fleet_id::text FROM nodes WHERE name=$1 FOR UPDATE`, node).Scan(&nodeFleet)
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Revision{}, false, ErrNotFound
		}
		if err != nil {
			return model.Revision{}, false, err
		}
		if nodeFleet != fleetID {
			return model.Revision{}, false, &validate.Error{Details: []string{"düğüm bu filoda değil"}}
		}
	}
	if expected != 0 && expected != version {
		return model.Revision{}, false, ErrVersion
	}
	svc, err := s.getServiceTx(ctx, tx, id)
	if err != nil {
		return model.Revision{}, false, err
	}
	_, errs, err := s.validateInTx(ctx, tx, svc)
	if err != nil {
		return model.Revision{}, false, err
	}
	if len(errs) > 0 {
		return model.Revision{}, false, &validate.Error{Details: errs}
	}
	services, err := s.servicesForPublish(ctx, tx, fleetID, id)
	if err != nil {
		return model.Revision{}, false, err
	}
	var revisionID int64
	if err := tx.QueryRow(ctx, `SELECT nextval(pg_get_serial_sequence('revisions','id'))`).Scan(&revisionID); err != nil {
		return model.Revision{}, false, err
	}
	var parent *int64
	var parentValue int64
	err = tx.QueryRow(ctx, `SELECT id FROM revisions WHERE fleet_id=$1 ORDER BY id DESC LIMIT 1`, fleetID).Scan(&parentValue)
	if err == nil {
		parent = &parentValue
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return model.Revision{}, false, err
	}
	listeners, err := s.listeners(ctx, tx, fleetID)
	if err != nil {
		return model.Revision{}, false, err
	}
	pkg, summary, err := buildPackage(revisionID, fleetID, services, listeners)
	if err != nil {
		return model.Revision{}, false, err
	}
	payload, err := canonical.Marshal(pkg)
	if err != nil {
		return model.Revision{}, false, err
	}
	envelope, err := signer.Sign(payload)
	if err != nil {
		return model.Revision{}, false, err
	}
	envelope.Revision = revisionID
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO revisions (id, fleet_id, parent_revision_id, content_hash, signature, key_id, payload, summary, created_by, created_at)
		OVERRIDING SYSTEM VALUE
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		revisionID, fleetID, parent, envelope.ContentHash, envelope.Signature, envelope.KeyID, string(payload), summary, actor, now)
	if err != nil {
		return model.Revision{}, false, err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE services SET status='active', version=version+1, published_version=version+1, updated_at=$2
		WHERE id=$1 AND version=$3`, id, now, version)
	if err != nil {
		return model.Revision{}, false, err
	}
	if tag.RowsAffected() != 1 {
		return model.Revision{}, false, ErrVersion
	}
	if err := pinDesired(ctx, tx, fleetID, node, revisionID); err != nil {
		return model.Revision{}, false, err
	}
	revision := model.Revision{
		ID: revisionID, FleetID: fleetID, ParentRevisionID: parent, ContentHash: envelope.ContentHash,
		KeyID: envelope.KeyID, Summary: summary, CreatedBy: actor, CreatedAt: now,
	}
	if err := insertAudit(ctx, tx, actor, "service.publish", "service:"+id, map[string]any{"version": version}, map[string]any{"revision": revisionID, "hash": envelope.ContentHash}, "ok"); err != nil {
		return model.Revision{}, false, err
	}
	if idemKey != "" {
		raw, err := json.Marshal(revision)
		if err != nil {
			return model.Revision{}, false, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO idempotency_keys (key, request_hash, response) VALUES ($1,$2,$3)`, idemKey, requestHash, raw); err != nil {
			if isUnique(err) {
				return model.Revision{}, false, ErrConflict
			}
			return model.Revision{}, false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Revision{}, false, err
	}
	return revision, false, nil
}

func (s *Store) Rollback(ctx context.Context, revisionID int64, actor string, signer sign.Signer) (model.Revision, error) {
	source, err := s.GetRevision(ctx, revisionID)
	if err != nil {
		return model.Revision{}, err
	}
	var pkg model.Package
	if err := json.Unmarshal([]byte(source.Payload), &pkg); err != nil {
		return model.Revision{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return model.Revision{}, err
	}
	defer tx.Rollback(ctx)
	var lockedFleet string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM fleets WHERE id=$1 FOR UPDATE`, source.FleetID).Scan(&lockedFleet); err != nil {
		return model.Revision{}, err
	}
	var head int64
	if err := tx.QueryRow(ctx, `SELECT id FROM revisions WHERE fleet_id=$1 ORDER BY id DESC LIMIT 1`, source.FleetID).Scan(&head); err != nil {
		return model.Revision{}, err
	}
	if head == revisionID {
		return model.Revision{}, ErrCurrent
	}
	var nextID int64
	if err := tx.QueryRow(ctx, `SELECT nextval(pg_get_serial_sequence('revisions','id'))`).Scan(&nextID); err != nil {
		return model.Revision{}, err
	}
	pkg.Revision = nextID
	pkg.IssuedAt = time.Now().UTC().Format(time.RFC3339)
	payload, err := canonical.Marshal(pkg)
	if err != nil {
		return model.Revision{}, err
	}
	envelope, err := signer.Sign(payload)
	if err != nil {
		return model.Revision{}, err
	}
	now := time.Now().UTC()
	summary := fmt.Sprintf("revision %d paketine dönüş", revisionID)
	_, err = tx.Exec(ctx, `
		INSERT INTO revisions (id, fleet_id, parent_revision_id, content_hash, signature, key_id, payload, summary, created_by, created_at)
		OVERRIDING SYSTEM VALUE
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		nextID, source.FleetID, head, envelope.ContentHash, envelope.Signature, envelope.KeyID, string(payload), summary, actor, now)
	if err != nil {
		return model.Revision{}, err
	}
	if err := pinDesired(ctx, tx, source.FleetID, "", nextID); err != nil {
		return model.Revision{}, err
	}
	revision := model.Revision{
		ID: nextID, FleetID: source.FleetID, ParentRevisionID: &head, ContentHash: envelope.ContentHash,
		KeyID: envelope.KeyID, Summary: summary, CreatedBy: actor, CreatedAt: now,
	}
	if err := insertAudit(ctx, tx, actor, "revision.rollback", fmt.Sprintf("revision:%d", revisionID), map[string]any{"from": head, "source": revisionID}, map[string]any{"revision": nextID}, "ok"); err != nil {
		return model.Revision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Revision{}, err
	}
	return revision, nil
}

func (s *Store) ListRevisions(ctx context.Context, limit, offset int) ([]model.Revision, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, fleet_id::text, parent_revision_id, content_hash, key_id, summary, created_by, created_at
		FROM revisions ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Revision{}
	for rows.Next() {
		var item model.Revision
		if err := rows.Scan(&item.ID, &item.FleetID, &item.ParentRevisionID, &item.ContentHash, &item.KeyID, &item.Summary, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) GetRevision(ctx context.Context, id int64) (model.Revision, error) {
	var item model.Revision
	err := s.pool.QueryRow(ctx, `
		SELECT id, fleet_id::text, parent_revision_id, content_hash, key_id, summary, created_by, created_at, payload
		FROM revisions WHERE id=$1`, id).Scan(
		&item.ID, &item.FleetID, &item.ParentRevisionID, &item.ContentHash, &item.KeyID, &item.Summary, &item.CreatedBy, &item.CreatedAt, &item.Payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Revision{}, ErrNotFound
	}
	return item, err
}

func (s *Store) ListAudit(ctx context.Context, limit, offset int) ([]model.AuditEvent, error) {
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, actor, action, target, before_state, after_state, result, created_at
		FROM audit_events ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AuditEvent{}
	for rows.Next() {
		var item model.AuditEvent
		var before, after []byte
		if err := rows.Scan(&item.ID, &item.Actor, &item.Action, &item.Target, &before, &after, &item.Result, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Before = decodeJSON(before)
		item.After = decodeJSON(after)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) DesiredEnvelope(ctx context.Context, nodeName string) (model.Envelope, error) {
	var env model.Envelope
	var payload string
	err := s.pool.QueryRow(ctx, `
		SELECT r.id, r.payload, r.content_hash, r.signature, r.key_id
		FROM nodes n
		JOIN revisions r ON r.id = n.desired_revision
		WHERE n.name = $1`, nodeName).Scan(&env.Revision, &payload, &env.ContentHash, &env.Signature, &env.KeyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Envelope{}, ErrNotFound
	}
	if err != nil {
		return model.Envelope{}, err
	}
	env.Payload = []byte(payload)
	return env, nil
}

func (s *Store) Heartbeat(ctx context.Context, beat HeartbeatInput) error {
	if beat.Probes == nil {
		beat.Probes = []model.Probe{}
	}
	raw, err := json.Marshal(beat.Probes)
	if err != nil {
		return err
	}
	detail := model.NackDetail(beat.NackDetail)
	var nack *int64
	storedDetail := ""
	if !beat.ClearNack && beat.NackRevision != nil {
		nack = beat.NackRevision
		storedDetail = detail
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE nodes
		SET delivered_revision = COALESCE($2, delivered_revision),
		    acked_revision = COALESCE($3, acked_revision),
		    nack_revision = CASE WHEN $4 THEN $5::bigint ELSE nack_revision END,
		    nack_detail = CASE WHEN $4 THEN $6::text ELSE nack_detail END,
		    probe = $7,
		    last_heartbeat_at = now()
		WHERE name = $1`, beat.Node, beat.DeliveredRevision, beat.AckedRevision, beat.ClearNack || beat.NackRevision != nil, nack, storedDetail, raw)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Store) getService(ctx context.Context, q querier, id string) (model.Service, error) {
	return s.getServiceTx(ctx, q, id)
}

func (s *Store) getServiceTx(ctx context.Context, q querier, id string) (model.Service, error) {
	var svc model.Service
	err := q.QueryRow(ctx, `
		SELECT id::text, name, description, owner, importance, environment_id::text, team_id::text,
		       fleet_id::text, listener_id::text, status, version, published_version, created_at, updated_at
		FROM services WHERE id=$1`, id).Scan(
		&svc.ID, &svc.Name, &svc.Description, &svc.Owner, &svc.Importance, &svc.EnvironmentID, &svc.TeamID,
		&svc.FleetID, &svc.ListenerID, &svc.Status, &svc.Version, &svc.PublishedVersion, &svc.CreatedAt, &svc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Service{}, ErrNotFound
	}
	if err != nil {
		return model.Service{}, err
	}
	svc.Hostnames = []string{}
	rows, err := q.Query(ctx, `SELECT hostname FROM hostnames WHERE service_id=$1 ORDER BY hostname`, id)
	if err != nil {
		return model.Service{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var host string
		if err := rows.Scan(&host); err != nil {
			return model.Service{}, err
		}
		svc.Hostnames = append(svc.Hostnames, host)
	}
	if err := rows.Err(); err != nil {
		return model.Service{}, err
	}
	rows.Close()
	svc.Routes = []model.Route{}
	routeRows, err := q.Query(ctx, `
		SELECT r.id::text, r.path, r.match_type, r.methods, p.id::text, p.name, p.algorithm,
		       p.connect_timeout_ms, p.request_timeout_ms
		FROM routes r JOIN upstream_pools p ON p.id = r.upstream_pool_id
		WHERE r.service_id=$1
		ORDER BY r.priority, r.path`, id)
	if err != nil {
		return model.Service{}, err
	}
	defer routeRows.Close()
	for routeRows.Next() {
		var route model.Route
		var methods []string
		if err := routeRows.Scan(&route.ID, &route.Path, &route.MatchType, &methods, &route.Upstream.ID, &route.Upstream.Name, &route.Upstream.Algorithm, &route.Upstream.ConnectTimeoutMs, &route.Upstream.RequestTimeoutMs); err != nil {
			return model.Service{}, err
		}
		if methods == nil {
			methods = []string{}
		}
		route.Methods = methods
		route.Upstream.Endpoints = []model.Endpoint{}
		svc.Routes = append(svc.Routes, route)
	}
	if err := routeRows.Err(); err != nil {
		return model.Service{}, err
	}
	for i := range svc.Routes {
		epRows, err := q.Query(ctx, `SELECT id::text, address, port, weight FROM endpoints WHERE pool_id=$1 ORDER BY address, port`, svc.Routes[i].Upstream.ID)
		if err != nil {
			return model.Service{}, err
		}
		for epRows.Next() {
			var ep model.Endpoint
			if err := epRows.Scan(&ep.ID, &ep.Address, &ep.Port, &ep.Weight); err != nil {
				epRows.Close()
				return model.Service{}, err
			}
			svc.Routes[i].Upstream.Endpoints = append(svc.Routes[i].Upstream.Endpoints, ep)
		}
		err = epRows.Err()
		epRows.Close()
		if err != nil {
			return model.Service{}, err
		}
	}
	return svc, nil
}

func (s *Store) ensureScope(ctx context.Context, tx pgx.Tx, in model.ServiceInput) error {
	var env string
	err := tx.QueryRow(ctx, `SELECT environment_id::text FROM fleets WHERE id=$1`, in.FleetID).Scan(&env)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && env != in.EnvironmentID) {
		return &validate.Error{Details: []string{"fleet bu ortama ait değil"}}
	}
	if err != nil {
		return err
	}
	var exists int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM teams WHERE id=$1`, in.TeamID).Scan(&exists); err != nil {
		return err
	}
	if exists != 1 {
		return &validate.Error{Details: []string{"ekip bulunamadı"}}
	}
	return nil
}

func (s *Store) listener(ctx context.Context, tx pgx.Tx, in model.ServiceInput) (model.Listener, error) {
	var item model.Listener
	query := `SELECT id::text, fleet_id::text, name, protocol, port FROM listeners WHERE fleet_id=$1`
	args := []any{in.FleetID}
	if in.ListenerID != "" {
		query += ` AND id=$2`
		args = append(args, in.ListenerID)
	} else {
		query += ` AND protocol='http'`
	}
	query += ` ORDER BY port LIMIT 1`
	err := tx.QueryRow(ctx, query, args...).Scan(&item.ID, &item.FleetID, &item.Name, &item.Protocol, &item.Port)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Listener{}, &validate.Error{Details: []string{"listener bulunamadı"}}
	}
	return item, err
}

func (s *Store) ownedRoutes(ctx context.Context, q querier, fleetID, skipID string) ([]validate.OwnedRoute, error) {
	rows, err := q.Query(ctx, `
		SELECT s.id::text, h.hostname, r.match_type, r.path, r.methods
		FROM routes r
		JOIN services s ON s.id = r.service_id
		JOIN hostnames h ON h.service_id = s.id
		WHERE s.fleet_id=$1 AND ($2 = '' OR s.id::text <> $2)`, fleetID, skipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []validate.OwnedRoute
	for rows.Next() {
		var item validate.OwnedRoute
		if err := rows.Scan(&item.ServiceID, &item.Hostname, &item.MatchType, &item.Path, &item.Methods); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *Store) validateInTx(ctx context.Context, tx pgx.Tx, svc model.Service) (model.Service, []string, error) {
	input := inputFromService(svc)
	normalized, errs := validate.Normalize(input)
	if len(errs) > 0 {
		return svc, errs, nil
	}
	existing, err := s.ownedRoutes(ctx, tx, svc.FleetID, svc.ID)
	if err != nil {
		return model.Service{}, nil, err
	}
	return svc, validate.Conflicts(svc.ID, normalized.Hostnames, normalized.Routes, existing), nil
}

func (s *Store) servicesForPublish(ctx context.Context, tx pgx.Tx, fleetID, includeID string) ([]model.Service, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text FROM services
		WHERE fleet_id=$1 AND (status='active' OR id::text=$2)
		ORDER BY name`, fleetID, includeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	out := make([]model.Service, 0, len(ids))
	for _, id := range ids {
		svc, err := s.getServiceTx(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, svc)
	}
	return out, nil
}

func insertGraph(ctx context.Context, tx pgx.Tx, serviceID, fleetID string, in model.ServiceInput) error {
	for _, host := range in.Hostnames {
		if _, err := tx.Exec(ctx, `INSERT INTO hostnames (id, service_id, fleet_id, hostname) VALUES ($1,$2,$3,$4)`, newID(), serviceID, fleetID, host); err != nil {
			return err
		}
	}
	for _, route := range in.Routes {
		poolID := newID()
		if _, err := tx.Exec(ctx, `
			INSERT INTO upstream_pools (id, service_id, name, algorithm, connect_timeout_ms, request_timeout_ms)
			VALUES ($1,$2,$3,$4,$5,$6)`,
			poolID, serviceID, route.Upstream.Name, route.Upstream.Algorithm, route.Upstream.ConnectTimeoutMs, route.Upstream.RequestTimeoutMs); err != nil {
			return err
		}
		for _, ep := range route.Upstream.Endpoints {
			if _, err := tx.Exec(ctx, `INSERT INTO endpoints (id, pool_id, address, port, weight) VALUES ($1,$2,$3,$4,$5)`, newID(), poolID, ep.Address, ep.Port, ep.Weight); err != nil {
				return err
			}
		}
		methods := route.Methods
		if methods == nil {
			methods = []string{}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO routes (id, service_id, match_type, path, methods, upstream_pool_id, priority)
			VALUES ($1,$2,$3,$4,$5,$6,$7)`,
			newID(), serviceID, route.MatchType, route.Path, methods, poolID, routePriority(route.MatchType, route.Path)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) listeners(ctx context.Context, tx pgx.Tx, fleetID string) ([]model.Listener, error) {
	rows, err := tx.Query(ctx, `SELECT id::text, fleet_id::text, name, protocol, port FROM listeners WHERE fleet_id=$1 ORDER BY port, protocol`, fleetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []model.Listener{}
	for rows.Next() {
		var item model.Listener
		if err := rows.Scan(&item.ID, &item.FleetID, &item.Name, &item.Protocol, &item.Port); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func buildPackage(revision int64, fleetID string, services []model.Service, listeners []model.Listener) (model.Package, string, error) {
	byID := map[string]model.Listener{}
	for _, listener := range listeners {
		byID[listener.ID] = listener
	}
	grouped := map[string][]model.Service{}
	for _, svc := range services {
		listener, ok := byID[svc.ListenerID]
		if !ok {
			return model.Package{}, "", &validate.Error{Details: []string{"servisin listener'ı bu filoda yok"}}
		}
		if !httpFamily(listener.Protocol) && len(grouped[svc.ListenerID]) == 1 {
			return model.Package{}, "", &validate.Error{Details: []string{listener.Protocol + " dinleyicisinde tek servis olabilir"}}
		}
		grouped[svc.ListenerID] = append(grouped[svc.ListenerID], svc)
	}
	pkg := model.Package{
		Revision: revision, FleetID: fleetID, IssuedAt: time.Now().UTC().Format(time.RFC3339),
		Listeners: []model.PkgListener{}, VirtualHosts: []model.PkgVHost{}, Clusters: []model.PkgCluster{},
	}
	routeCount := 0
	ids := make([]string, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		listener := byID[id]
		group := grouped[id]
		sort.Slice(group, func(i, j int) bool { return group[i].Name < group[j].Name })
		item := model.PkgListener{
			Name: listener.Name, Protocol: listener.Protocol, Port: uint32(listener.Port),
			RouteConfigName: routeConfigName(listener),
		}
		if listener.Protocol == "https" {
			item.TlsSecret = "local-dev"
		}
		for _, svc := range group {
			domains := append([]string(nil), svc.Hostnames...)
			sort.Strings(domains)
			if listener.Protocol == "tls_passthrough" {
				item.ServerNames = domains
			}
			vhost := model.PkgVHost{Name: svc.ID, Listener: listener.Name, Protocol: listener.Protocol, Domains: domains, Routes: []model.PkgRoute{}}
			routes := append([]model.Route(nil), svc.Routes...)
			sort.Slice(routes, func(i, j int) bool {
				pi, pj := routePriority(routes[i].MatchType, routes[i].Path), routePriority(routes[j].MatchType, routes[j].Path)
				if pi != pj {
					return pi < pj
				}
				return routes[i].Path < routes[j].Path
			})
			for _, route := range routes {
				methods := append([]string(nil), route.Methods...)
				if methods == nil {
					methods = []string{}
				}
				timeout := route.Upstream.RequestTimeoutMs
				if listener.Protocol == "sse" {
					timeout = 0
				}
				pkgRoute := model.PkgRoute{
					Name: route.ID, MatchType: route.MatchType, Path: route.Path, Methods: methods,
					Cluster: route.Upstream.ID, Priority: routePriority(route.MatchType, route.Path),
					TimeoutMs: timeout, Grpc: listener.Protocol == "grpc",
				}
				vhost.Routes = append(vhost.Routes, pkgRoute)
				if item.Cluster == "" {
					item.Cluster = route.Upstream.ID
				}
				endpoints := make([]model.PkgEndpoint, 0, len(route.Upstream.Endpoints))
				for _, ep := range route.Upstream.Endpoints {
					endpoints = append(endpoints, model.PkgEndpoint{Address: ep.Address, Port: uint32(ep.Port), Weight: uint32(ep.Weight)})
				}
				pkg.Clusters = append(pkg.Clusters, model.PkgCluster{
					Name: route.Upstream.ID, Algorithm: route.Upstream.Algorithm,
					ConnectTimeoutMs: route.Upstream.ConnectTimeoutMs, Endpoints: endpoints,
					UpstreamHTTP2: listener.Protocol == "grpc",
					Health:        clusterHealth(listener.Protocol),
				})
				routeCount++
			}
			if httpFamily(listener.Protocol) {
				pkg.VirtualHosts = append(pkg.VirtualHosts, vhost)
			}
		}
		pkg.Listeners = append(pkg.Listeners, item)
	}
	sort.Slice(pkg.Clusters, func(i, j int) bool { return pkg.Clusters[i].Name < pkg.Clusters[j].Name })
	return pkg, fmt.Sprintf("%d servis, %d rota", len(services), routeCount), nil
}

func httpFamily(protocol string) bool {
	switch protocol {
	case "http", "https", "http2", "websocket", "sse", "grpc":
		return true
	default:
		return false
	}
}

func routeConfigName(listener model.Listener) string {
	if listener.Protocol == "http" {
		return "edge"
	}
	return "routes-" + listener.Protocol
}

func clusterHealth(protocol string) string {
	switch protocol {
	case "tcp", "tls_passthrough", "grpc":
		return "tcp"
	case "udp":
		return "none"
	default:
		return "http"
	}
}

func inputFromService(svc model.Service) model.ServiceInput {
	in := model.ServiceInput{
		Name: svc.Name, Description: svc.Description, Owner: svc.Owner, Importance: svc.Importance,
		EnvironmentID: svc.EnvironmentID, TeamID: svc.TeamID, FleetID: svc.FleetID, ListenerID: svc.ListenerID,
		Hostnames: append([]string(nil), svc.Hostnames...),
	}
	for _, route := range svc.Routes {
		endpoints := make([]model.EndpointInput, 0, len(route.Upstream.Endpoints))
		for _, ep := range route.Upstream.Endpoints {
			endpoints = append(endpoints, model.EndpointInput{Address: ep.Address, Port: ep.Port, Weight: ep.Weight})
		}
		in.Routes = append(in.Routes, model.RouteInput{
			Path: route.Path, MatchType: route.MatchType, Methods: append([]string(nil), route.Methods...),
			Upstream: model.UpstreamInput{
				Name: route.Upstream.Name, Algorithm: route.Upstream.Algorithm,
				ConnectTimeoutMs: route.Upstream.ConnectTimeoutMs, RequestTimeoutMs: route.Upstream.RequestTimeoutMs,
				Endpoints: endpoints,
			},
		})
	}
	return in
}

func insertAudit(ctx context.Context, tx pgx.Tx, actor, action, target string, before, after any, result string) error {
	beforeRaw, err := marshalAudit(before)
	if err != nil {
		return err
	}
	afterRaw, err := marshalAudit(after)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (actor, action, target, before_state, after_state, result)
		VALUES ($1,$2,$3,$4,$5,$6)`, actor, action, target, beforeRaw, afterRaw, result)
	return err
}

func marshalAudit(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func decodeJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

func routePriority(matchType, path string) int {
	if matchType == model.MatchExact {
		return len(path)
	}
	n := len(path)
	if n > 900 {
		n = 900
	}
	return 1000 - n
}

func likePattern(query string) string {
	query = strings.ReplaceAll(query, `\`, `\\`)
	query = strings.ReplaceAll(query, `%`, `\%`)
	query = strings.ReplaceAll(query, `_`, `\_`)
	return "%" + query + "%"
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func scanNode(rows pgx.Rows, now time.Time) (model.Node, error) {
	var node model.Node
	var probe []byte
	if err := rows.Scan(&node.ID, &node.FleetID, &node.Name, &node.Location, &node.DesiredRevision, &node.DeliveredRevision, &node.AckedRevision, &node.NackRevision, &node.NackDetail, &node.LastHeartbeatAt, &probe); err != nil {
		return model.Node{}, err
	}
	node.Heartbeat = "unknown"
	if node.LastHeartbeatAt != nil {
		if now.Sub(node.LastHeartbeatAt.UTC()) > 20*time.Second {
			node.Heartbeat = "stale"
		} else {
			node.Heartbeat = "fresh"
		}
	}
	node.TrafficHealth = "unknown"
	node.Probes = []model.Probe{}
	if len(probe) > 0 {
		_ = json.Unmarshal(probe, &node.Probes)
	}
	if node.Probes == nil {
		node.Probes = []model.Probe{}
	}
	return node, nil
}

func RequestHash(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
