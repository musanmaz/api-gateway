-- +goose Up
ALTER TABLE services ADD COLUMN ip_allowlist text[] NOT NULL DEFAULT ARRAY[]::text[];
ALTER TABLE services ADD COLUMN ip_denylist text[] NOT NULL DEFAULT ARRAY[]::text[];
-- +goose Down
ALTER TABLE services DROP COLUMN IF EXISTS ip_denylist;
ALTER TABLE services DROP COLUMN IF EXISTS ip_allowlist;
