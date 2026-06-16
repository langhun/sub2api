//go:build integration

package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration156BackfillsUsersBalanceFromRetiredBankTable(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()

	migrationPath := filepath.Join("..", "..", "migrations", "156_backfill_users_balance_from_retired_bank.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
CREATE TABLE users_bank_account (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE,
    balance DECIMAL(38, 18) NOT NULL DEFAULT 0,
    frozen_amount DECIMAL(20, 8) NOT NULL DEFAULT 0,
    credit_limit DECIMAL(20, 8) NOT NULL DEFAULT 0,
    total_debt DECIMAL(20, 8) NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    debt_principal DECIMAL(20, 8) NOT NULL DEFAULT 0,
    debt_interest DECIMAL(20, 8) NOT NULL DEFAULT 0
)
`)
	require.NoError(t, err)

	var userWithBankID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('bank-retire-sync@example.com', 'hash', 'user', 'active', 5, 5)
RETURNING id
`).Scan(&userWithBankID))

	var userWithoutBankID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
INSERT INTO users (email, password_hash, role, status, balance, concurrency)
VALUES ('bank-retire-keep@example.com', 'hash', 'user', 'active', 12.5, 5)
RETURNING id
`).Scan(&userWithoutBankID))

	_, err = tx.ExecContext(ctx, `
INSERT INTO users_bank_account (user_id, balance)
VALUES ($1, 1623185201562.878770799999999894)
`, userWithBankID)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var syncedBalance float64
	err = tx.QueryRowContext(ctx, `
SELECT balance
FROM users
WHERE id = $1
`, userWithBankID).Scan(&syncedBalance)
	require.NoError(t, err)
	require.Equal(t, 1623185201562.8787, syncedBalance)

	var unchangedBalance float64
	err = tx.QueryRowContext(ctx, `
SELECT balance
FROM users
WHERE id = $1
`, userWithoutBankID).Scan(&unchangedBalance)
	require.NoError(t, err)
	require.Equal(t, 12.5, unchangedBalance)
}
