-- Move the game-hall access switch out of the shared users table.
CREATE TABLE IF NOT EXISTS game_hall_user_access (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Preserve every existing per-user decision before removing the legacy column.
INSERT INTO game_hall_user_access (user_id, disabled, created_at, updated_at)
SELECT id, game_hall_disabled, NOW(), NOW()
FROM users
ON CONFLICT (user_id) DO NOTHING;

ALTER TABLE users
    DROP COLUMN IF EXISTS game_hall_disabled;
