-- +goose Up
ALTER TABLE nodes ADD COLUMN nack_revision bigint;
ALTER TABLE nodes ADD COLUMN nack_detail text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE nodes DROP COLUMN IF EXISTS nack_detail;
ALTER TABLE nodes DROP COLUMN IF EXISTS nack_revision;
