UPDATE custom_api_key_route_profiles
SET exploration_rate = CASE
        WHEN exploration_rate < 0 THEN 0
        ELSE 0.01
    END,
    updated_at = NOW()
WHERE exploration_rate < 0 OR exploration_rate > 0.01;

ALTER TABLE custom_api_key_route_profiles
    DROP CONSTRAINT IF EXISTS custom_api_key_route_profiles_exploration_rate_check;

ALTER TABLE custom_api_key_route_profiles
    ADD CONSTRAINT custom_api_key_route_profiles_exploration_rate_check
    CHECK (exploration_rate >= 0 AND exploration_rate <= 0.01);

ALTER TABLE custom_api_key_route_profiles
    ALTER COLUMN exploration_rate SET DEFAULT 0.01;
