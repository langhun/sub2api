-- Restore the upstream Ops vNext tables when migration history records 033
-- but the physical core tables are absent. This migration is intentionally
-- additive: never modify the historical 033 migration or remove data.

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS ops_error_logs (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(64), client_request_id VARCHAR(64), user_id BIGINT,
    api_key_id BIGINT, account_id BIGINT, group_id BIGINT, client_ip INET,
    platform VARCHAR(32), model VARCHAR(100), request_path VARCHAR(256),
    stream BOOLEAN NOT NULL DEFAULT false, user_agent TEXT,
    error_phase VARCHAR(32) NOT NULL, error_type VARCHAR(64) NOT NULL,
    severity VARCHAR(8) NOT NULL DEFAULT 'P2', status_code INT,
    is_business_limited BOOLEAN NOT NULL DEFAULT false,
    error_message TEXT, error_body TEXT, error_source VARCHAR(64),
    error_owner VARCHAR(32), account_status VARCHAR(50),
    upstream_status_code INT, upstream_error_message TEXT,
    upstream_error_detail TEXT, provider_error_code VARCHAR(64),
    provider_error_type VARCHAR(64), network_error_type VARCHAR(50),
    retry_after_seconds INT, duration_ms INT, time_to_first_token_ms BIGINT,
    auth_latency_ms BIGINT, routing_latency_ms BIGINT,
    upstream_latency_ms BIGINT, response_latency_ms BIGINT,
    upstream_errors JSONB, is_count_tokens BOOLEAN NOT NULL DEFAULT false,
    resolved BOOLEAN NOT NULL DEFAULT false, resolved_at TIMESTAMPTZ,
    resolved_by_user_id BIGINT, inbound_endpoint VARCHAR(256),
    upstream_endpoint VARCHAR(256), requested_model VARCHAR(100),
    upstream_model VARCHAR(100), request_type SMALLINT,
    attempted_key_prefix VARCHAR(32), deleted_key_owner_user_id BIGINT,
    deleted_key_name VARCHAR(100), api_key_prefix VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ops_system_metrics (
    id BIGSERIAL PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    window_minutes INT NOT NULL DEFAULT 1, platform VARCHAR(32), group_id BIGINT,
    success_count BIGINT NOT NULL DEFAULT 0, error_count_total BIGINT NOT NULL DEFAULT 0,
    business_limited_count BIGINT NOT NULL DEFAULT 0, error_count_sla BIGINT NOT NULL DEFAULT 0,
    upstream_error_count_excl_429_529 BIGINT NOT NULL DEFAULT 0,
    upstream_429_count BIGINT NOT NULL DEFAULT 0, upstream_529_count BIGINT NOT NULL DEFAULT 0,
    token_consumed BIGINT NOT NULL DEFAULT 0, qps DOUBLE PRECISION, tps DOUBLE PRECISION,
    duration_p50_ms INT, duration_p90_ms INT, duration_p95_ms INT, duration_p99_ms INT,
    duration_avg_ms DOUBLE PRECISION, duration_max_ms INT,
    ttft_p50_ms INT, ttft_p90_ms INT, ttft_p95_ms INT, ttft_p99_ms INT,
    ttft_avg_ms DOUBLE PRECISION, ttft_max_ms INT,
    cpu_usage_percent DOUBLE PRECISION, memory_used_mb BIGINT, memory_total_mb BIGINT,
    memory_usage_percent DOUBLE PRECISION, db_ok BOOLEAN, redis_ok BOOLEAN,
    db_conn_active INT, db_conn_idle INT, db_conn_waiting INT, goroutine_count INT,
    concurrency_queue_depth INT, redis_conn_total INT, redis_conn_idle INT,
    account_switch_count BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ops_job_heartbeats (
    job_name VARCHAR(64) PRIMARY KEY, last_run_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ, last_error_at TIMESTAMPTZ, last_error TEXT,
    last_duration_ms BIGINT, last_result TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ops_alert_rules (
    id BIGSERIAL PRIMARY KEY, name VARCHAR(128) NOT NULL, description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true, severity VARCHAR(16) NOT NULL DEFAULT 'warning',
    metric_type VARCHAR(64) NOT NULL, operator VARCHAR(8) NOT NULL,
    threshold DOUBLE PRECISION NOT NULL, window_minutes INT NOT NULL DEFAULT 5,
    sustained_minutes INT NOT NULL DEFAULT 5, cooldown_minutes INT NOT NULL DEFAULT 10,
    filters JSONB, last_triggered_at TIMESTAMPTZ, notify_email BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ops_alert_events (
    id BIGSERIAL PRIMARY KEY, rule_id BIGINT, severity VARCHAR(16) NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'firing', title VARCHAR(200), description TEXT,
    metric_value DOUBLE PRECISION, threshold_value DOUBLE PRECISION, dimensions JSONB,
    fired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), resolved_at TIMESTAMPTZ,
    email_sent BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ops_alert_silences (
    id BIGSERIAL PRIMARY KEY, rule_id BIGINT NOT NULL, platform VARCHAR(64) NOT NULL,
    group_id BIGINT, region VARCHAR(64), until TIMESTAMPTZ NOT NULL, reason TEXT,
    created_by BIGINT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE ops_metrics_hourly ADD COLUMN IF NOT EXISTS ttft_sample_count BIGINT NOT NULL DEFAULT 0;
ALTER TABLE ops_metrics_daily ADD COLUMN IF NOT EXISTS ttft_sample_count BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_ops_error_logs_created_at ON ops_error_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_platform_time ON ops_error_logs (platform, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_group_time ON ops_error_logs (group_id, created_at DESC) WHERE group_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_account_time ON ops_error_logs (account_id, created_at DESC) WHERE account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_status_time ON ops_error_logs (status_code, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_phase_time ON ops_error_logs (error_phase, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_type_time ON ops_error_logs (error_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_request_id ON ops_error_logs (request_id);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_client_request_id ON ops_error_logs (client_request_id);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_is_count_tokens ON ops_error_logs (is_count_tokens) WHERE is_count_tokens = true;
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_resolved_time ON ops_error_logs (resolved, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_unresolved_time ON ops_error_logs (created_at DESC) WHERE resolved = false;
CREATE INDEX IF NOT EXISTS idx_ops_error_logs_user_time ON ops_error_logs (user_id, created_at DESC) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_created_at ON ops_system_metrics (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_window_time ON ops_system_metrics (window_minutes, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_platform_time ON ops_system_metrics (platform, created_at DESC) WHERE platform IS NOT NULL AND platform <> '' AND group_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_ops_system_metrics_group_time ON ops_system_metrics (group_id, created_at DESC) WHERE group_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_ops_alert_rules_name_unique ON ops_alert_rules (name);
CREATE INDEX IF NOT EXISTS idx_ops_alert_rules_enabled ON ops_alert_rules (enabled);
CREATE INDEX IF NOT EXISTS idx_ops_alert_events_rule_status ON ops_alert_events (rule_id, status);
CREATE INDEX IF NOT EXISTS idx_ops_alert_events_fired_at ON ops_alert_events (fired_at DESC);
CREATE INDEX IF NOT EXISTS idx_ops_alert_silences_lookup ON ops_alert_silences (rule_id, platform, group_id, region, until);
