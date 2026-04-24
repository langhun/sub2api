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
