-- +goose Up
ALTER TABLE nodes ADD COLUMN telemetry_http_requests bigint NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN telemetry_http_2xx bigint NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN telemetry_http_4xx bigint NOT NULL DEFAULT 0;
ALTER TABLE nodes ADD COLUMN telemetry_http_5xx bigint NOT NULL DEFAULT 0;
-- +goose Down
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_http_5xx;
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_http_4xx;
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_http_2xx;
ALTER TABLE nodes DROP COLUMN IF EXISTS telemetry_http_requests;
