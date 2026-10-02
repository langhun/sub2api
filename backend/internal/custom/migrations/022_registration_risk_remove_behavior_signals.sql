-- Permission checks for check-in and API-key creation are normal user actions.
-- Remove their old observation-only signals from existing profiles; future
-- requests no longer record them in the registration risk score. Recalculate
-- the action from the remaining evidence so a repeated fingerprint or cluster
-- signal is never silently downgraded by this cleanup.
WITH config AS (
    SELECT
        COALESCE((
            SELECT CASE
                WHEN value ~ '^[0-9]{1,3}$' AND value::INTEGER BETWEEN 1 AND 100
                    THEN value::INTEGER
            END
            FROM settings
            WHERE key = 'registration_cluster_medium_threshold'
        ), 3) AS medium_threshold,
        GREATEST(
            COALESCE((
                SELECT CASE
                    WHEN value ~ '^[0-9]{1,3}$' AND value::INTEGER BETWEEN 1 AND 100
                        THEN value::INTEGER
                END
                FROM settings
                WHERE key = 'registration_cluster_high_threshold'
            ), 5),
            COALESCE((
                SELECT CASE
                    WHEN value ~ '^[0-9]{1,3}$' AND value::INTEGER BETWEEN 1 AND 100
                        THEN value::INTEGER
                END
                FROM settings
                WHERE key = 'registration_cluster_medium_threshold'
            ), 3)
        ) AS high_threshold,
        COALESCE((
            SELECT CASE LOWER(value)
                WHEN 'false' THEN FALSE
                WHEN 'true' THEN TRUE
            END
            FROM settings
            WHERE key = 'registration_auto_high_risk_lock_enabled'
        ), TRUE) AS auto_high_lock
), cleaned AS (
    SELECT
        p.user_id,
        p.admin_override,
        GREATEST(
            0,
            p.risk_score - COALESCE((
                SELECT COUNT(*)
                FROM jsonb_array_elements_text(p.risk_signals) AS signal(value)
                WHERE signal.value IN ('fast_checkin', 'api_key_created')
            ), 0)::INTEGER
        ) AS remaining_score,
        COALESCE((
            SELECT jsonb_agg(signal.value ORDER BY signal.ordinality)
            FROM jsonb_array_elements_text(p.risk_signals) WITH ORDINALITY AS signal(value, ordinality)
            WHERE signal.value NOT IN ('fast_checkin', 'api_key_created')
        ), '[]'::jsonb) AS remaining_signals
    FROM custom_registration_risk_profiles p
    WHERE p.risk_signals ?| ARRAY['fast_checkin', 'api_key_created']
), recalculated AS (
    SELECT
        c.*,
        CASE
            WHEN c.remaining_score >= cfg.high_threshold
                 AND c.remaining_signals ? 'exact_fingerprint_repeat' THEN
                CASE WHEN cfg.auto_high_lock THEN 'high' ELSE 'medium' END
            WHEN c.remaining_score >= cfg.medium_threshold THEN 'medium'
            ELSE 'none'
        END AS recalculated_action
    FROM cleaned c
    CROSS JOIN config cfg
)
UPDATE custom_registration_risk_profiles p
SET risk_score = r.remaining_score,
    risk_signals = r.remaining_signals,
    risk_action = CASE
        WHEN r.admin_override <> '' THEN p.risk_action
        ELSE r.recalculated_action
    END,
    updated_at = NOW()
FROM recalculated r
WHERE p.user_id = r.user_id;

-- The admin list joins events by both sides of the fingerprint relationship.
-- Keep each side indexed without exposing raw fingerprint values.
CREATE INDEX IF NOT EXISTS custom_registration_fingerprint_events_user_repeat_hash_idx
    ON custom_registration_fingerprint_events (user_id, is_repeat, fingerprint_hash);

CREATE INDEX IF NOT EXISTS custom_registration_fingerprint_events_hash_registered_user_idx
    ON custom_registration_fingerprint_events (fingerprint_hash, registered_at, user_id);
