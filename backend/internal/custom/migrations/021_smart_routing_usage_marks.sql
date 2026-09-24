-- Durable request markers for the usage history UI. The route outcome and
-- decision tables are intentionally short-lived telemetry, so history needs a
-- separate small table that is never pruned with those metrics.
CREATE TABLE IF NOT EXISTS custom_usage_log_routing_marks (
    request_id TEXT NOT NULL,
    api_key_id BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (request_id, api_key_id)
);

CREATE INDEX IF NOT EXISTS custom_usage_log_routing_marks_key_idx
    ON custom_usage_log_routing_marks (api_key_id, created_at DESC);
