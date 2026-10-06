ALTER TABLE custom_route_attempts
    ADD COLUMN IF NOT EXISTS account_speed_ms BIGINT NOT NULL DEFAULT 0;
