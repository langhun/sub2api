-- Migrate the pre-balance-feature check-in history without replacing records
-- already written to the new table. Keep the legacy table for rollback/audit.
CREATE TABLE IF NOT EXISTS legacy_checkin_migration_rejects (
    legacy_id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    checkin_date TEXT NOT NULL,
    reason TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    IF to_regclass('checkin_records') IS NULL OR to_regclass('checkins') IS NULL THEN
        RETURN;
    END IF;

    EXECUTE $rejects$
        INSERT INTO legacy_checkin_migration_rejects (
            legacy_id, user_id, checkin_date, reason
        )
        SELECT id, user_id, checkin_date, 'invalid checkin_date; expected YYYY-MM-DD calendar date'
        FROM checkin_records
        WHERE checkin_date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
           OR (
               checkin_date ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
               AND to_char(to_date(checkin_date, 'YYYY-MM-DD'), 'YYYY-MM-DD') <> checkin_date
           )
        ON CONFLICT (legacy_id) DO NOTHING
    $rejects$;

    EXECUTE $migration$
        WITH valid_legacy AS (
            SELECT
                id,
                user_id,
                checkin_date::date AS checkin_date,
                reward_amount,
                CASE WHEN mode = 'lucky' THEN 'luck' ELSE 'normal' END AS checkin_type,
                CASE WHEN mode = 'lucky' THEN balance_before ELSE 0 END AS bet_amount,
                CASE WHEN mode = 'lucky' THEN 1 + random_value ELSE 0 END AS multiplier,
                checked_at
            FROM checkin_records
            WHERE checkin_date ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
              AND to_char(to_date(checkin_date, 'YYYY-MM-DD'), 'YYYY-MM-DD') = checkin_date
        ),
        dated AS (
            SELECT
                *,
                checkin_date - (row_number() OVER (
                    PARTITION BY user_id ORDER BY checkin_date, id
                ))::int AS streak_group
            FROM valid_legacy
        ),
        migrated AS (
            SELECT
                *,
                row_number() OVER (
                    PARTITION BY user_id, streak_group ORDER BY checkin_date, id
                )::int AS streak_days
            FROM dated
        )
        INSERT INTO checkins (
            user_id,
            checkin_date,
            reward_amount,
            streak_days,
            checkin_type,
            bet_amount,
            multiplier,
            created_at
        )
        SELECT
            user_id,
            checkin_date,
            reward_amount,
            streak_days,
            checkin_type,
            bet_amount,
            multiplier,
            checked_at
        FROM migrated
        ON CONFLICT (user_id, checkin_date) DO NOTHING
    $migration$;
END
$$;
