-- Backfill redeem_codes for existing checkin records and registration bonus
-- Uses encode(gen_random_bytes(8), 'hex') to generate unique 16-char codes

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

-- 1. Backfill checkin records from checkins table
INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, created_at)
SELECT
    encode(gen_random_bytes(8), 'hex'),
    'checkin',
    c.reward_amount,
    'used',
    c.user_id,
    c.created_at,
    c.created_at
FROM checkins c
WHERE NOT EXISTS (
    SELECT 1 FROM redeem_codes rc
    WHERE rc.type = 'checkin'
      AND rc.used_by = c.user_id
      AND ABS(rc.value - c.reward_amount) < 0.001
      AND DATE(rc.used_at) = DATE(c.checkin_date)
);

-- 2. Backfill registration bonus for users who have positive total_recharged
--    but no invitation-type redeem_code record (default balance = 5)
--    Only for users where total_recharged >= 5 (indicating they received the bonus)
INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, created_at)
SELECT
    'reg-' || u.id || '-' || encode(gen_random_bytes(4), 'hex'),
    'invitation',
    5.0,
    'used',
    u.id,
    u.created_at,
    u.created_at
FROM users u
WHERE u.total_recharged >= 5
  AND NOT EXISTS (
    SELECT 1 FROM redeem_codes rc
    WHERE rc.used_by = u.id
      AND rc.type IN ('invitation', 'balance', 'admin_balance')
      AND ABS(rc.value - 5.0) < 0.001
  )
  AND NOT EXISTS (
    SELECT 1 FROM redeem_codes rc
    WHERE rc.used_by = u.id
      AND rc.type = 'invitation'
  );
