ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS auth_latency_ms INT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS routing_latency_ms INT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS upstream_latency_ms INT;
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS response_latency_ms INT;
