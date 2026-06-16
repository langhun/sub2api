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

func TestMigration148BackfillsDedicatedGameHallBalances(t *testing.T) {
	content, err := FS.ReadFile("148_backfill_game_hall_dedicated_balances.sql")
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
}

func TestMigration156BackfillsUsersBalanceFromRetiredBank(t *testing.T) {
	content, err := FS.ReadFile("156_backfill_users_balance_from_retired_bank.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "SELECT to_regclass('public.users_bank_account') IS NOT NULL")
	require.Contains(t, sql, "ALTER TABLE users")
	require.Contains(t, sql, "ALTER COLUMN balance TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "UPDATE users u")
	require.Contains(t, sql, "SET balance = uba.balance")
	require.Contains(t, sql, "FROM users_bank_account uba")
	require.Contains(t, sql, "WHERE uba.user_id = u.id")
	require.Contains(t, sql, "u.deleted_at IS NULL")
}

func TestMigration157DropsObsoleteBankAndLegacyGameHallTables(t *testing.T) {
	content, err := FS.ReadFile("157_drop_obsolete_bank_and_legacy_gamehall_tables.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "DROP TABLE IF EXISTS exchange_orders")
	require.Contains(t, sql, "DROP TABLE IF EXISTS users_bank_account")
	require.Contains(t, sql, "DROP VIEW IF EXISTS ledger_transaction_balances")
	require.Contains(t, sql, "DROP TABLE IF EXISTS ledger_entries")
	require.Contains(t, sql, "DROP TABLE IF EXISTS game_wallet_transactions")
	require.Contains(t, sql, "DROP TABLE IF EXISTS game_wallets")
	require.Contains(t, sql, "DROP TABLE IF EXISTS game_jackpot_transactions")
	require.Contains(t, sql, "DROP TABLE IF EXISTS game_ledger")
	require.Contains(t, sql, "DROP TABLE IF EXISTS game_jackpots")
	require.Contains(t, sql, "DROP TABLE IF EXISTS deleted_api_key_audits")
	require.Contains(t, sql, "DROP SCHEMA IF EXISTS finance_v2_20260605095822 CASCADE")
	require.Contains(t, sql, "DROP SCHEMA IF EXISTS finance_v2_20260605095901 CASCADE")
}

func TestMigration158WidensCheckinAndRedeemCodeAmounts(t *testing.T) {
	content, err := FS.ReadFile("158_widen_checkin_and_redeem_amount_columns.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ALTER TABLE users")
	require.Contains(t, sql, "ALTER COLUMN total_recharged TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER TABLE checkins")
	require.Contains(t, sql, "ALTER COLUMN reward_amount TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER COLUMN bet_amount TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER TABLE redeem_codes")
	require.Contains(t, sql, "ALTER COLUMN value TYPE DECIMAL(38, 18)")
	require.Contains(t, sql, "ALTER COLUMN bet_amount TYPE DECIMAL(38, 18)")
}
