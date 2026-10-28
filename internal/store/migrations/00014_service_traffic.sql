-- +goose Up
CREATE TABLE service_traffic_minutes (
  node_name text NOT NULL REFERENCES nodes(name) ON DELETE CASCADE,
  boot_id text NOT NULL,
  service_id uuid NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  bucket_start timestamptz NOT NULL,
  metrics jsonb NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (node_name, boot_id, service_id, bucket_start)
);
CREATE INDEX service_traffic_recent ON service_traffic_minutes (service_id, bucket_start DESC);
-- +goose Down
DROP TABLE IF EXISTS service_traffic_minutes;
