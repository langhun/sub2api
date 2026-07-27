-- Keep activity-only check-in wager metadata outside the upstream redeem_codes model.
CREATE TABLE IF NOT EXISTS custom_activity_redeem_metadata (
    redeem_code_id BIGINT PRIMARY KEY REFERENCES redeem_codes(id) ON DELETE CASCADE,
    multiplier DECIMAL(20,8) NOT NULL DEFAULT 0,
    bet_amount DECIMAL(38,18) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Backfill legacy custom values without changing published redeem_codes columns.
INSERT INTO custom_activity_redeem_metadata (
    redeem_code_id, multiplier, bet_amount, created_at, updated_at
)
SELECT id, multiplier, bet_amount, NOW(), NOW()
FROM redeem_codes
WHERE multiplier <> 0 OR bet_amount <> 0
ON CONFLICT (redeem_code_id) DO NOTHING;
