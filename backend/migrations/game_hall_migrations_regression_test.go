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
