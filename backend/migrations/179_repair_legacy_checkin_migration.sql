-- Repair databases that applied the original 177 migration before reject
-- auditing and lucky check-in wager metadata were added.
CREATE TABLE IF NOT EXISTS legacy_checkin_migration_rejects (
    legacy_id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    checkin_date TEXT NOT NULL,
    reason TEXT NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
        WHERE checkin_date !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
           OR (
               checkin_date ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$'
               AND to_char(to_date(checkin_date, 'YYYY-MM-DD'), 'YYYY-MM-DD') <> checkin_date
           )
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
             AND target.checkin_date = CASE
                    WHEN legacy.checkin_date ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
                        CASE
                            WHEN to_char(to_date(legacy.checkin_date, 'YYYY-MM-DD'), 'YYYY-MM-DD') = legacy.checkin_date
                            THEN legacy.checkin_date::date
                        END
                 END
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
