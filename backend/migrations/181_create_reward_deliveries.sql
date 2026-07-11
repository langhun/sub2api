-- Durable reward delivery outbox for blind-box and future activity rewards.
-- The reward snapshot is frozen when eligibility is created; workers must not
-- re-read mutable prize configuration while delivering it.

CREATE TABLE IF NOT EXISTS reward_deliveries (
    id BIGSERIAL PRIMARY KEY,
    source_type VARCHAR(50) NOT NULL,
    source_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    prize_item_id BIGINT REFERENCES checkin_prize_items(id) ON DELETE SET NULL,
    reward_snapshot JSONB NOT NULL DEFAULT '{}'::JSONB,
    reward_type VARCHAR(30) NOT NULL,
    reward_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    reward_detail TEXT NOT NULL DEFAULT '',
    rule_version VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    next_retry_at TIMESTAMPTZ,
    locked_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    compensated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_reward_deliveries_idempotency_key UNIQUE (idempotency_key),
    CONSTRAINT uq_reward_deliveries_source UNIQUE (source_type, source_id),
    CONSTRAINT chk_reward_deliveries_status CHECK (
        status IN ('pending', 'delivering', 'delivered', 'failed', 'compensated')
    ),
    CONSTRAINT chk_reward_deliveries_source_type CHECK (BTRIM(source_type) <> ''),
    CONSTRAINT chk_reward_deliveries_source_id CHECK (source_id > 0),
    CONSTRAINT chk_reward_deliveries_user_id CHECK (user_id > 0),
    CONSTRAINT chk_reward_deliveries_reward_type CHECK (BTRIM(reward_type) <> ''),
    CONSTRAINT chk_reward_deliveries_reward_value CHECK (reward_value >= 0),
    CONSTRAINT chk_reward_deliveries_rule_version CHECK (BTRIM(rule_version) <> ''),
    CONSTRAINT chk_reward_deliveries_idempotency_key CHECK (BTRIM(idempotency_key) <> ''),
    CONSTRAINT chk_reward_deliveries_attempts CHECK (attempts >= 0),
    CONSTRAINT chk_reward_deliveries_snapshot_object CHECK (
        JSONB_TYPEOF(reward_snapshot) = 'object'
    )
);

CREATE INDEX IF NOT EXISTS idx_reward_deliveries_due
    ON reward_deliveries (COALESCE(next_retry_at, created_at), id)
    WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_reward_deliveries_user
    ON reward_deliveries (user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_reward_deliveries_status
    ON reward_deliveries (status, created_at DESC, id DESC);
