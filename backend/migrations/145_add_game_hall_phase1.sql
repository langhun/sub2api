CREATE TABLE IF NOT EXISTS game_wallets (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    dg_balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS game_wallet_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tx_type VARCHAR(32) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    balance_before DECIMAL(20, 8) NOT NULL,
    balance_after DECIMAL(20, 8) NOT NULL,
    reference_type VARCHAR(32) NOT NULL DEFAULT '',
    reference_id VARCHAR(128) NOT NULL DEFAULT '',
    idempotency_key VARCHAR(160) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_wallet_transactions_user_idempotency
    ON game_wallet_transactions(user_id, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_game_wallet_transactions_user_created_at
    ON game_wallet_transactions(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS game_jackpots (
    code VARCHAR(32) PRIMARY KEY,
    balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO game_jackpots (code, balance, enabled)
VALUES ('game_hall', 0, TRUE)
ON CONFLICT (code) DO NOTHING;
