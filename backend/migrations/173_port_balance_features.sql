-- Preserve and provision the balance feature tables from feat/balance-transfer.
-- Every statement is additive so this migration is safe for existing installations.

CREATE TABLE IF NOT EXISTS checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL,
    streak_days INTEGER NOT NULL DEFAULT 1,
    checkin_type VARCHAR(20) NOT NULL DEFAULT 'normal',
    bet_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    multiplier DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, checkin_date)
);

ALTER TABLE checkins ADD COLUMN IF NOT EXISTS checkin_type VARCHAR(20) NOT NULL DEFAULT 'normal';
ALTER TABLE checkins ADD COLUMN IF NOT EXISTS bet_amount DECIMAL(20,8) NOT NULL DEFAULT 0;
ALTER TABLE checkins ADD COLUMN IF NOT EXISTS multiplier DECIMAL(20,8) NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_checkins_user_id ON checkins(user_id);

CREATE TABLE IF NOT EXISTS checkin_prize_items (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    rarity VARCHAR(20) NOT NULL DEFAULT 'common',
    reward_type VARCHAR(30) NOT NULL DEFAULT 'balance',
    reward_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    reward_value_max DECIMAL(20,8) NOT NULL DEFAULT 0,
    subscription_id BIGINT,
    subscription_days INT NOT NULL DEFAULT 0,
    weight INT NOT NULL DEFAULT 100,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_checkin_prize_items_enabled ON checkin_prize_items(is_enabled);

CREATE TABLE IF NOT EXISTS checkin_blindbox_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    prize_item_id BIGINT NOT NULL,
    prize_name VARCHAR(100) NOT NULL,
    rarity VARCHAR(20) NOT NULL,
    reward_type VARCHAR(30) NOT NULL,
    reward_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    streak_days INT NOT NULL DEFAULT 0,
    reward_detail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
ALTER TABLE checkin_blindbox_records ADD COLUMN IF NOT EXISTS reward_detail TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_blindbox_records_user_id ON checkin_blindbox_records(user_id);
CREATE INDEX IF NOT EXISTS idx_blindbox_records_created_at ON checkin_blindbox_records(created_at);

CREATE TABLE IF NOT EXISTS balance_transfers (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,8) NOT NULL,
    fee DECIMAL(20,8) NOT NULL DEFAULT 0,
    fee_rate DECIMAL(10,6) NOT NULL DEFAULT 0,
    gross_amount DECIMAL(20,8) NOT NULL,
    transfer_type VARCHAR(20) NOT NULL DEFAULT 'direct',
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    memo TEXT,
    redpacket_id BIGINT,
    frozen_at TIMESTAMPTZ,
    frozen_by BIGINT,
    revoke_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_sender_id ON balance_transfers(sender_id);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_receiver_id ON balance_transfers(receiver_id);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_status ON balance_transfers(status);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_transfer_type ON balance_transfers(transfer_type);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_created_at ON balance_transfers(created_at);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_sender_created ON balance_transfers(sender_id, created_at);
CREATE INDEX IF NOT EXISTS idx_balance_transfers_receiver_created ON balance_transfers(receiver_id, created_at);

CREATE TABLE IF NOT EXISTS balance_redpackets (
    id BIGSERIAL PRIMARY KEY,
    sender_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_amount DECIMAL(20,8) NOT NULL,
    total_count INT NOT NULL,
    remaining_amount DECIMAL(20,8) NOT NULL,
    remaining_count INT NOT NULL,
    redpacket_type VARCHAR(20) NOT NULL DEFAULT 'equal',
    fee DECIMAL(20,8) NOT NULL DEFAULT 0,
    fee_rate DECIMAL(10,6) NOT NULL DEFAULT 0,
    code VARCHAR(32) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    memo TEXT,
    expire_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_balance_redpackets_sender_id ON balance_redpackets(sender_id);
CREATE INDEX IF NOT EXISTS idx_balance_redpackets_status ON balance_redpackets(status);
CREATE INDEX IF NOT EXISTS idx_balance_redpackets_expire_at ON balance_redpackets(expire_at);

CREATE TABLE IF NOT EXISTS balance_redpacket_claims (
    id BIGSERIAL PRIMARY KEY,
    redpacket_id BIGINT NOT NULL REFERENCES balance_redpackets(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(20,8) NOT NULL,
    transfer_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (redpacket_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_balance_redpacket_claims_user_id ON balance_redpacket_claims(user_id);

ALTER TABLE redeem_codes ADD COLUMN IF NOT EXISTS multiplier DECIMAL(20,8) NOT NULL DEFAULT 0;
ALTER TABLE redeem_codes ADD COLUMN IF NOT EXISTS bet_amount DECIMAL(20,8) NOT NULL DEFAULT 0;

INSERT INTO settings (key, value, updated_at) VALUES
    ('checkin_enabled', 'false', NOW()),
    ('checkin_min_balance', '0.10', NOW()),
    ('checkin_max_balance', '1.00', NOW()),
    ('checkin_luck_enabled', 'false', NOW()),
    ('checkin_luck_min_multiplier', '0.10', NOW()),
    ('checkin_luck_max_multiplier', '3.00', NOW()),
    ('checkin_blindbox_enabled', 'false', NOW()),
    ('checkin_blindbox_trigger_type', 'streak', NOW()),
    ('checkin_blindbox_interval', '7', NOW()),
    ('transfer_enabled', 'false', NOW()),
    ('transfer_fee_rate', '0.010000', NOW()),
    ('transfer_min_amount', '0.01000000', NOW()),
    ('transfer_max_amount', '1000.00000000', NOW()),
    ('transfer_daily_limit', '1000.00000000', NOW()),
    ('transfer_daily_count_limit', '50', NOW()),
    ('transfer_vip_fee_exempt', 'false', NOW()),
    ('redpacket_enabled', 'false', NOW()),
    ('redpacket_max_count', '100', NOW()),
    ('redpacket_expire_hours', '24', NOW())
ON CONFLICT (key) DO NOTHING;
