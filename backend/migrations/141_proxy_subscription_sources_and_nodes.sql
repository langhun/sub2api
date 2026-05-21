CREATE TABLE IF NOT EXISTS proxy_subscription_sources (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    url TEXT NOT NULL,
    source_format VARCHAR(32) NOT NULL DEFAULT 'auto',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    refresh_interval_hours INTEGER NOT NULL DEFAULT 6,
    auto_add_to_pool BOOLEAN NOT NULL DEFAULT FALSE,
    last_refreshed_at TIMESTAMPTZ NULL,
    last_success_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    last_node_count INTEGER NOT NULL DEFAULT 0,
    last_materialized_proxy_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS proxy_subscription_nodes (
    id BIGSERIAL PRIMARY KEY,
    source_id BIGINT NOT NULL REFERENCES proxy_subscription_sources(id) ON DELETE CASCADE,
    node_key VARCHAR(256) NOT NULL,
    display_name VARCHAR(255) NULL,
    node_type VARCHAR(32) NOT NULL,
    server VARCHAR(255) NOT NULL,
    port INTEGER NOT NULL,
    config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    landing_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    last_error TEXT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

ALTER TABLE proxies
    ADD COLUMN IF NOT EXISTS subscription_source_id BIGINT NULL REFERENCES proxy_subscription_sources(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS subscription_node_id BIGINT NULL REFERENCES proxy_subscription_nodes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS managed_by_subscription BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_proxy_subscription_sources_enabled ON proxy_subscription_sources(enabled);
CREATE INDEX IF NOT EXISTS idx_proxy_subscription_sources_source_format ON proxy_subscription_sources(source_format);
CREATE INDEX IF NOT EXISTS idx_proxy_subscription_sources_deleted_at ON proxy_subscription_sources(deleted_at);

CREATE INDEX IF NOT EXISTS idx_proxy_subscription_nodes_source_id ON proxy_subscription_nodes(source_id);
CREATE INDEX IF NOT EXISTS idx_proxy_subscription_nodes_landing_status ON proxy_subscription_nodes(landing_status);
CREATE INDEX IF NOT EXISTS idx_proxy_subscription_nodes_deleted_at ON proxy_subscription_nodes(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_proxy_subscription_nodes_source_node_key_active
    ON proxy_subscription_nodes(source_id, node_key)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_proxies_subscription_source_id ON proxies(subscription_source_id);
CREATE INDEX IF NOT EXISTS idx_proxies_subscription_node_id ON proxies(subscription_node_id);
CREATE INDEX IF NOT EXISTS idx_proxies_managed_by_subscription ON proxies(managed_by_subscription);
