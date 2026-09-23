UPDATE custom_api_key_route_profiles
SET exploration_rate = CASE
        WHEN exploration_rate < 0 THEN 0
        ELSE 0.01
    END,
    updated_at = NOW()
WHERE exploration_rate < 0 OR exploration_rate > 0.01;

-- Managed PostgreSQL restores can leave this Overlay table owned by a role
-- different from the application role. The data normalization above only
-- needs UPDATE privilege; owner-only hardening must not prevent startup.
DO $$
BEGIN
    ALTER TABLE custom_api_key_route_profiles
        DROP CONSTRAINT IF EXISTS custom_api_key_route_profiles_exploration_rate_check;

    ALTER TABLE custom_api_key_route_profiles
        ADD CONSTRAINT custom_api_key_route_profiles_exploration_rate_check
        CHECK (exploration_rate >= 0 AND exploration_rate <= 0.01);

    ALTER TABLE custom_api_key_route_profiles
        ALTER COLUMN exploration_rate SET DEFAULT 0.01;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'skipping owner-only smart routing constraint hardening: %', SQLERRM;
END;
$$;
