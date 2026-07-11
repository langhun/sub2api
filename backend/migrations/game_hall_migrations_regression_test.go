package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration175AddsDedicatedGameHallTablesWithoutDroppingLegacy(t *testing.T) {
	content, err := FS.ReadFile("175_add_game_hall_dedicated_tables.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_wallets")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_wallet_transactions")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_jackpots")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_jackpot_transactions")
	require.Contains(t, sql, "INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)")
	require.Contains(t, sql, "INSERT INTO game_hall_jackpot_transactions (")
	require.NotContains(t, sql, "DROP TABLE")
}

func TestMigration176BackfillsDedicatedGameHallBalancesWithoutDroppingLegacy(t *testing.T) {
	content, err := FS.ReadFile("176_backfill_game_hall_dedicated_balances.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)")
	require.Contains(t, sql, "FROM game_wallets gw")
	require.Contains(t, sql, "FROM game_wallet_transactions gwt")
	require.Contains(t, sql, "FROM game_hall_wallet_transactions ghwt")
	require.Contains(t, sql, "INSERT INTO game_hall_jackpots (code, balance, enabled, created_at, updated_at)")
	require.Contains(t, sql, "FROM game_jackpots gj")
	require.Contains(t, sql, "FROM game_jackpot_transactions gjt")
	require.Contains(t, sql, "FROM game_hall_jackpot_transactions ghjt")
	require.NotContains(t, sql, "DROP TABLE")
}

func TestMigration178AddsImmutableGameHallRounds(t *testing.T) {
	content, err := FS.ReadFile("178_add_game_hall_rounds.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_rounds")
	require.Contains(t, sql, "UNIQUE INDEX IF NOT EXISTS idx_game_hall_rounds_user_idempotency")
	require.Contains(t, sql, "symbols JSONB")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_main_balance_transactions")
	require.NotContains(t, sql, "DROP TABLE")
}

func TestMigration180AddsPerUserGameHallDisableSwitch(t *testing.T) {
	content, err := FS.ReadFile("180_add_user_game_hall_disabled.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER TABLE users")
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS game_hall_disabled BOOLEAN NOT NULL DEFAULT FALSE")
	require.NotContains(t, sql, "DROP TABLE")
}
