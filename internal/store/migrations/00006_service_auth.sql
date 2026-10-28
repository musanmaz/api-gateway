-- +goose Up
ALTER TABLE services
    ADD COLUMN auth_mode text NOT NULL DEFAULT 'public' CHECK (auth_mode IN ('public','api_key')),
    ADD COLUMN auth_key_hash text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE services DROP COLUMN IF EXISTS auth_key_hash;
ALTER TABLE services DROP COLUMN IF EXISTS auth_mode;
