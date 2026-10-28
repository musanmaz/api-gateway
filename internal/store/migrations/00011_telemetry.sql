-- +goose Up
ALTER TABLE nodes ADD COLUMN telemetry_checks bigint NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN telemetry_denied bigint NOT NULL DEFAULT 0;
-- +goose Down
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_denied;
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_checks;
