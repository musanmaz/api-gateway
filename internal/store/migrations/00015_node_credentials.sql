-- +goose Up
CREATE TABLE node_agent_credentials (
  node_name text PRIMARY KEY REFERENCES nodes(name) ON DELETE CASCADE,
  token_sha256 bytea NOT NULL UNIQUE,
  rotated_at timestamptz NOT NULL DEFAULT now()
);
-- +goose Down
DROP TABLE IF EXISTS node_agent_credentials;
