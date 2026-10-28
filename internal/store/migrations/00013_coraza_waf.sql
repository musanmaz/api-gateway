-- +goose Up
ALTER TABLE services ADD COLUMN waf_mode text NOT NULL DEFAULT 'off' CHECK (waf_mode IN ('off','detect','block'));
ALTER TABLE nodes ADD COLUMN telemetry_waf_matches bigint NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN telemetry_waf_blocked bigint NOT NULL DEFAULT 0;
-- +goose Down
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_waf_blocked;
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_waf_matches;
ALTER TABLE services DROP COLUMN IF EXISTS waf_mode;
