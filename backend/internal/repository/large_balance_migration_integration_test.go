//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLargeBalanceMigration_PersistsLuckyCheckinAuditAmounts(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	schemaName := fmt.Sprintf("large_balance_%d", os.Getpid())

	_, err := tx.ExecContext(ctx, `CREATE SCHEMA `+schemaName)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `SET LOCAL search_path TO `+schemaName)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    balance DECIMAL(20,8) NOT NULL DEFAULT 0,
    total_recharged DECIMAL(20,8) NOT NULL DEFAULT 0
);
CREATE TABLE checkins (
    reward_amount DECIMAL(20,8) NOT NULL,
    bet_amount DECIMAL(20,8) NOT NULL
);
CREATE TABLE redeem_codes (
    value DECIMAL(20,8) NOT NULL,
    bet_amount DECIMAL(20,8) NOT NULL
);
`)
	require.NoError(t, err)

	migrationPath := filepath.Join("..", "..", "migrations", "185_widen_large_balance_amount_columns.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	const largeAmount = "81058106150981.250000000000000000"
	for _, query := range []string{
		`INSERT INTO users(id, balance, total_recharged) VALUES (1, $1, $1)`,
		`INSERT INTO checkins(reward_amount, bet_amount) VALUES ($1, $1)`,
		`INSERT INTO redeem_codes(value, bet_amount) VALUES ($1, $1)`,
	} {
		_, err = tx.ExecContext(ctx, query, largeAmount)
		require.NoError(t, err)
	}

	for _, query := range []string{
		`SELECT balance::text FROM users WHERE id = 1`,
		`SELECT bet_amount::text FROM checkins LIMIT 1`,
		`SELECT bet_amount::text FROM redeem_codes LIMIT 1`,
	} {
		var got string
		require.NoError(t, tx.QueryRowContext(ctx, query).Scan(&got))
		require.Equal(t, largeAmount, got)
	}
}
