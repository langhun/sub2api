-- Additional one-way registration controls owned by the Overlay.
-- Raw IP, e-mail, device values and CAPTCHA payloads are never stored.
CREATE TABLE IF NOT EXISTS custom_registration_challenges (
    id BIGSERIAL PRIMARY KEY,
    challenge_hash TEXT NOT NULL UNIQUE,
    action TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'unknown',
    subject_hash TEXT NOT NULL DEFAULT '',
    network_hash TEXT NOT NULL DEFAULT '',
    device_hash TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_challenge_hash_check CHECK (length(challenge_hash) = 64)
);

CREATE INDEX IF NOT EXISTS custom_registration_challenges_expiry_idx
    ON custom_registration_challenges (expires_at, consumed_at);

CREATE TABLE IF NOT EXISTS custom_registration_quota_reservations (
    id BIGSERIAL PRIMARY KEY,
    quota_hash TEXT NOT NULL,
    quota_kind TEXT NOT NULL,
    reservation_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    committed_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_quota_hash_check CHECK (length(quota_hash) = 64),
    CONSTRAINT custom_registration_quota_kind_check CHECK (quota_kind IN ('ip', 'network', 'email', 'domain', 'device', 'oauth_subject'))
);

CREATE INDEX IF NOT EXISTS custom_registration_quota_reservations_lookup_idx
    ON custom_registration_quota_reservations (quota_kind, quota_hash, expires_at);

CREATE TABLE IF NOT EXISTS custom_registration_trial_cost_reservations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    request_hash TEXT NOT NULL UNIQUE,
    reserved_cost NUMERIC(18, 8) NOT NULL,
    settled_cost NUMERIC(18, 8) NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    settled_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_trial_reserved_cost_check CHECK (reserved_cost > 0),
    CONSTRAINT custom_registration_trial_settled_cost_check CHECK (settled_cost >= 0)
);

CREATE INDEX IF NOT EXISTS custom_registration_trial_cost_user_idx
    ON custom_registration_trial_cost_reservations (user_id, expires_at, settled_at, released_at);
