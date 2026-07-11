DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_wallets'
    ) AND EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_hall_wallet_transactions'
    ) THEN
        INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)
        SELECT
            gw.user_id,
            ROUND(
                gw.dg_balance + COALESCE((
                    SELECT SUM(ghwt.balance_after - ghwt.balance_before)
                    FROM game_hall_wallet_transactions ghwt
                    WHERE ghwt.user_id = gw.user_id
                ), 0),
                8
            ),
            COALESCE(gw.created_at::timestamptz, NOW()),
            NOW()
        FROM game_wallets gw
        JOIN users u ON u.id = gw.user_id
        WHERE NOT EXISTS (
            SELECT 1
            FROM game_wallet_transactions gwt
            WHERE gwt.user_id = gw.user_id
        )
        ON CONFLICT (user_id) DO UPDATE
        SET dg_balance = EXCLUDED.dg_balance,
            updated_at = NOW();
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_jackpots'
    ) AND EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_hall_jackpot_transactions'
    ) THEN
        INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)
        SELECT
            gj.code,
            ROUND(
                gj.balance + COALESCE((
                    SELECT SUM(ghjt.balance_after - ghjt.balance_before)
                    FROM game_hall_jackpot_transactions ghjt
                    WHERE ghjt.jackpot_code = gj.code
                ), 0),
                8
            ),
            COALESCE(gj.enabled, TRUE),
            COALESCE(gj.created_at::timestamptz, NOW()),
            NOW()
        FROM game_jackpots gj
        WHERE gj.code = 'game_hall'
          AND NOT EXISTS (
              SELECT 1
              FROM game_jackpot_transactions gjt
              WHERE gjt.jackpot_code = gj.code
          )
        ON CONFLICT (code) DO UPDATE
        SET balance = EXCLUDED.balance,
            enabled = EXCLUDED.enabled,
            updated_at = NOW();
    END IF;
END $$;
