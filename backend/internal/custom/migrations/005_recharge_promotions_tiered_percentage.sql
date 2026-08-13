ALTER TABLE custom_recharge_promotion_orders
    ALTER COLUMN mode TYPE VARCHAR(32),
    ADD COLUMN IF NOT EXISTS tiered_percentage_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS matched_minimum_amount NUMERIC(20, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS matched_maximum_amount NUMERIC(20, 8) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS applied_percentage_rate NUMERIC(8, 2) NOT NULL DEFAULT 0;

ALTER TABLE custom_recharge_promotion_orders
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_mode_check,
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_matched_minimum_amount_check,
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_matched_maximum_amount_check,
    DROP CONSTRAINT IF EXISTS custom_recharge_promotion_orders_applied_percentage_rate_check;

ALTER TABLE custom_recharge_promotion_orders
    ADD CONSTRAINT custom_recharge_promotion_orders_mode_check
        CHECK (mode IN ('percentage', 'tiered', 'tiered_percentage')),
    ADD CONSTRAINT custom_recharge_promotion_orders_matched_minimum_amount_check
        CHECK (matched_minimum_amount >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_matched_maximum_amount_check
        CHECK (matched_maximum_amount >= 0),
    ADD CONSTRAINT custom_recharge_promotion_orders_applied_percentage_rate_check
        CHECK (applied_percentage_rate >= 0 AND applied_percentage_rate <= 100);
