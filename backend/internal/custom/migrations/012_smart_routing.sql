CREATE TABLE IF NOT EXISTS custom_api_key_route_profiles (
    api_key_id BIGINT PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    mode VARCHAR(16) NOT NULL DEFAULT 'single',
    strategy VARCHAR(16) NOT NULL DEFAULT 'auto',
    price_weight DOUBLE PRECISION NOT NULL DEFAULT 40,
    speed_weight DOUBLE PRECISION NOT NULL DEFAULT 30,
    success_weight DOUBLE PRECISION NOT NULL DEFAULT 30,
    -- Retained for backwards-compatible reads; cross-group sticky routing is disabled.
    sticky_ttl_seconds INTEGER NOT NULL DEFAULT 0,
    exploration_rate DOUBLE PRECISION NOT NULL DEFAULT 0.05,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_api_key_route_profiles_mode_ck CHECK (mode IN ('single', 'smart')),
    CONSTRAINT custom_api_key_route_profiles_strategy_ck CHECK (strategy IN ('auto', 'price', 'speed', 'success', 'custom')),
    CONSTRAINT custom_api_key_route_profiles_weights_ck CHECK (
        price_weight >= 0 AND speed_weight >= 0 AND success_weight >= 0 AND
        price_weight + speed_weight + success_weight = 100
    )
);

CREATE TABLE IF NOT EXISTS custom_api_key_route_groups (
    profile_id BIGINT NOT NULL REFERENCES custom_api_key_route_profiles(api_key_id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    PRIMARY KEY (profile_id, group_id)
);

CREATE INDEX IF NOT EXISTS custom_api_key_route_groups_group_idx
    ON custom_api_key_route_groups (group_id, enabled);

CREATE TABLE IF NOT EXISTS custom_route_decisions (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(128) NOT NULL,
    api_key_id BIGINT NOT NULL,
    profile_id BIGINT NOT NULL,
    requested_model VARCHAR(200) NOT NULL DEFAULT '',
    endpoint_type VARCHAR(64) NOT NULL DEFAULT '',
    selected_group_id BIGINT,
    selected_account_id BIGINT,
    strategy VARCHAR(16) NOT NULL DEFAULT '',
    score_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    fallback_depth INTEGER NOT NULL DEFAULT 0,
    decision_reason VARCHAR(200) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS custom_route_decisions_request_idx
    ON custom_route_decisions (request_id);
CREATE INDEX IF NOT EXISTS custom_route_decisions_key_created_idx
    ON custom_route_decisions (api_key_id, created_at DESC);
