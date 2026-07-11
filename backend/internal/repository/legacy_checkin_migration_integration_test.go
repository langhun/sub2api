//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLegacyCheckinMigration_IsIdempotentAndPreservesNewRecords(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	schemaName := fmt.Sprintf("legacy_checkins_%d", os.Getpid())

	_, err := tx.ExecContext(ctx, `CREATE SCHEMA `+schemaName)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `SET LOCAL search_path TO `+schemaName)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
CREATE TABLE users (id BIGINT PRIMARY KEY);
CREATE TABLE checkin_records (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    mode VARCHAR(20) NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_before DECIMAL(20,8) NOT NULL DEFAULT 0,
    balance_after DECIMAL(20,8) NOT NULL DEFAULT 0,
    random_value DECIMAL(20,8) NOT NULL DEFAULT 0,
    checkin_date VARCHAR(10) NOT NULL,
    checked_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
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
INSERT INTO checkin_records(
    user_id, mode, reward_amount, balance_before, balance_after, random_value, checkin_date, checked_at
) VALUES
    (101, 'normal', 1.25, 10.00, 11.25, 0.00, '2026-07-01', '2026-07-01 08:00:00+00'),
    (101, 'lucky', -2.50, 5.00, 2.50, -0.50, '2026-07-02', '2026-07-02 08:00:00+00'),
    (101, 'normal', 3.75, 2.50, 6.25, 0.00, '2026-07-04', '2026-07-04 08:00:00+00'),
    (102, 'normal', 9.99, 0.00, 9.99, 0.00, 'not-a-date', '2026-07-04 08:00:00+00'),
    (102, 'normal', 9.99, 0.00, 9.99, 0.00, '2026-02-31', '2026-07-04 08:00:00+00');
INSERT INTO checkins(user_id, checkin_date, reward_amount, streak_days, checkin_type)
VALUES (101, DATE '2026-07-02', 88.00, 42, 'normal');
`)
	require.NoError(t, err)

	migrationSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "177_migrate_legacy_checkin_records.sql"))
	require.NoError(t, err)

	for range 2 {
		_, err = tx.ExecContext(ctx, string(migrationSQL))
		require.NoError(t, err)
	}

	rows, err := tx.QueryContext(ctx, `
SELECT checkin_date, reward_amount, streak_days, checkin_type, bet_amount, multiplier, created_at
FROM checkins WHERE user_id = 101 ORDER BY checkin_date`)
	require.NoError(t, err)
	defer rows.Close()

	type migratedRow struct {
		date       string
		reward     float64
		streak     int
		checkinTyp string
		bet        float64
		multiplier float64
	}
	var got []migratedRow
	for rows.Next() {
		var row migratedRow
		var date, createdAt time.Time
		require.NoError(t, rows.Scan(
			&date, &row.reward, &row.streak, &row.checkinTyp, &row.bet, &row.multiplier, &createdAt,
		))
		row.date = date.Format("2006-01-02")
		got = append(got, row)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []migratedRow{
		{date: "2026-07-01", reward: 1.25, streak: 1, checkinTyp: "normal", bet: 0, multiplier: 0},
		{date: "2026-07-02", reward: 88, streak: 42, checkinTyp: "normal", bet: 0, multiplier: 0},
		{date: "2026-07-04", reward: 3.75, streak: 1, checkinTyp: "normal", bet: 0, multiplier: 0},
	}, got)

	var count int
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM checkins WHERE user_id = 102`).Scan(&count))
	require.Zero(t, count, "invalid legacy dates must not abort or pollute the migration")
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM legacy_checkin_migration_rejects
WHERE user_id = 102 AND checkin_date IN ('not-a-date', '2026-02-31')
`).Scan(&count))
	require.Equal(t, 2, count, "invalid legacy rows must remain detectable after migration")
}

func TestLegacyCheckinMigration_PreservesLuckySemantics(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	schemaName := fmt.Sprintf("legacy_lucky_checkins_%d", os.Getpid())

	_, err := tx.ExecContext(ctx, `CREATE SCHEMA `+schemaName)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `SET LOCAL search_path TO `+schemaName)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `
CREATE TABLE users (id BIGINT PRIMARY KEY);
CREATE TABLE checkin_records (
    id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, mode VARCHAR(20) NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL, balance_before DECIMAL(20,8) NOT NULL,
    balance_after DECIMAL(20,8) NOT NULL, random_value DECIMAL(20,8) NOT NULL,
    checkin_date VARCHAR(10) NOT NULL, checked_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE checkins (
    id BIGSERIAL PRIMARY KEY, user_id BIGINT NOT NULL, checkin_date DATE NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL, streak_days INTEGER NOT NULL,
    checkin_type VARCHAR(20) NOT NULL, bet_amount DECIMAL(20,8) NOT NULL,
    multiplier DECIMAL(20,8) NOT NULL, created_at TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, checkin_date)
);
INSERT INTO users(id) VALUES (201);
INSERT INTO checkin_records(
    user_id, mode, reward_amount, balance_before, balance_after, random_value, checkin_date, checked_at
) VALUES (201, 'lucky', -2.5, 5, 2.5, -0.5, '2026-07-02', NOW());
`)
	require.NoError(t, err)
	migrationSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "177_migrate_legacy_checkin_records.sql"))
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	var typ string
	var bet, multiplier float64
	require.NoError(t, tx.QueryRowContext(ctx, `
SELECT checkin_type, bet_amount, multiplier FROM checkins WHERE user_id = 201
`).Scan(&typ, &bet, &multiplier))
	require.Equal(t, "luck", typ)
	require.InDelta(t, 5, bet, 1e-8)
	require.InDelta(t, 0.5, multiplier, 1e-8)
}
