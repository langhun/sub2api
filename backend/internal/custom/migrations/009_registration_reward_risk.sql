-- Registration reward eligibility history owned by the Overlay.
-- Fingerprints are one-way hashes of the trusted client IP and a normalized
-- User-Agent. The raw User-Agent is intentionally never persisted.
CREATE TABLE IF NOT EXISTS custom_registration_fingerprint_history (
    fingerprint_hash TEXT PRIMARY KEY,
    first_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    first_source TEXT NOT NULL DEFAULT '',
    first_registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    registration_count BIGINT NOT NULL DEFAULT 1,
    last_registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_fingerprint_hash_check
        CHECK (length(fingerprint_hash) = 64),
    CONSTRAINT custom_registration_fingerprint_count_check
        CHECK (registration_count > 0)
);

CREATE TABLE IF NOT EXISTS custom_registration_fingerprint_events (
    id BIGSERIAL PRIMARY KEY,
    fingerprint_hash TEXT NOT NULL
        REFERENCES custom_registration_fingerprint_history(fingerprint_hash),
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    source TEXT NOT NULL DEFAULT '',
    is_repeat BOOLEAN NOT NULL,
    reward_blocked BOOLEAN NOT NULL,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_registration_fingerprint_event_hash_check
        CHECK (length(fingerprint_hash) = 64)
);

CREATE INDEX IF NOT EXISTS custom_registration_fingerprint_events_user_idx
    ON custom_registration_fingerprint_events (user_id, reward_blocked, id DESC);

-- A reservation prevents two simultaneous first registrations from both
-- receiving the initial reward. Reservations are short lived and are removed
-- when a registration commits or is abandoned.
CREATE TABLE IF NOT EXISTS custom_registration_fingerprint_reservations (
    fingerprint_hash TEXT PRIMARY KEY,
    reservation_token UUID NOT NULL UNIQUE,
    source TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT custom_registration_fingerprint_reservation_hash_check
        CHECK (length(fingerprint_hash) = 64),
    CONSTRAINT custom_registration_fingerprint_reservation_expiry_check
        CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS custom_registration_fingerprint_reservations_expiry_idx
    ON custom_registration_fingerprint_reservations (expires_at);
