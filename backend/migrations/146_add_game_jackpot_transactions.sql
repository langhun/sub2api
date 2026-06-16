DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_jackpots'
    ) AND EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'game_jackpots'
          AND column_name = 'code'
    ) THEN
        EXECUTE $sql$
            CREATE TABLE IF NOT EXISTS game_jackpot_transactions (
                id BIGSERIAL PRIMARY KEY,
                jackpot_code VARCHAR(32) NOT NULL REFERENCES game_jackpots(code) ON DELETE CASCADE,
                tx_type VARCHAR(32) NOT NULL,
                amount DECIMAL(20, 8) NOT NULL,
                balance_before DECIMAL(20, 8) NOT NULL,
                balance_after DECIMAL(20, 8) NOT NULL,
                reference_type VARCHAR(32) NOT NULL DEFAULT '',
                reference_id VARCHAR(128) NOT NULL DEFAULT '',
                user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
                idempotency_key VARCHAR(160) NOT NULL,
                metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
                created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
            )
        $sql$;

        EXECUTE $sql$
            CREATE UNIQUE INDEX IF NOT EXISTS idx_game_jackpot_transactions_code_idempotency
                ON game_jackpot_transactions(jackpot_code, idempotency_key)
        $sql$;

        EXECUTE $sql$
            CREATE INDEX IF NOT EXISTS idx_game_jackpot_transactions_code_created_at
                ON game_jackpot_transactions(jackpot_code, created_at DESC)
        $sql$;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_wallets'
    ) THEN
        EXECUTE $sql$COMMENT ON TABLE game_wallets IS '娱乐大厅用户 DG 钱包表'$sql$;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_wallet_transactions'
    ) THEN
        EXECUTE $sql$COMMENT ON TABLE game_wallet_transactions IS '娱乐大厅用户钱包流水表'$sql$;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_jackpots'
    ) THEN
        EXECUTE $sql$COMMENT ON TABLE game_jackpots IS '娱乐大厅奖池表'$sql$;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'game_jackpot_transactions'
    ) THEN
        EXECUTE $sql$COMMENT ON TABLE game_jackpot_transactions IS '娱乐大厅奖池流水表'$sql$;
    END IF;
END $$;
