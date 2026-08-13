//go:build remoteverify

// Package remoteverify exercises the Overlay against an explicitly isolated
// PostgreSQL database. It is intentionally excluded from normal test runs.
package remoteverify_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	custommigrations "github.com/Wei-Shaw/sub2api/internal/custom/migrations"
	rechargepromotions "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions"
	rechargepromotionsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
)

const rechargeRemoteVerifyEnableEnv = "SUB2API_REMOTE_VERIFY"

var rechargeRemoteDatabaseNamePattern = regexp.MustCompile(`\Asub2api_overlay_verify_[0-9a-f]{12}\z`)

func TestRemoteDatabaseRechargePromotionVerification(t *testing.T) {
	if os.Getenv(rechargeRemoteVerifyEnableEnv) != "1" {
		t.Skipf("set %s=1 to permit an isolated remote database verification", rechargeRemoteVerifyEnableEnv)
	}
	if os.Getenv("DATA_DIR") == "" {
		t.Fatal("DATA_DIR must point at the existing local configuration directory")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Minute)
	defer cancel()
	db, client := openRechargeIsolatedDatabase(t, ctx)
	prepareMinimalPromotionSchema(t, ctx, db)

	t.Run("migrations_are_append_only_and_reentrant", func(t *testing.T) {
		before := rechargeMigrationLedger(t, ctx, db, "schema_migrations")

		t.Log("apply custom migrations first time")
		if err := custommigrations.Apply(ctx, db); err != nil {
			t.Fatalf("first custom migration application: %v", err)
		}
		t.Log("apply custom migrations second time")
		if err := custommigrations.Apply(ctx, db); err != nil {
			t.Fatalf("second custom migration application: %v", err)
		}
		t.Log("custom migrations complete")

		after := rechargeMigrationLedger(t, ctx, db, "schema_migrations")
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("custom migrations changed schema_migrations: before=%d rows after=%d rows", len(before), len(after))
		}
		if exists := rechargeRelationExists(t, ctx, db, "custom_schema_migrations"); !exists {
			t.Fatal("custom_schema_migrations was not created")
		}
		if exists := rechargeRelationExists(t, ctx, db, "custom_recharge_promotion_orders"); !exists {
			t.Fatal("custom_recharge_promotion_orders was not created")
		}

		customLedger := rechargeMigrationLedger(t, ctx, db, "custom_schema_migrations")
		wantCustomMigrations := []string{
			"001_activity_redeem_metadata.sql",
			"002_support_tickets.sql",
			"003_recharge_promotions.sql",
		}
		if len(customLedger) != len(wantCustomMigrations) {
			t.Fatalf("unexpected custom migration ledger: %#v", customLedger)
		}
		for index, filename := range wantCustomMigrations {
			if customLedger[index].Filename != filename {
				t.Fatalf("custom migration ledger[%d] = %q, want %q", index, customLedger[index].Filename, filename)
			}
			var leaked int
			if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, filename).Scan(&leaked); err != nil {
				t.Fatalf("check core migration ledger for custom entry %q: %v", filename, err)
			}
			if leaked != 0 {
				t.Fatalf("custom migration %q leaked into schema_migrations: %d rows", filename, leaked)
			}
		}
	})

	t.Run("recharge_promotion_snapshot_is_atomic_unique_and_refund_safe", func(t *testing.T) {
		promotion := rechargepromotions.NewModule(rechargePromotionSettingsStub{config: rechargepromotionsettings.Config{
			Enabled: true, Mode: rechargepromotionsettings.ModePercentage, PercentageRate: 20,
		}}, db)
		input := service.BalanceOrderExtensionInput{
			PaymentAmount: 100, BaseCredit: 120, PayAmount: 100, Now: time.Now().UTC(),
		}
		result, err := promotion.PrepareBalanceOrder(ctx, input)
		if err != nil {
			t.Fatalf("calculate percentage promotion: %v", err)
		}
		if math.Abs(result.CreditedAmount-144) > 1e-8 {
			t.Fatalf("promoted credit = %.8f, want 144", result.CreditedAmount)
		}

		userID := createPromotionUser(t, ctx, db)
		input.UserID = userID

		// The extension write is deliberately in the same transaction as the order.
		// A unique violation must leave neither record committed.
		rollbackTradeNo := "remoteverify-promotion-rollback-" + rechargeRemoteDatabaseName(t)
		tx, err := client.Tx(ctx)
		if err != nil {
			t.Fatalf("begin promotion rollback transaction: %v", err)
		}
		txCtx := dbent.NewTxContext(ctx, tx)
		rollbackOrder := createPaymentOrder(t, txCtx, tx.Client(), userID, rollbackTradeNo, result.CreditedAmount)
		if err := promotion.PersistBalanceOrder(txCtx, tx.Client(), rollbackOrder, input, result); err != nil {
			_ = tx.Rollback()
			t.Fatalf("persist first promotion snapshot: %v", err)
		}
		if err := promotion.PersistBalanceOrder(txCtx, tx.Client(), rollbackOrder, input, result); err == nil {
			_ = tx.Rollback()
			t.Fatal("duplicate promotion snapshot unexpectedly succeeded")
		}
		if err := tx.Rollback(); err != nil {
			t.Fatalf("rollback failed promotion transaction: %v", err)
		}
		var rollbackOrderCount int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_orders WHERE out_trade_no = $1`, rollbackTradeNo).Scan(&rollbackOrderCount); err != nil {
			t.Fatalf("count rolled back payment order: %v", err)
		}
		if rollbackOrderCount != 0 {
			t.Fatalf("promotion snapshot failure committed %d payment order rows", rollbackOrderCount)
		}

		order := createPaymentOrder(t, ctx, client, userID, "remoteverify-promotion-"+rechargeRemoteDatabaseName(t), result.CreditedAmount)
		persistPromotionSnapshot(t, ctx, client, promotion, order, input, result)
		assertPromotionSnapshot(t, ctx, db, order.ID, "120.00000000", "24.00000000", "144.00000000", "0.00000000", "0.00000000", "0.00000000")

		// The application permits one refund attempt per order. Replaying the same
		// extension call is nevertheless stable under concurrent transaction locks.
		const refundWorkers = 8
		ready := make(chan struct{})
		refundErrs := make(chan error, refundWorkers)
		var refundWorkersGroup sync.WaitGroup
		for range refundWorkers {
			refundWorkersGroup.Add(1)
			go func() {
				defer refundWorkersGroup.Done()
				<-ready
				workerTx, workerErr := client.Tx(ctx)
				if workerErr != nil {
					refundErrs <- workerErr
					return
				}
				workerCtx := dbent.NewTxContext(ctx, workerTx)
				workerErr = promotion.RecordBalanceRefund(workerCtx, workerTx.Client(), service.BalanceRefundExtensionInput{
					Order: order, RefundAmount: 72, RefundedAt: time.Now().UTC(),
				})
				if workerErr == nil {
					workerErr = workerTx.Commit()
				} else {
					_ = workerTx.Rollback()
				}
				if workerErr != nil {
					refundErrs <- workerErr
				}
			}()
		}
		close(ready)
		refundWorkersGroup.Wait()
		close(refundErrs)
		for refundErr := range refundErrs {
			t.Fatalf("concurrent promotion refund snapshot: %v", refundErr)
		}
		assertPromotionSnapshot(t, ctx, db, order.ID, "120.00000000", "24.00000000", "144.00000000", "72.00000000", "60.00000000", "12.00000000")

		// A full refund records the whole original base and bonus; the amount is
		// capped at final credit so a malformed caller cannot exceed the snapshot.
		fullRefundTx, err := client.Tx(ctx)
		if err != nil {
			t.Fatalf("begin full promotion refund transaction: %v", err)
		}
		fullRefundCtx := dbent.NewTxContext(ctx, fullRefundTx)
		if err := promotion.RecordBalanceRefund(fullRefundCtx, fullRefundTx.Client(), service.BalanceRefundExtensionInput{
			Order: order, RefundAmount: 999, RefundedAt: time.Now().UTC(),
		}); err != nil {
			_ = fullRefundTx.Rollback()
			t.Fatalf("record capped promotion refund: %v", err)
		}
		if err := fullRefundTx.Commit(); err != nil {
			t.Fatalf("commit capped promotion refund: %v", err)
		}
		assertPromotionSnapshot(t, ctx, db, order.ID, "120.00000000", "24.00000000", "144.00000000", "144.00000000", "120.00000000", "24.00000000")
	})

}

type rechargeLedgerRow struct {
	Filename string
	Checksum string
}

func openRechargeIsolatedDatabase(t *testing.T, ctx context.Context) (*sql.DB, *dbent.Client) {
	t.Helper()
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		t.Fatalf("load existing remote configuration: %v", err)
	}
	adminConfig := cfg.Database
	adminConfig.DBName = "postgres"
	adminDB, err := sql.Open("postgres", adminConfig.DSNWithTimezone("UTC"))
	if err != nil {
		t.Fatalf("open remote PostgreSQL administrative connection: %v", err)
	}
	preflightCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := adminDB.PingContext(preflightCtx); err != nil {
		_ = adminDB.Close()
		t.Fatalf("ping remote PostgreSQL administrative connection: %v", err)
	}
	t.Log("remote verification: administrative PostgreSQL connection established")
	var canCreateDatabase bool
	if err := adminDB.QueryRowContext(ctx, `SELECT rolcreatedb FROM pg_roles WHERE rolname = CURRENT_USER`).Scan(&canCreateDatabase); err != nil {
		_ = adminDB.Close()
		t.Fatalf("check remote temporary-database capability: %v", err)
	}
	if !canCreateDatabase {
		_ = adminDB.Close()
		t.Fatal("current remote PostgreSQL role cannot create an isolated verification database; provide a dedicated test database or CREATEDB role")
	}
	cleanupStaleRechargeRemoteVerificationDatabases(t, ctx, adminDB)

	name := rechargeRemoteDatabaseName(t)
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+rechargeQuoteIdentifier(name)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create isolated verification database %q: %v", name, err)
	}
	t.Logf("remote verification: created isolated database %s", name)

	verifyConfig := cfg.Database
	verifyConfig.DBName = name
	db, err := sql.Open("postgres", verifyConfig.DSNWithTimezone("UTC"))
	if err != nil {
		cleanupRechargeRemoteDatabase(t, ctx, adminDB, name)
		_ = adminDB.Close()
		t.Fatalf("open isolated verification database: %v", err)
	}
	db.SetMaxOpenConns(24)
	db.SetMaxIdleConns(8)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		cleanupRechargeRemoteDatabase(t, ctx, adminDB, name)
		_ = adminDB.Close()
		t.Fatalf("ping isolated verification database: %v", err)
	}
	t.Log("remote verification: isolated database connection established")
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
		cleanupRechargeRemoteDatabase(t, context.Background(), adminDB, name)
		_ = adminDB.Close()
	})
	return db, client
}

func rechargeRemoteDatabaseName(t *testing.T) string {
	t.Helper()
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate isolated database suffix: %v", err)
	}
	return "sub2api_overlay_verify_" + hex.EncodeToString(bytes)
}

func cleanupRechargeRemoteDatabase(t *testing.T, ctx context.Context, adminDB *sql.DB, name string) {
	t.Helper()
	if _, err := adminDB.ExecContext(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, name); err != nil {
		t.Errorf("terminate isolated verification database connections: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+rechargeQuoteIdentifier(name)); err != nil {
		t.Errorf("drop isolated verification database %q: %v", name, err)
	}
}

func cleanupStaleRechargeRemoteVerificationDatabases(t *testing.T, ctx context.Context, adminDB *sql.DB) {
	t.Helper()
	rows, err := adminDB.QueryContext(ctx, `SELECT datname FROM pg_database WHERE datname LIKE 'sub2api_overlay_verify_%' ORDER BY datname`)
	if err != nil {
		t.Fatalf("list stale isolated verification databases: %v", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan stale isolated verification database: %v", err)
		}
		if !rechargeRemoteDatabaseNamePattern.MatchString(name) {
			continue
		}
		t.Logf("remote verification: cleaning stale isolated database %s", name)
		cleanupRechargeRemoteDatabase(t, ctx, adminDB, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate stale isolated verification databases: %v", err)
	}
}

func rechargeQuoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func rechargeRelationExists(t *testing.T, ctx context.Context, db *sql.DB, relation string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+relation).Scan(&exists); err != nil {
		t.Fatalf("check relation %s: %v", relation, err)
	}
	return exists
}

func rechargeMigrationLedger(t *testing.T, ctx context.Context, db *sql.DB, table string) []rechargeLedgerRow {
	t.Helper()
	rows, err := db.QueryContext(ctx, "SELECT filename, checksum FROM "+rechargeQuoteIdentifier(table)+" ORDER BY filename")
	if err != nil {
		t.Fatalf("read %s: %v", table, err)
	}
	defer func() { _ = rows.Close() }()
	ledger := make([]rechargeLedgerRow, 0)
	for rows.Next() {
		var row rechargeLedgerRow
		if err := rows.Scan(&row.Filename, &row.Checksum); err != nil {
			t.Fatalf("scan %s: %v", table, err)
		}
		ledger = append(ledger, row)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s: %v", table, err)
	}
	return ledger
}

func prepareMinimalPromotionSchema(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
CREATE TABLE schema_migrations (
    filename TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO schema_migrations (filename, checksum) VALUES ('upstream-baseline.sql', 'remoteverify') ON CONFLICT DO NOTHING;
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE redeem_codes (
    id BIGSERIAL PRIMARY KEY,
    multiplier NUMERIC(20, 8) NOT NULL DEFAULT 0,
    bet_amount NUMERIC(38, 18) NOT NULL DEFAULT 0
);
CREATE TABLE payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    out_trade_no TEXT NOT NULL UNIQUE,
    amount NUMERIC(20, 8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`)
	if err != nil {
		t.Fatalf("prepare minimal upstream schema for Overlay verification: %v", err)
	}
}

func createPromotionUser(t *testing.T, ctx context.Context, db *sql.DB) int64 {
	t.Helper()
	var userID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users DEFAULT VALUES RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("create isolated promotion user: %v", err)
	}
	return userID
}

type rechargePromotionSettingsStub struct {
	config rechargepromotionsettings.Config
}

func (s rechargePromotionSettingsStub) ReadRechargePromotions(context.Context) (rechargepromotionsettings.Config, error) {
	return s.config, nil
}

func createPaymentOrder(t *testing.T, ctx context.Context, client *dbent.Client, userID int64, outTradeNo string, amount float64) *dbent.PaymentOrder {
	t.Helper()
	rows, err := client.QueryContext(ctx, `
INSERT INTO payment_orders (user_id, out_trade_no, amount) VALUES ($1, $2, $3) RETURNING id`, userID, outTradeNo, amount)
	if err != nil {
		t.Fatalf("create isolated payment order: %v", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		t.Fatalf("create isolated payment order returned no row: %v", rows.Err())
	}
	var orderID int64
	if err := rows.Scan(&orderID); err != nil {
		t.Fatalf("scan isolated payment order: %v", err)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate isolated payment order: %v", err)
	}
	return &dbent.PaymentOrder{ID: orderID, UserID: userID, Amount: amount, OrderType: "balance"}
}

func persistPromotionSnapshot(t *testing.T, ctx context.Context, client *dbent.Client, promotion *rechargepromotions.Module, order *dbent.PaymentOrder, input service.BalanceOrderExtensionInput, result service.BalanceOrderExtensionResult) {
	t.Helper()
	tx, err := client.Tx(ctx)
	if err != nil {
		t.Fatalf("begin promotion snapshot transaction: %v", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	if err := promotion.PersistBalanceOrder(txCtx, tx.Client(), order, input, result); err != nil {
		_ = tx.Rollback()
		t.Fatalf("persist promotion snapshot: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit promotion snapshot: %v", err)
	}
}

func assertPromotionSnapshot(t *testing.T, ctx context.Context, db *sql.DB, orderID int64, wantBase, wantBonus, wantFinal, wantRefunded, wantRefundedBase, wantRefundedBonus string) {
	t.Helper()
	var base, bonus, final, refunded, refundedBase, refundedBonus string
	err := db.QueryRowContext(ctx, `
SELECT base_credit::text, bonus_credit::text, final_credit::text,
       refunded_credit::text, refunded_base_credit::text, refunded_bonus_credit::text
FROM custom_recharge_promotion_orders WHERE payment_order_id = $1`, orderID).Scan(&base, &bonus, &final, &refunded, &refundedBase, &refundedBonus)
	if err != nil {
		t.Fatalf("read promotion snapshot: %v", err)
	}
	got := []string{base, bonus, final, refunded, refundedBase, refundedBonus}
	want := []string{wantBase, wantBonus, wantFinal, wantRefunded, wantRefundedBase, wantRefundedBonus}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("promotion snapshot = %#v, want %#v", got, want)
	}
}
