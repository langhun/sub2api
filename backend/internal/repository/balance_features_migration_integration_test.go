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

func TestBalanceFeaturesMigration_PreservesLegacyBranchData(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	schemaName := fmt.Sprintf("balance_features_%d", os.Getpid())

	_, err := tx.ExecContext(ctx, `CREATE SCHEMA `+schemaName)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `SET LOCAL search_path TO `+schemaName)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
CREATE TABLE users (id BIGINT PRIMARY KEY);
CREATE TABLE redeem_codes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    value DECIMAL(20,8) NOT NULL,
    status VARCHAR(20) NOT NULL
);
CREATE TABLE settings (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL,
    streak_days INTEGER NOT NULL DEFAULT 1,
    checkin_type VARCHAR(20) NOT NULL DEFAULT 'normal',
    bet_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    multiplier DECIMAL(20,8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, checkin_date)
);
INSERT INTO users(id) VALUES (101), (102);
INSERT INTO checkins(user_id, checkin_date, reward_amount, streak_days, checkin_type, bet_amount, multiplier)
VALUES (101, DATE '2026-07-09', -2.50000000, 9, 'luck', 5.00000000, 0.50000000);
INSERT INTO settings(key, value) VALUES ('checkin_enabled', 'true'), ('transfer_enabled', 'true');
`)
	require.NoError(t, err)

	migrationPath := filepath.Join("..", "..", "migrations", "173_port_balance_features.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var (
		reward      float64
		streak      int
		checkinType string
		bet         float64
		multiplier  float64
	)
	err = tx.QueryRowContext(ctx, `
SELECT reward_amount, streak_days, checkin_type, bet_amount, multiplier
FROM checkins WHERE user_id = 101 AND checkin_date = DATE '2026-07-09'
`).Scan(&reward, &streak, &checkinType, &bet, &multiplier)
	require.NoError(t, err)
	require.InDelta(t, -2.5, reward, 1e-8)
	require.Equal(t, 9, streak)
	require.Equal(t, "luck", checkinType)
	require.InDelta(t, 5, bet, 1e-8)
	require.InDelta(t, 0.5, multiplier, 1e-8)

	var enabled string
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = 'checkin_enabled'`).Scan(&enabled))
	require.Equal(t, "true", enabled, "migration defaults must not overwrite production settings")

	for _, table := range []string{
		"checkin_prize_items",
		"checkin_blindbox_records",
		"balance_transfers",
		"balance_redpackets",
		"balance_redpacket_claims",
	} {
		var exists bool
		err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM information_schema.tables
    WHERE table_schema = $1 AND table_name = $2
)
`, schemaName, table).Scan(&exists)
		require.NoError(t, err)
		require.True(t, exists, "expected table %s", table)
	}

	// The migration is intentionally idempotent for installations that already
	// ran the feature-branch migrations outside this repository's sequence.
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)
}
