CREATE TABLE IF NOT EXISTS game_hall_rounds (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    game_type VARCHAR(32) NOT NULL,
    bet_amount DECIMAL(20, 8) NOT NULL,
    payout_amount DECIMAL(20, 8) NOT NULL,
    net_amount DECIMAL(20, 8) NOT NULL,
    multiplier DECIMAL(20, 8) NOT NULL,
    balance_before DECIMAL(20, 8) NOT NULL,
    balance_after DECIMAL(20, 8) NOT NULL,
    jackpot_before DECIMAL(20, 8) NOT NULL,
    jackpot_after DECIMAL(20, 8) NOT NULL,
    outcome VARCHAR(16) NOT NULL,
    symbols JSONB NOT NULL DEFAULT '[]'::jsonb,
    idempotency_key VARCHAR(160) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_game_hall_round_nonnegative CHECK (
        bet_amount > 0 AND payout_amount >= 0 AND balance_before >= 0 AND balance_after >= 0
        AND jackpot_before >= 0 AND jackpot_after >= 0
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_hall_rounds_user_idempotency
    ON game_hall_rounds(user_id, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_game_hall_rounds_user_created_at
    ON game_hall_rounds(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_game_hall_rounds_created_at
    ON game_hall_rounds(created_at DESC);

CREATE TABLE IF NOT EXISTS game_hall_main_balance_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    direction VARCHAR(32) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    balance_before DECIMAL(20, 8) NOT NULL,
    balance_after DECIMAL(20, 8) NOT NULL,
    idempotency_key VARCHAR(160) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_hall_main_balance_user_idempotency
    ON game_hall_main_balance_transactions(user_id, idempotency_key);

CREATE INDEX IF NOT EXISTS idx_game_hall_main_balance_user_created_at
    ON game_hall_main_balance_transactions(user_id, created_at DESC);

COMMENT ON TABLE game_hall_rounds IS '娱乐大厅不可变游戏回合审计记录';
COMMENT ON TABLE game_hall_main_balance_transactions IS '娱乐大厅兑换引起的主余额不可变审计记录';
