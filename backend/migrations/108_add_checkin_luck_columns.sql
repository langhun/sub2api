-- Add luck check-in columns to checkins table.
-- checkin_type: 'normal' (default) or 'luck'
-- bet_amount: the amount the user bet (only for luck check-in)
-- multiplier: the random multiplier applied (only for luck check-in)

-- Compatibility: this migration is numbered before 130_add_checkins.sql, so
-- fresh databases that have not seen check-in tables yet need the base table
-- here before the luck columns can be added.
CREATE TABLE IF NOT EXISTS checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL,
    streak_days INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, checkin_date)
);

CREATE INDEX IF NOT EXISTS idx_checkins_user_id ON checkins(user_id);

ALTER TABLE checkins ADD COLUMN IF NOT EXISTS checkin_type VARCHAR(20) NOT NULL DEFAULT 'normal';
ALTER TABLE checkins ADD COLUMN IF NOT EXISTS bet_amount DECIMAL(20, 8) NOT NULL DEFAULT 0;
ALTER TABLE checkins ADD COLUMN IF NOT EXISTS multiplier DECIMAL(20, 8) NOT NULL DEFAULT 0;
