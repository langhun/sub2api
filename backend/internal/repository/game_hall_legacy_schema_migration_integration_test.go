//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestApplyMigrations_LegacyGameHallSchemaCompat(t *testing.T) {
	ctx := context.Background()

	postgresImage := selectDockerImage(ctx, postgresImageTag)
	pgContainer, err := tcpostgres.Run(
		ctx,
		postgresImage,
		tcpostgres.WithDatabase("sub2api_legacy_game_hall"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
	require.NoError(t, err)

	db, err := openSQLWithRetry(ctx, dsn, 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	seedLegacyGameHallSchema(t, db)
	require.NoError(t, ApplyMigrations(ctx, db))

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = tx.Rollback()
	})

	requireColumn(t, tx, "game_wallets", "dg_balance", "numeric", 0, false)
	requireColumn(t, tx, "game_jackpots", "code", "character varying", 32, false)
	requireColumn(t, tx, "game_jackpots", "enabled", "boolean", 0, false)
	requireColumn(t, tx, "game_jackpots", "created_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "game_jackpot_transactions", "jackpot_code", "character varying", 32, false)
	requireColumn(t, tx, "game_wallet_transactions", "metadata", "jsonb", 0, false)
	requireIndex(t, tx, "game_jackpots", "idx_game_jackpots_code")
	requireIndex(t, tx, "game_jackpot_transactions", "idx_game_jackpot_transactions_code_idempotency")

	var (
		code    string
		balance float64
		enabled bool
	)
	err = tx.QueryRowContext(ctx, `
SELECT code, balance, enabled
FROM game_jackpots
WHERE name = '全局奖池'
LIMIT 1
`).Scan(&code, &balance, &enabled)
	require.NoError(t, err)
	require.Equal(t, "game_hall", code)
	require.Equal(t, 12.5, balance)
	require.True(t, enabled)

	var legacyCode string
	err = tx.QueryRowContext(ctx, `
SELECT code
FROM game_jackpots
WHERE name = '老虎机奖池'
LIMIT 1
`).Scan(&legacyCode)
	require.NoError(t, err)
	require.NotEqual(t, "game_hall", legacyCode)
	require.Contains(t, legacyCode, "legacy_migr_")

	_, err = tx.ExecContext(ctx, `
INSERT INTO game_jackpot_transactions (
    jackpot_code, tx_type, amount, balance_before, balance_after, idempotency_key
) VALUES ($1, $2, $3, $4, $5, $6)
`, "game_hall", "legacy_compat", 1.0, 12.5, 13.5, "legacy-compat-1")
	require.NoError(t, err)

	var appliedCount int
	err = tx.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM schema_migrations
WHERE filename IN (
    '144_game_hall_legacy_schema_compat.sql',
    '145_add_game_hall_phase1.sql',
    '146_add_game_jackpot_transactions.sql'
)
`).Scan(&appliedCount)
	require.NoError(t, err)
	require.Equal(t, 3, appliedCount)
}

func seedLegacyGameHallSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(`
CREATE TABLE game_wallets (
    user_id BIGINT PRIMARY KEY,
    dg_balance VARCHAR(255) NOT NULL DEFAULT '0',
    frozen_balance VARCHAR(255) NOT NULL DEFAULT '0',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE game_jackpots (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    balance VARCHAR(255) NOT NULL DEFAULT '0',
    today_in VARCHAR(255) NOT NULL DEFAULT '0',
    today_out VARCHAR(255) NOT NULL DEFAULT '0',
    last_reset_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO game_jackpots (name, balance, today_in, today_out)
VALUES
    ('全局奖池', '12.5', '0', '0'),
    ('老虎机奖池', '2', '0', '0');
`)
	require.NoError(t, err)
}
