-- Immutable audit records for balances granted by registration defaults.
-- Existing accounts are intentionally not backfilled because their original
-- grant configuration cannot be reconstructed reliably.
CREATE TABLE IF NOT EXISTS custom_signup_balance_ledger (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    amount NUMERIC(20, 8) NOT NULL CHECK (amount > 0),
    signup_source VARCHAR(32) NOT NULL DEFAULT 'unknown',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS custom_signup_balance_ledger_user_created_idx
    ON custom_signup_balance_ledger (user_id, created_at DESC);
