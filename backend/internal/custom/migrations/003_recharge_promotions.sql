CREATE TABLE IF NOT EXISTS custom_recharge_promotion_orders (
    id BIGSERIAL PRIMARY KEY,
    payment_order_id BIGINT NOT NULL UNIQUE REFERENCES payment_orders(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode VARCHAR(16) NOT NULL CHECK (mode IN ('percentage', 'tiered')),
    payment_amount NUMERIC(20, 8) NOT NULL CHECK (payment_amount >= 0),
    pay_amount NUMERIC(20, 8) NOT NULL CHECK (pay_amount >= 0),
    base_credit NUMERIC(20, 8) NOT NULL CHECK (base_credit >= 0),
    bonus_credit NUMERIC(20, 8) NOT NULL CHECK (bonus_credit >= 0),
    final_credit NUMERIC(20, 8) NOT NULL CHECK (final_credit >= 0),
    percentage_rate NUMERIC(8, 2) NOT NULL DEFAULT 0 CHECK (percentage_rate >= 0 AND percentage_rate <= 100),
    minimum_amount NUMERIC(20, 8) NOT NULL DEFAULT 0 CHECK (minimum_amount >= 0),
    maximum_bonus NUMERIC(20, 8) NOT NULL DEFAULT 0 CHECK (maximum_bonus >= 0),
    tier_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    campaign_started_at TIMESTAMPTZ,
    campaign_ended_at TIMESTAMPTZ,
    refunded_credit NUMERIC(20, 8) NOT NULL DEFAULT 0 CHECK (refunded_credit >= 0),
    refunded_base_credit NUMERIC(20, 8) NOT NULL DEFAULT 0 CHECK (refunded_base_credit >= 0),
    refunded_bonus_credit NUMERIC(20, 8) NOT NULL DEFAULT 0 CHECK (refunded_bonus_credit >= 0),
    refunded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_recharge_promotion_orders_total_check CHECK (final_credit = base_credit + bonus_credit),
    CONSTRAINT custom_recharge_promotion_orders_refund_check CHECK (refunded_credit <= final_credit),
    CONSTRAINT custom_recharge_promotion_orders_refunded_base_check CHECK (refunded_base_credit <= base_credit),
    CONSTRAINT custom_recharge_promotion_orders_refunded_bonus_check CHECK (refunded_bonus_credit <= bonus_credit),
    CONSTRAINT custom_recharge_promotion_orders_refund_split_check CHECK (refunded_credit = refunded_base_credit + refunded_bonus_credit)
);

CREATE INDEX IF NOT EXISTS custom_recharge_promotion_orders_user_created_idx
    ON custom_recharge_promotion_orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS custom_recharge_promotion_orders_created_idx
    ON custom_recharge_promotion_orders (created_at DESC);
CREATE INDEX IF NOT EXISTS custom_recharge_promotion_orders_mode_created_idx
    ON custom_recharge_promotion_orders (mode, created_at DESC);
