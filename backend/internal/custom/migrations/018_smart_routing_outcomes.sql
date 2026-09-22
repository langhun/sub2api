CREATE TABLE IF NOT EXISTS custom_route_outcomes (
    request_id TEXT PRIMARY KEY,
    api_key_id BIGINT NOT NULL,
    final_group_id BIGINT,
    final_account_id BIGINT,
    final_success BOOLEAN NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    group_switches INTEGER NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    first_token_ms BIGINT NOT NULL DEFAULT 0,
    error_status INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS custom_route_outcomes_key_created_idx
    ON custom_route_outcomes (api_key_id, created_at DESC);
