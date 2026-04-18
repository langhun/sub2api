-- Fix backfill data:
-- 1. Revert invitation codes that were incorrectly changed from value=0 to value=5
-- 2. Convert backfill registration records from type=invitation to type=registration
-- 3. Add missing registration records for users 4, 7, 8

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

-- Revert the invitation codes that were used during registration back to value=0
-- These are actual invitation codes, not registration bonuses
UPDATE redeem_codes
SET value = 0
WHERE type = 'invitation'
  AND value = 5
  AND used_by IN (4, 7, 8);

-- Convert the backfill-created records (users 2,3,5,6) from invitation to registration
UPDATE redeem_codes
SET type = 'registration'
WHERE type = 'invitation'
  AND value = 5
  AND code LIKE 'backfill-reg-%';

-- Add missing registration bonus records for users 4, 7, 8
-- (their invitation codes were reverted above, they need separate registration records)
INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, created_at) VALUES
('backfill-reg-u4', 'registration', 5.0, 'used', 4, '2026-04-18 21:10:50', '2026-04-18 21:10:50'),
('backfill-reg-u7', 'registration', 5.0, 'used', 7, '2026-04-18 21:14:57', '2026-04-18 21:14:57'),
('backfill-reg-u8', 'registration', 5.0, 'used', 8, '2026-04-18 21:19:30', '2026-04-18 21:19:30')
ON CONFLICT (code) DO NOTHING;
