-- Repair databases that applied the original 177 migration before reject
-- auditing and lucky check-in wager metadata were added.
CREATE TABLE IF NOT EXISTS legacy_checkin_migration_rejects (
    legacy_id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    checkin_date TEXT NOT NULL,
    reason TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Invalid calendar dates must be audited without aborting the migration.
CREATE OR REPLACE FUNCTION pg_temp.sub2api_legacy_checkin_date(value TEXT)
RETURNS DATE LANGUAGE plpgsql STRICT AS $parse_date$
BEGIN
    IF value !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
        RETURN NULL;
    END IF;
    RETURN value::date;
EXCEPTION
    WHEN datetime_field_overflow OR invalid_datetime_format THEN
        RETURN NULL;
END
$parse_date$;

DO $$
BEGIN
    IF to_regclass('checkin_records') IS NULL THEN
        RETURN;
    END IF;

    EXECUTE $rejects$
        INSERT INTO legacy_checkin_migration_rejects (
            legacy_id, user_id, checkin_date, reason
        )
        SELECT id, user_id, checkin_date, 'invalid checkin_date; expected YYYY-MM-DD calendar date'
        FROM checkin_records
        WHERE pg_temp.sub2api_legacy_checkin_date(checkin_date) IS NULL
        ON CONFLICT (legacy_id) DO NOTHING
    $rejects$;

    IF to_regclass('checkins') IS NULL THEN
        RETURN;
    END IF;

    EXECUTE $repair$
        WITH repair_candidates AS (
            SELECT
                target.id AS checkin_id,
                min(legacy.balance_before) AS bet_amount,
                min(1 + legacy.random_value) AS multiplier
            FROM checkins AS target
            JOIN checkin_records AS legacy
              ON target.user_id = legacy.user_id
             AND target.checkin_date = pg_temp.sub2api_legacy_checkin_date(legacy.checkin_date)
             AND target.created_at = legacy.checked_at
             AND target.reward_amount = legacy.reward_amount
            WHERE legacy.mode = 'lucky'
              AND target.checkin_type = 'luck'
              AND target.bet_amount = 0
              AND target.multiplier = 0
            GROUP BY target.id
            HAVING count(*) = 1
        )
        UPDATE checkins AS target
        SET
            bet_amount = candidate.bet_amount,
            multiplier = candidate.multiplier
        FROM repair_candidates AS candidate
        WHERE target.id = candidate.checkin_id
    $repair$;
END
$$;
