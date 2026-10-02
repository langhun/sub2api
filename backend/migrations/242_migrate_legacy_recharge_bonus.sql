-- Preserve the legacy Overlay table for audit/rollback, while exposing its
-- already-recorded bonus amount through the upstream payment-order contract.
-- This is intentionally data-only and idempotent; legacy settings are kept
-- because fixed-credit tiers cannot be represented losslessly as percentages.
DO $$
BEGIN
    IF to_regclass('public.custom_recharge_promotion_orders') IS NULL
       OR to_regclass('public.payment_orders') IS NULL THEN
        RETURN;
    END IF;

    UPDATE payment_orders AS po
    SET bonus_amount = COALESCE(src.bonus_credit, 0)
    FROM (
        SELECT payment_order_id, MAX(bonus_credit) AS bonus_credit
        FROM custom_recharge_promotion_orders
        GROUP BY payment_order_id
    ) AS src
    WHERE po.id = src.payment_order_id
      AND po.order_type = 'balance'
      AND po.bonus_amount = 0
      AND src.bonus_credit > 0;
END $$;
