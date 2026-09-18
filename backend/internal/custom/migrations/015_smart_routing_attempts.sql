CREATE TABLE IF NOT EXISTS custom_route_attempts (
    request_id TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    api_key_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    account_id BIGINT NOT NULL,
    requested_model TEXT NOT NULL,
    endpoint_type TEXT NOT NULL,
    stream BOOLEAN NOT NULL,
    success BOOLEAN NOT NULL,
    speed_ms BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (request_id, attempt)
);
CREATE INDEX IF NOT EXISTS custom_route_attempts_metric_idx
    ON custom_route_attempts (group_id, requested_model, endpoint_type, stream, created_at DESC);
CREATE INDEX IF NOT EXISTS custom_route_attempts_retention_idx ON custom_route_attempts (created_at);
