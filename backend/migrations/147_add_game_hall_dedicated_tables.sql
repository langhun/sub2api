CREATE TABLE IF NOT EXISTS game_hall_wallets (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    dg_balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS game_hall_wallet_transactions (
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_hall_wallet_transactions_user_idempotency
    ON game_hall_wallet_transactions(user_id, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_game_hall_wallet_transactions_user_created_at
    ON game_hall_wallet_transactions(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS game_hall_jackpots (
    code VARCHAR(32) PRIMARY KEY,
    balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO game_hall_jackpots (code, balance, enabled)
VALUES ('game_hall', 0, TRUE)
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS game_hall_jackpot_transactions (
    id BIGSERIAL PRIMARY KEY,
    jackpot_code VARCHAR(32) NOT NULL REFERENCES game_hall_jackpots(code) ON DELETE CASCADE,
    tx_type VARCHAR(32) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    balance_before DECIMAL(20, 8) NOT NULL,
    balance_after DECIMAL(20, 8) NOT NULL,
    reference_type VARCHAR(32) NOT NULL DEFAULT '',
    reference_id VARCHAR(128) NOT NULL DEFAULT '',
    user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
    idempotency_key VARCHAR(160) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_hall_jackpot_transactions_code_idempotency
    ON game_hall_jackpot_transactions(jackpot_code, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_game_hall_jackpot_transactions_code_created_at
    ON game_hall_jackpot_transactions(jackpot_code, created_at DESC);

INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
SELECT
    gw.user_id,
    gw.dg_balance,
    COALESCE(gw.created_at::timestamptz, NOW()),
    COALESCE(gw.updated_at::timestamptz, NOW())
FROM game_wallets gw
WHERE EXISTS (
    SELECT 1
    FROM game_wallet_transactions gwt
    WHERE gwt.user_id = gw.user_id
)
ON CONFLICT (user_id) DO UPDATE
SET dg_balance = EXCLUDED.dg_balance,
    updated_at = EXCLUDED.updated_at;

INSERT INTO game_hall_wallet_transactions (
    user_id, tx_type, amount, balance_before, balance_after,
    reference_type, reference_id, idempotency_key, metadata, created_at
)
SELECT
    user_id, tx_type, amount, balance_before, balance_after,
    reference_type, reference_id, idempotency_key, metadata, created_at
FROM game_wallet_transactions
ON CONFLICT (user_id, idempotency_key) DO NOTHING;

INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)
SELECT
    gj.code,
    gj.balance,
    COALESCE(gj.enabled, TRUE),
    COALESCE(gj.created_at::timestamptz, NOW()),
    COALESCE(gj.updated_at::timestamptz, NOW())
FROM game_jackpots gj
WHERE gj.code = 'game_hall'
  AND EXISTS (
      SELECT 1
      FROM game_jackpot_transactions gjt
      WHERE gjt.jackpot_code = gj.code
  )
ON CONFLICT (code) DO UPDATE
SET balance = EXCLUDED.balance,
    enabled = EXCLUDED.enabled,
    updated_at = EXCLUDED.updated_at;

INSERT INTO game_hall_jackpot_transactions (
    jackpot_code, tx_type, amount, balance_before, balance_after,
    reference_type, reference_id, user_id, idempotency_key, metadata, created_at
)
SELECT
    jackpot_code, tx_type, amount, balance_before, balance_after,
    reference_type, reference_id, user_id, idempotency_key, metadata, created_at
FROM game_jackpot_transactions
WHERE jackpot_code = 'game_hall'
ON CONFLICT (jackpot_code, idempotency_key) DO NOTHING;

COMMENT ON TABLE game_hall_wallets IS '娱乐大厅用户 DG 钱包表（独立）';
COMMENT ON TABLE game_hall_wallet_transactions IS '娱乐大厅用户钱包流水表（独立）';
COMMENT ON TABLE game_hall_jackpots IS '娱乐大厅奖池表（独立）';
COMMENT ON TABLE game_hall_jackpot_transactions IS '娱乐大厅奖池流水表（独立）';
