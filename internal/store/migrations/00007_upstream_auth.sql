-- +goose Up
ALTER TABLE services
 ADD COLUMN upstream_auth_mode text NOT NULL DEFAULT 'none' CHECK (upstream_auth_mode IN ('none','basic','bearer','header')),
 ADD COLUMN upstream_auth_username text NOT NULL DEFAULT '',
 ADD COLUMN upstream_auth_header text NOT NULL DEFAULT '',
 ADD COLUMN upstream_auth_cipher text NOT NULL DEFAULT '',
 ADD COLUMN required_client_header text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE services
 DROP COLUMN IF EXISTS required_client_header,
 DROP COLUMN IF EXISTS upstream_auth_cipher,
 DROP COLUMN IF EXISTS upstream_auth_header,
 DROP COLUMN IF EXISTS upstream_auth_username,
 DROP COLUMN IF EXISTS upstream_auth_mode;
