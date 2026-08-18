-- Cluster-based registration risk state owned by the Overlay.
-- Only one-way hashes and coarse network labels are persisted.
CREATE TABLE IF NOT EXISTS custom_registration_cluster_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    network_hash TEXT NOT NULL,
    user_agent_hash TEXT NOT NULL,
    network_prefix TEXT NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_cluster_hash_check CHECK (length(network_hash) = 64),
    CONSTRAINT custom_registration_cluster_ua_hash_check CHECK (length(user_agent_hash) = 64)
);

CREATE INDEX IF NOT EXISTS custom_registration_events_network_idx
    ON custom_registration_cluster_events (network_hash, registered_at DESC);

CREATE TABLE IF NOT EXISTS custom_registration_risk_profiles (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    risk_score INTEGER NOT NULL DEFAULT 0,
    risk_action TEXT NOT NULL DEFAULT 'none',
    risk_signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    risk_expires_at TIMESTAMPTZ,
    payment_unlocked BOOLEAN NOT NULL DEFAULT FALSE,
    admin_override TEXT NOT NULL DEFAULT '',
    override_reason TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_risk_score_check CHECK (risk_score >= 0),
    CONSTRAINT custom_registration_risk_action_check CHECK (risk_action IN ('none', 'medium', 'high', 'manual_lock', 'manual_unlock'))
);

CREATE INDEX IF NOT EXISTS custom_registration_risk_profiles_action_idx
    ON custom_registration_risk_profiles (risk_action, risk_expires_at);

CREATE TABLE IF NOT EXISTS custom_registration_risk_audit (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    risk_score INTEGER NOT NULL DEFAULT 0,
    signals JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS custom_registration_risk_audit_user_idx
    ON custom_registration_risk_audit (user_id, created_at DESC);
