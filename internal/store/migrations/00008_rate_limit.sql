-- +goose Up
ALTER TABLE services
 ADD COLUMN rate_limit_rps integer NOT NULL DEFAULT 0
 CHECK (rate_limit_rps BETWEEN 0 AND 10000);

-- +goose Down
ALTER TABLE services DROP COLUMN IF EXISTS rate_limit_rps;
