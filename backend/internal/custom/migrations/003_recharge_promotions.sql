CREATE TABLE IF NOT EXISTS custom_recharge_promotion_orders (
    id BIGSERIAL PRIMARY KEY,
    payment_order_id BIGINT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payment_amount NUMERIC(20, 8) NOT NULL CHECK (payment_amount >= 0),
    pay_amount NUMERIC(20, 8) NOT NULL CHECK (pay_amount >= 0),
    base_credit NUMERIC(20, 8) NOT NULL CHECK (base_credit >= 0),
    bonus_credit NUMERIC(20, 8) NOT NULL CHECK (bonus_credit >= 0),
    final_credit NUMERIC(20, 8) NOT NULL CHECK (final_credit >= 0),
    bonus_rate NUMERIC(8, 2) NOT NULL CHECK (bonus_rate > 0 AND bonus_rate <= 100),
    min_amount NUMERIC(20, 8) NOT NULL CHECK (min_amount >= 0),
    max_bonus NUMERIC(20, 8) NOT NULL CHECK (max_bonus >= 0),
    campaign_started_at TIMESTAMPTZ,
    campaign_ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_recharge_promotion_orders_total_check
        CHECK (final_credit = base_credit + bonus_credit)
);

CREATE INDEX IF NOT EXISTS custom_recharge_promotion_orders_user_created_idx
    ON custom_recharge_promotion_orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS custom_recharge_promotion_orders_created_idx
    ON custom_recharge_promotion_orders (created_at DESC);
