-- +goose Up
CREATE TABLE environments (
    id uuid PRIMARY KEY,
    name text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE teams (
    id uuid PRIMARY KEY,
    name text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE fleets (
    id uuid PRIMARY KEY,
    environment_id uuid NOT NULL REFERENCES environments (id),
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (environment_id, name)
);

CREATE TABLE nodes (
    id uuid PRIMARY KEY,
    fleet_id uuid NOT NULL REFERENCES fleets (id),
    name text NOT NULL UNIQUE,
    location text NOT NULL DEFAULT '',
    desired_revision bigint,
    delivered_revision bigint,
    acked_revision bigint,
    last_heartbeat_at timestamptz,
    probe jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE listeners (
    id uuid PRIMARY KEY,
    fleet_id uuid NOT NULL REFERENCES fleets (id),
    name text NOT NULL,
    protocol text NOT NULL CHECK (protocol IN ('http')),
    port integer NOT NULL CHECK (port BETWEEN 1 AND 65535),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (fleet_id, protocol, port)
);

CREATE TABLE services (
    id uuid PRIMARY KEY,
    team_id uuid NOT NULL REFERENCES teams (id),
    environment_id uuid NOT NULL REFERENCES environments (id),
    fleet_id uuid NOT NULL REFERENCES fleets (id),
    listener_id uuid NOT NULL REFERENCES listeners (id),
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    owner text NOT NULL,
    importance text NOT NULL CHECK (importance IN ('low', 'medium', 'high')),
    status text NOT NULL CHECK (status IN ('draft', 'active')),
    version integer NOT NULL DEFAULT 1,
    published_version integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (fleet_id, name)
);

CREATE TABLE hostnames (
    id uuid PRIMARY KEY,
    service_id uuid NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    fleet_id uuid NOT NULL REFERENCES fleets (id),
    hostname text NOT NULL,
    UNIQUE (fleet_id, hostname)
);

CREATE TABLE upstream_pools (
    id uuid PRIMARY KEY,
    service_id uuid NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    name text NOT NULL,
    algorithm text NOT NULL CHECK (algorithm IN ('round_robin', 'least_request')),
    connect_timeout_ms integer NOT NULL,
    request_timeout_ms integer NOT NULL
);

CREATE TABLE endpoints (
    id uuid PRIMARY KEY,
    pool_id uuid NOT NULL REFERENCES upstream_pools (id) ON DELETE CASCADE,
    address text NOT NULL,
    port integer NOT NULL,
    weight integer NOT NULL
);

CREATE TABLE routes (
    id uuid PRIMARY KEY,
    service_id uuid NOT NULL REFERENCES services (id) ON DELETE CASCADE,
    match_type text NOT NULL CHECK (match_type IN ('prefix', 'exact')),
    path text NOT NULL,
    methods text[] NOT NULL DEFAULT '{}',
    upstream_pool_id uuid NOT NULL REFERENCES upstream_pools (id) ON DELETE CASCADE,
    priority integer NOT NULL
);

CREATE TABLE revisions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    fleet_id uuid NOT NULL REFERENCES fleets (id),
    parent_revision_id bigint REFERENCES revisions (id),
    content_hash text NOT NULL,
    signature text NOT NULL,
    key_id text NOT NULL,
    payload text NOT NULL,
    summary text NOT NULL,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX revisions_fleet_idx ON revisions (fleet_id, id DESC);

CREATE TABLE idempotency_keys (
    key text PRIMARY KEY,
    request_hash text NOT NULL,
    response jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    actor text NOT NULL,
    action text NOT NULL,
    target text NOT NULL,
    before_state jsonb,
    after_state jsonb,
    result text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS revisions;
DROP TABLE IF EXISTS routes;
DROP TABLE IF EXISTS endpoints;
DROP TABLE IF EXISTS upstream_pools;
DROP TABLE IF EXISTS hostnames;
DROP TABLE IF EXISTS services;
DROP TABLE IF EXISTS listeners;
DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS fleets;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS environments;
