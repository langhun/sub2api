DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'custom_recharge_promotion_orders'
          AND column_name = 'bonus_rate'
    ) THEN
        ALTER TABLE custom_recharge_promotion_orders RENAME COLUMN bonus_rate TO percentage_rate;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'custom_recharge_promotion_orders'
          AND column_name = 'min_amount'
    ) THEN
        ALTER TABLE custom_recharge_promotion_orders RENAME COLUMN min_amount TO minimum_amount;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'custom_recharge_promotion_orders'
          AND column_name = 'max_bonus'
    ) THEN
        ALTER TABLE custom_recharge_promotion_orders RENAME COLUMN max_bonus TO maximum_bonus;
    END IF;
END $$;

ALTER TABLE custom_recharge_promotion_orders
    ADD COLUMN IF NOT EXISTS mode VARCHAR(16),
    ADD COLUMN IF NOT EXISTS tier_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS refunded_credit NUMERIC(20, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refunded_base_credit NUMERIC(20, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refunded_bonus_credit NUMERIC(20, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ;

UPDATE custom_recharge_promotion_orders
SET mode = 'percentage'
WHERE mode IS NULL;

ALTER TABLE custom_recharge_promotion_orders
    ALTER COLUMN mode SET NOT NULL,
    ALTER COLUMN percentage_rate SET DEFAULT 0,
    ALTER COLUMN minimum_amount SET DEFAULT 0,
    ALTER COLUMN maximum_bonus SET DEFAULT 0,
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_bonus_rate_check,
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_min_amount_check,
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_max_bonus_check;

ALTER TABLE custom_recharge_promotion_orders
    ADD CONSTRAINT custom_recharge_promotion_orders_mode_check
        CHECK (mode IN ('percentage', 'tiered')),
    ADD CONSTRAINT custom_recharge_promotion_orders_percentage_rate_check
        CHECK (percentage_rate >= 0 AND percentage_rate <= 100),
    ADD CONSTRAINT custom_recharge_promotion_orders_minimum_amount_check
        CHECK (minimum_amount >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_maximum_bonus_check
        CHECK (maximum_bonus >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_refunded_credit_check
        CHECK (refunded_credit >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_refunded_base_credit_check
        CHECK (refunded_base_credit >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_refunded_bonus_credit_check
        CHECK (refunded_bonus_credit >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_refund_check
        CHECK (refunded_credit <= final_credit),
    ADD CONSTRAINT custom_recharge_promotion_orders_refunded_base_check
        CHECK (refunded_base_credit <= base_credit),
    ADD CONSTRAINT custom_recharge_promotion_orders_refunded_bonus_check
        CHECK (refunded_bonus_credit <= bonus_credit),
    ADD CONSTRAINT custom_recharge_promotion_orders_refund_split_check
        CHECK (refunded_credit = refunded_base_credit + refunded_bonus_credit);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conrelid = 'custom_recharge_promotion_orders'::regclass
          AND conname = 'custom_recharge_promotion_orders_payment_order_fk'
    ) THEN
        ALTER TABLE custom_recharge_promotion_orders
            ADD CONSTRAINT custom_recharge_promotion_orders_payment_order_fk
            FOREIGN KEY (payment_order_id) REFERENCES payment_orders(id) ON DELETE CASCADE;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS custom_recharge_promotion_orders_mode_created_idx
    ON custom_recharge_promotion_orders (mode, created_at DESC);
