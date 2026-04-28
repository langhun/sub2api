-- Checkin Blind Box: prize items configured by admin
CREATE TABLE IF NOT EXISTS checkin_prize_items (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    rarity VARCHAR(20) NOT NULL DEFAULT 'common',
    reward_type VARCHAR(30) NOT NULL DEFAULT 'balance',
    reward_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    reward_value_max DECIMAL(20,8) NOT NULL DEFAULT 0,
    subscription_id BIGINT DEFAULT NULL,
    subscription_days INT NOT NULL DEFAULT 0,
    weight INT NOT NULL DEFAULT 100,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Checkin Blind Box: draw records
CREATE TABLE IF NOT EXISTS checkin_blindbox_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    prize_item_id BIGINT NOT NULL,
    prize_name VARCHAR(100) NOT NULL,
    rarity VARCHAR(20) NOT NULL,
    reward_type VARCHAR(30) NOT NULL,
    reward_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    streak_days INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_blindbox_records_user_id ON checkin_blindbox_records(user_id);
CREATE INDEX IF NOT EXISTS idx_blindbox_records_created_at ON checkin_blindbox_records(created_at);
