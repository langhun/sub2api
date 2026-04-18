-- Backfill redeem_codes for checkin rewards and registration bonus
-- Checkin data from checkins table, registration bonus = 5 for all regular users

SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

-- ==================== 1. Checkin reward records ====================

INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, created_at) VALUES
('chk-0002-' || encode(gen_random_bytes(4), 'hex'), 'checkin', 16.17, 'used', 1, '2026-04-18 13:20:31', '2026-04-18 13:20:31'),
('chk-0003-' || encode(gen_random_bytes(4), 'hex'), 'checkin', 14.65, 'used', 2, '2026-04-18 21:04:43', '2026-04-18 21:04:43'),
('chk-0004-' || encode(gen_random_bytes(4), 'hex'), 'checkin',  9.58, 'used', 4, '2026-04-18 21:11:09', '2026-04-18 21:11:09'),
('chk-0005-' || encode(gen_random_bytes(4), 'hex'), 'checkin', 16.01, 'used', 5, '2026-04-18 21:12:47', '2026-04-18 21:12:47'),
('chk-0006-' || encode(gen_random_bytes(4), 'hex'), 'checkin',  8.51, 'used', 6, '2026-04-18 21:11:46', '2026-04-18 21:11:46'),
('chk-0007-' || encode(gen_random_bytes(4), 'hex'), 'checkin', 12.75, 'used', 7, '2026-04-18 21:15:03', '2026-04-18 21:15:03')
ON CONFLICT (code) DO NOTHING;

-- ==================== 2. Registration bonus records ====================
-- Users 2-8 all got default balance = 5 at registration
-- First update existing invitation records with value=0, then insert missing ones

UPDATE redeem_codes
SET value = 5.0
WHERE type = 'invitation'
  AND value = 0
  AND used_by IN (2, 3, 4, 5, 6, 7, 8);

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
WHERE u.id IN (2, 3, 4, 5, 6, 7, 8)
  AND NOT EXISTS (
    SELECT 1 FROM redeem_codes rc
    WHERE rc.used_by = u.id AND rc.type = 'invitation'
  );
