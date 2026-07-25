//go:build integration

package gamehall

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	integrationDB        *sql.DB
	integrationEntClient *dbent.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	if !gameHallDockerAvailable(ctx) {
		if os.Getenv("CI") != "" {
			os.Exit(1)
		}
		os.Exit(0)
	}

	container, err := tcpostgres.Run(
		ctx,
		"postgres:18.1-alpine3.23",
		tcpostgres.WithDatabase("game_hall_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start game-hall postgres container:", err)
		os.Exit(1)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
	if err != nil {
		fmt.Fprintln(os.Stderr, "get game-hall postgres connection string:", err)
		os.Exit(1)
	}
	integrationDB, err = gameHallOpenDB(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open game-hall postgres:", err)
		os.Exit(1)
	}
	defer func() { _ = integrationDB.Close() }()
	if err := createGameHallIntegrationSchema(ctx, integrationDB); err != nil {
		fmt.Fprintln(os.Stderr, "create game-hall integration schema:", err)
		os.Exit(1)
	}

	driver := entsql.OpenDB(dialect.Postgres, integrationDB)
	integrationEntClient = dbent.NewClient(dbent.Driver(driver))
	defer func() { _ = integrationEntClient.Close() }()
	os.Exit(m.Run())
}

func gameHallDockerAvailable(ctx context.Context) bool {
	command := exec.CommandContext(ctx, "docker", "info")
	command.Env = os.Environ()
	return command.Run() == nil
}

func gameHallOpenDB(ctx context.Context, dsn string) (*sql.DB, error) {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = db.PingContext(pingCtx)
			cancel()
		}
		if err == nil {
			return db, nil
		}
		if db != nil {
			_ = db.Close()
		}
		lastErr = err
		time.Sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("postgres not ready after 30s: %w", lastErr)
}

func testEntClient(t *testing.T) *dbent.Client {
	t.Helper()
	return integrationEntClient
}

type gameHallTestUser struct {
	ID           int64
	Email        string
	PasswordHash string
	Balance      float64
}

func mustCreateGameHallTestUser(t *testing.T, _ *dbent.Client, user *gameHallTestUser) *gameHallTestUser {
	t.Helper()
	row := integrationDB.QueryRowContext(context.Background(), `
INSERT INTO users (email, password_hash, balance, updated_at)
VALUES ($1, $2, $3, NOW())
RETURNING id`, user.Email, user.PasswordHash, user.Balance)
	require.NoError(t, row.Scan(&user.ID))
	return user
}

func createGameHallIntegrationSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
    deleted_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE game_hall_wallets (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    dg_balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE game_hall_user_access (
    user_id BIGINT PRIMARY KEY REFERENCES users(id),
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE game_hall_jackpots (
    code VARCHAR(32) PRIMARY KEY,
    balance NUMERIC(20, 8) NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE game_hall_wallet_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    tx_type VARCHAR(32) NOT NULL,
    amount NUMERIC(20, 8) NOT NULL,
    balance_before NUMERIC(20, 8) NOT NULL,
    balance_after NUMERIC(20, 8) NOT NULL,
    reference_type VARCHAR(32) NOT NULL,
    reference_id VARCHAR(128) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, idempotency_key)
);
CREATE TABLE game_hall_jackpot_transactions (
    id BIGSERIAL PRIMARY KEY,
    jackpot_code VARCHAR(32) NOT NULL REFERENCES game_hall_jackpots(code),
    tx_type VARCHAR(32) NOT NULL,
    amount NUMERIC(20, 8) NOT NULL,
    balance_before NUMERIC(20, 8) NOT NULL,
    balance_after NUMERIC(20, 8) NOT NULL,
    reference_type VARCHAR(32) NOT NULL,
    reference_id VARCHAR(128) NOT NULL,
    user_id BIGINT NULL REFERENCES users(id),
    idempotency_key VARCHAR(160) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (jackpot_code, idempotency_key)
);
CREATE TABLE game_hall_rounds (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    game_type VARCHAR(32) NOT NULL,
    bet_amount NUMERIC(20, 8) NOT NULL,
    payout_amount NUMERIC(20, 8) NOT NULL,
    net_amount NUMERIC(20, 8) NOT NULL,
    multiplier NUMERIC(20, 8) NOT NULL,
    balance_before NUMERIC(20, 8) NOT NULL,
    balance_after NUMERIC(20, 8) NOT NULL,
    jackpot_before NUMERIC(20, 8) NOT NULL,
    jackpot_after NUMERIC(20, 8) NOT NULL,
    outcome VARCHAR(16) NOT NULL,
    symbols JSONB NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, idempotency_key)
);
CREATE TABLE game_hall_main_balance_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    direction VARCHAR(32) NOT NULL,
    amount NUMERIC(20, 8) NOT NULL,
    balance_before NUMERIC(20, 8) NOT NULL,
    balance_after NUMERIC(20, 8) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, idempotency_key)
);`)
	return err
}
