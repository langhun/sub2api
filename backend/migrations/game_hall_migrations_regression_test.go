package migrations

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration146AddsGameJackpotTransactions(t *testing.T) {
	content, err := FS.ReadFile("146_add_game_jackpot_transactions.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_jackpot_transactions")
	require.Contains(t, sql, "jackpot_code VARCHAR(32) NOT NULL REFERENCES game_jackpots(code) ON DELETE CASCADE")
	require.Contains(t, sql, "user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL")
	require.Contains(t, sql, "idempotency_key VARCHAR(160) NOT NULL")
	require.Contains(t, sql, "metadata JSONB NOT NULL DEFAULT '{}'::jsonb")
	require.Contains(t, sql, "CREATE UNIQUE INDEX IF NOT EXISTS idx_game_jackpot_transactions_code_idempotency")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS idx_game_jackpot_transactions_code_created_at")
	require.Contains(t, sql, "COMMENT ON TABLE game_jackpot_transactions IS '娱乐大厅奖池流水表'")
}

func TestMigration144CompatLegacyGameHallSchema(t *testing.T) {
	content, err := FS.ReadFile("144_game_hall_legacy_schema_compat.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER COLUMN dg_balance TYPE DECIMAL(20, 8)")
	require.Contains(t, sql, "ALTER TABLE game_jackpots ADD COLUMN code VARCHAR(32)")
	require.Contains(t, sql, "SET code = 'game_hall'")
	require.Contains(t, sql, "CREATE UNIQUE INDEX idx_game_jackpots_code ON game_jackpots(code)")
}

func TestMigration147AddsDedicatedGameHallTables(t *testing.T) {
	content, err := FS.ReadFile("147_add_game_hall_dedicated_tables.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_wallets")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_wallet_transactions")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_jackpots")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS game_hall_jackpot_transactions")
	require.Contains(t, sql, "INSERT INTO game_hall_wallets (user_id, dg_balance, created_at, updated_at)")
	require.Contains(t, sql, "INSERT INTO game_hall_jackpot_transactions (")
}
