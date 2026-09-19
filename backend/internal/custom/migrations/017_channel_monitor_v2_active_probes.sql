-- V1 active-probe facts projected into the V2 channel status Overlay.
-- The table intentionally contains no API key, user, group, or token usage.
CREATE TABLE IF NOT EXISTS custom_channel_monitor_v2_active_probe_results (
    id BIGSERIAL PRIMARY KEY,
    monitor_id BIGINT NOT NULL REFERENCES channel_monitors(id) ON DELETE CASCADE,
    platform VARCHAR(32) NOT NULL,
    model VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    latency_ms INTEGER,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds BETWEEN 15 AND 3600),
    checked_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_channel_monitor_v2_active_probe_status_check
        CHECK (status IN ('operational', 'degraded', 'failed', 'error')),
    CONSTRAINT custom_channel_monitor_v2_active_probe_result_unique
        UNIQUE (monitor_id, model, checked_at)
);

CREATE INDEX IF NOT EXISTS idx_custom_channel_monitor_v2_active_probe_lookup
    ON custom_channel_monitor_v2_active_probe_results (platform, model, checked_at DESC);

CREATE INDEX IF NOT EXISTS idx_custom_channel_monitor_v2_active_probe_monitor
    ON custom_channel_monitor_v2_active_probe_results (monitor_id, checked_at DESC);

COMMENT ON TABLE custom_channel_monitor_v2_active_probe_results IS
    'Overlay-owned active probe facts merged into channel monitor V2 responses';
