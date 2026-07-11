-- Allow administrators to disable entertainment features for individual users.
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS game_hall_disabled BOOLEAN NOT NULL DEFAULT FALSE;
