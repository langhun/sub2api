DO $$
DECLARE
    wallet_table_exists BOOLEAN;
    wallet_balance_is_numeric BOOLEAN;
    wallet_has_created_at BOOLEAN;
    wallet_has_updated_at BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_wallets'
    ) INTO wallet_table_exists;

    IF wallet_table_exists THEN
        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'game_wallets'
              AND column_name = 'dg_balance'
              AND data_type = 'numeric'
        ) INTO wallet_balance_is_numeric;

        IF NOT wallet_balance_is_numeric THEN
            EXECUTE $wallet$
                ALTER TABLE game_wallets
                ALTER COLUMN dg_balance DROP DEFAULT,
                ALTER COLUMN dg_balance TYPE DECIMAL(20, 8)
                USING CASE
                    WHEN NULLIF(BTRIM(dg_balance::text), '') IS NULL THEN 0
                    WHEN BTRIM(dg_balance::text) ~ '^-?[0-9]+(\.[0-9]+)?$' THEN BTRIM(dg_balance::text)::DECIMAL(20, 8)
                    ELSE 0
                END
            $wallet$;
        END IF;

        EXECUTE 'UPDATE game_wallets SET dg_balance = 0 WHERE dg_balance IS NULL';
        EXECUTE 'ALTER TABLE game_wallets ALTER COLUMN dg_balance SET DEFAULT 0';
        EXECUTE 'ALTER TABLE game_wallets ALTER COLUMN dg_balance SET NOT NULL';

        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'game_wallets'
              AND column_name = 'created_at'
        ) INTO wallet_has_created_at;

        IF wallet_has_created_at THEN
            EXECUTE 'UPDATE game_wallets SET created_at = NOW() WHERE created_at IS NULL';
            EXECUTE 'ALTER TABLE game_wallets ALTER COLUMN created_at SET DEFAULT NOW()';
            EXECUTE 'ALTER TABLE game_wallets ALTER COLUMN created_at SET NOT NULL';
        ELSE
            EXECUTE 'ALTER TABLE game_wallets ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()';
        END IF;

        SELECT EXISTS (
            SELECT 1
            FROM information_schema.columns
            WHERE table_schema = 'public'
              AND table_name = 'game_wallets'
              AND column_name = 'updated_at'
        ) INTO wallet_has_updated_at;

        IF wallet_has_updated_at THEN
            EXECUTE 'UPDATE game_wallets SET updated_at = NOW() WHERE updated_at IS NULL';
            EXECUTE 'ALTER TABLE game_wallets ALTER COLUMN updated_at SET DEFAULT NOW()';
            EXECUTE 'ALTER TABLE game_wallets ALTER COLUMN updated_at SET NOT NULL';
        ELSE
            EXECUTE 'ALTER TABLE game_wallets ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()';
        END IF;
    END IF;
END $$;

DO $$
DECLARE
    jackpot_table_exists BOOLEAN;
    jackpot_has_id BOOLEAN;
    jackpot_has_name BOOLEAN;
    jackpot_has_code BOOLEAN;
    jackpot_has_enabled BOOLEAN;
    jackpot_has_created_at BOOLEAN;
    jackpot_has_updated_at BOOLEAN;
    jackpot_balance_is_numeric BOOLEAN;
    jackpot_has_code_uniqueness BOOLEAN;
BEGIN
    SELECT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_jackpots'
    ) INTO jackpot_table_exists;

    IF NOT jackpot_table_exists THEN
        RETURN;
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'id'
    ) INTO jackpot_has_id;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'name'
    ) INTO jackpot_has_name;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'code'
    ) INTO jackpot_has_code;

    IF NOT jackpot_has_code THEN
        EXECUTE 'ALTER TABLE game_jackpots ADD COLUMN code VARCHAR(32)';
    END IF;

    IF jackpot_has_id THEN
        EXECUTE $jackpot$
            UPDATE game_jackpots
            SET code = LEFT('legacy_migr_' || id::text, 32)
            WHERE code IS NULL OR BTRIM(code) = ''
        $jackpot$;
    ELSIF jackpot_has_name THEN
        EXECUTE $jackpot$
            UPDATE game_jackpots
            SET code = LEFT('legacy_migr_' || SUBSTR(MD5(BTRIM(name)), 1, 20), 32)
            WHERE code IS NULL OR BTRIM(code) = ''
        $jackpot$;
    ELSE
        EXECUTE $jackpot$
            UPDATE game_jackpots
            SET code = LEFT('legacy_migr_' || SUBSTR(MD5(ctid::text), 1, 20), 32)
            WHERE code IS NULL OR BTRIM(code) = ''
        $jackpot$;
    END IF;

    IF jackpot_has_name THEN
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN name DROP NOT NULL';

        EXECUTE $jackpot$
            WITH target AS (
                SELECT ctid
                FROM game_jackpots
                WHERE BTRIM(name) IN ('game_hall', '全局奖池')
                ORDER BY ctid
                LIMIT 1
            )
            UPDATE game_jackpots gj
            SET code = 'game_hall'
            FROM target
            WHERE gj.ctid = target.ctid
              AND NOT EXISTS (
                  SELECT 1
                  FROM game_jackpots existing
                  WHERE existing.code = 'game_hall'
                    AND existing.ctid <> gj.ctid
              )
        $jackpot$;
    END IF;

    EXECUTE $jackpot$
        UPDATE game_jackpots
        SET code = LEFT('legacy_migr_' || SUBSTR(MD5(ctid::text), 1, 20), 32)
        WHERE code IS NULL OR BTRIM(code) = ''
    $jackpot$;

    EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN code SET NOT NULL';

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'balance'
          AND data_type = 'numeric'
    ) INTO jackpot_balance_is_numeric;

    IF NOT jackpot_balance_is_numeric THEN
        EXECUTE $jackpot$
            ALTER TABLE game_jackpots
            ALTER COLUMN balance DROP DEFAULT,
            ALTER COLUMN balance TYPE DECIMAL(20, 8)
            USING CASE
                WHEN NULLIF(BTRIM(balance::text), '') IS NULL THEN 0
                WHEN BTRIM(balance::text) ~ '^-?[0-9]+(\.[0-9]+)?$' THEN BTRIM(balance::text)::DECIMAL(20, 8)
                ELSE 0
            END
        $jackpot$;
    END IF;

    EXECUTE 'UPDATE game_jackpots SET balance = 0 WHERE balance IS NULL';
    EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN balance SET DEFAULT 0';
    EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN balance SET NOT NULL';

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'enabled'
    ) INTO jackpot_has_enabled;

    IF jackpot_has_enabled THEN
        EXECUTE 'UPDATE game_jackpots SET enabled = TRUE WHERE enabled IS NULL';
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN enabled SET DEFAULT TRUE';
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN enabled SET NOT NULL';
    ELSE
        EXECUTE 'ALTER TABLE game_jackpots ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE';
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'created_at'
    ) INTO jackpot_has_created_at;

    IF jackpot_has_created_at THEN
        EXECUTE 'UPDATE game_jackpots SET created_at = NOW() WHERE created_at IS NULL';
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN created_at SET DEFAULT NOW()';
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN created_at SET NOT NULL';
    ELSE
        EXECUTE 'ALTER TABLE game_jackpots ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()';
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'updated_at'
    ) INTO jackpot_has_updated_at;

    IF jackpot_has_updated_at THEN
        EXECUTE 'UPDATE game_jackpots SET updated_at = NOW() WHERE updated_at IS NULL';
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN updated_at SET DEFAULT NOW()';
        EXECUTE 'ALTER TABLE game_jackpots ALTER COLUMN updated_at SET NOT NULL';
    ELSE
        EXECUTE 'ALTER TABLE game_jackpots ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()';
    END IF;

    SELECT EXISTS (
        SELECT 1
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE n.nspname = 'public'
          AND t.relname = 'game_jackpots'
          AND c.contype IN ('p', 'u')
          AND pg_get_constraintdef(c.oid) LIKE '%(code)%'
    ) OR EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'game_jackpots'
          AND indexdef ILIKE 'CREATE UNIQUE INDEX%'
          AND indexdef LIKE '%(code)%'
    ) INTO jackpot_has_code_uniqueness;

    IF NOT jackpot_has_code_uniqueness THEN
        EXECUTE 'CREATE UNIQUE INDEX idx_game_jackpots_code ON game_jackpots(code)';
    END IF;
END $$;
