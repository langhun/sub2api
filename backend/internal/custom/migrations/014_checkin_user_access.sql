-- Per-user check-in eligibility owned by the custom Overlay. This migration
-- deliberately does not alter upstream user or activity tables.
CREATE TABLE IF NOT EXISTS custom_checkin_user_access (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    reason TEXT NOT NULL DEFAULT 'refund',
    updated_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_custom_checkin_user_access_disabled
    ON custom_checkin_user_access (disabled)
    WHERE disabled = TRUE;
