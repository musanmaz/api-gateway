-- +goose Up
INSERT INTO nodes (id, fleet_id, name, location) VALUES
    ('00000000-0000-4000-8000-000000000006', '00000000-0000-4000-8000-000000000003', 'gw-local-2', 'local');

-- +goose Down
DELETE FROM nodes WHERE id = '00000000-0000-4000-8000-000000000006';
