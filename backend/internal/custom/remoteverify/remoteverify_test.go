//go:build remoteverify

// Package remoteverify exercises the Overlay against an explicitly isolated
// PostgreSQL database. It is intentionally excluded from normal test runs.
package remoteverify_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/config"
	custommigrations "github.com/Wei-Shaw/sub2api/internal/custom/migrations"
	activitycontract "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
	"github.com/Wei-Shaw/sub2api/internal/repository"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
)

const remoteVerifyEnableEnv = "SUB2API_REMOTE_VERIFY"

var remoteDatabaseNamePattern = regexp.MustCompile(`\Asub2api_overlay_verify_[0-9a-f]{12}\z`)

func TestRemoteDatabaseOverlayVerification(t *testing.T) {
	if os.Getenv(remoteVerifyEnableEnv) != "1" {
		t.Skipf("set %s=1 to permit an isolated remote database verification", remoteVerifyEnableEnv)
	}
	if os.Getenv("DATA_DIR") == "" {
		t.Fatal("DATA_DIR must point at the existing local configuration directory")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Minute)
	defer cancel()
	db, client := openIsolatedDatabase(t, ctx)

	t.Run("migrations_are_append_only_and_reentrant", func(t *testing.T) {
		t.Log("apply core migrations")
		if err := repository.ApplyMigrations(ctx, db); err != nil {
			t.Fatalf("apply core migrations to isolated database: %v", err)
		}
		t.Log("core migrations complete")
		before := migrationLedger(t, ctx, db, "schema_migrations")

		t.Log("apply custom migrations first time")
		if err := custommigrations.Apply(ctx, db); err != nil {
			t.Fatalf("first custom migration application: %v", err)
		}
		t.Log("apply custom migrations second time")
		if err := custommigrations.Apply(ctx, db); err != nil {
			t.Fatalf("second custom migration application: %v", err)
		}
		t.Log("custom migrations complete")

		after := migrationLedger(t, ctx, db, "schema_migrations")
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("custom migrations changed schema_migrations: before=%d rows after=%d rows", len(before), len(after))
		}
		if exists := relationExists(t, ctx, db, "custom_schema_migrations"); !exists {
			t.Fatal("custom_schema_migrations was not created")
		}
		if exists := relationExists(t, ctx, db, "custom_activity_redeem_metadata"); !exists {
			t.Fatal("custom_activity_redeem_metadata was not created")
		}

		customLedger := migrationLedger(t, ctx, db, "custom_schema_migrations")
		if len(customLedger) != 1 || customLedger[0].Filename != "001_activity_redeem_metadata.sql" {
			t.Fatalf("unexpected custom migration ledger: %#v", customLedger)
		}
		var leaked int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE filename = $1`, "001_activity_redeem_metadata.sql").Scan(&leaked); err != nil {
			t.Fatalf("check core migration ledger for custom entry: %v", err)
		}
		if leaked != 0 {
			t.Fatalf("custom migration leaked into schema_migrations: %d rows", leaked)
		}
	})

	t.Run("reward_transactions_idempotency_and_concurrency", func(t *testing.T) {
		rewardUser := createUser(t, ctx, client, "reward", 0)
		prize := client.CheckinPrizeItem.Create().
			SetName("remoteverify balance prize").
			SetRarity("common").
			SetRewardType("balance").
			SetRewardValue(3).
			SetRewardValueMax(3).
			SetWeight(1).
			SetIsEnabled(true)
		prizeItem, err := prize.Save(ctx)
		if err != nil {
			t.Fatalf("create isolated reward prize: %v", err)
		}

		writer := rewards.NewEntBalanceWriter(client)
		tx, err := client.Tx(ctx)
		if err != nil {
			t.Fatalf("begin reward rollback transaction: %v", err)
		}
		txCtx := dbent.NewTxContext(ctx, tx)
		if err := writer.Credit(txCtx, activitycontract.BalanceOperation{UserID: rewardUser.ID, Amount: 7}); err != nil {
			t.Fatalf("credit inside reward rollback transaction: %v", err)
		}
		assertBalance(t, ctx, tx.Client(), rewardUser.ID, 7)
		if err := tx.Rollback(); err != nil {
			t.Fatalf("rollback reward transaction: %v", err)
		}
		assertBalance(t, ctx, client, rewardUser.ID, 0)

		outbox := rewards.NewOutboxRepository(client, db)
		input := rewardDeliveryInput(t, prizeItem.ID, rewardUser.ID, 1001)
		const enqueueWorkers = 12
		results := make(chan *rewards.Delivery, enqueueWorkers)
		errs := make(chan error, enqueueWorkers)
		var workers sync.WaitGroup
		for range enqueueWorkers {
			workers.Add(1)
			go func() {
				defer workers.Done()
				delivery, enqueueErr := outbox.Enqueue(ctx, input)
				if enqueueErr != nil {
					errs <- enqueueErr
					return
				}
				results <- delivery
			}()
		}
		workers.Wait()
		close(results)
		close(errs)
		for enqueueErr := range errs {
			t.Fatalf("concurrent reward enqueue: %v", enqueueErr)
		}
		var deliveryID int64
		for delivery := range results {
			if deliveryID == 0 {
				deliveryID = delivery.ID
			}
			if delivery.ID != deliveryID {
				t.Fatalf("idempotent reward enqueue returned different ids: %d and %d", deliveryID, delivery.ID)
			}
		}
		if deliveryID == 0 {
			t.Fatal("concurrent reward enqueue returned no delivery")
		}
		var deliveryCount int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reward_deliveries WHERE idempotency_key = $1`, input.IdempotencyKey).Scan(&deliveryCount); err != nil {
			t.Fatalf("count idempotent reward deliveries: %v", err)
		}
		if deliveryCount != 1 {
			t.Fatalf("expected one reward delivery for idempotency key, got %d", deliveryCount)
		}

		claimed, err := outbox.ClaimByID(ctx, deliveryID, time.Now().UTC())
		if err != nil || claimed == nil {
			t.Fatalf("claim reward delivery: delivery=%v err=%v", claimed, err)
		}
		if err := outbox.ExecuteClaimed(ctx, deliveryID, func(applyCtx context.Context, _ rewards.Delivery) (string, error) {
			return "remoteverify credited", writer.Credit(applyCtx, activitycontract.BalanceOperation{UserID: rewardUser.ID, Amount: 3})
		}); err != nil {
			t.Fatalf("execute claimed reward: %v", err)
		}
		assertBalance(t, ctx, client, rewardUser.ID, 3)
		delivery, err := outbox.Get(ctx, deliveryID)
		if err != nil {
			t.Fatalf("load delivered reward: %v", err)
		}
		if delivery.Status != rewards.DeliveryStatusDelivered || delivery.RewardDetail != "remoteverify credited" {
			t.Fatalf("unexpected delivered reward state: %#v", delivery)
		}
		if err := outbox.ExecuteClaimed(ctx, deliveryID, func(context.Context, rewards.Delivery) (string, error) {
			return "must not execute", nil
		}); !errors.Is(err, rewards.ErrStateConflict) {
			t.Fatalf("second reward delivery execution = %v, want state conflict", err)
		}
		assertBalance(t, ctx, client, rewardUser.ID, 3)

		rollbackInput := rewardDeliveryInput(t, prizeItem.ID, rewardUser.ID, 1002)
		rollbackDelivery, err := outbox.Enqueue(ctx, rollbackInput)
		if err != nil {
			t.Fatalf("enqueue rollback reward: %v", err)
		}
		if claimed, err := outbox.ClaimByID(ctx, rollbackDelivery.ID, time.Now().UTC()); err != nil || claimed == nil {
			t.Fatalf("claim rollback reward: delivery=%v err=%v", claimed, err)
		}
		applyErr := errors.New("remoteverify force reward rollback")
		if err := outbox.ExecuteClaimed(ctx, rollbackDelivery.ID, func(applyCtx context.Context, _ rewards.Delivery) (string, error) {
			if creditErr := writer.Credit(applyCtx, activitycontract.BalanceOperation{UserID: rewardUser.ID, Amount: 3}); creditErr != nil {
				return "", creditErr
			}
			return "", applyErr
		}); !errors.Is(err, applyErr) {
			t.Fatalf("force reward rollback = %v, want %v", err, applyErr)
		}
		assertBalance(t, ctx, client, rewardUser.ID, 3)
		rolledBackDelivery, err := outbox.Get(ctx, rollbackDelivery.ID)
		if err != nil {
			t.Fatalf("load rolled back reward: %v", err)
		}
		if rolledBackDelivery.Status != rewards.DeliveryStatusDelivering {
			t.Fatalf("failed reward transaction changed delivery state to %q", rolledBackDelivery.Status)
		}
	})

}

type ledgerRow struct {
	Filename string
	Checksum string
}

func openIsolatedDatabase(t *testing.T, ctx context.Context) (*sql.DB, *dbent.Client) {
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
	cleanupStaleRemoteVerificationDatabases(t, ctx, adminDB)

	name := remoteDatabaseName(t)
	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE "+quoteIdentifier(name)); err != nil {
		_ = adminDB.Close()
		t.Fatalf("create isolated verification database %q: %v", name, err)
	}
	t.Logf("remote verification: created isolated database %s", name)

	verifyConfig := cfg.Database
	verifyConfig.DBName = name
	db, err := sql.Open("postgres", verifyConfig.DSNWithTimezone("UTC"))
	if err != nil {
		cleanupRemoteDatabase(t, ctx, adminDB, name)
		_ = adminDB.Close()
		t.Fatalf("open isolated verification database: %v", err)
	}
	db.SetMaxOpenConns(24)
	db.SetMaxIdleConns(8)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		cleanupRemoteDatabase(t, ctx, adminDB, name)
		_ = adminDB.Close()
		t.Fatalf("ping isolated verification database: %v", err)
	}
	t.Log("remote verification: isolated database connection established")
	client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
		cleanupRemoteDatabase(t, context.Background(), adminDB, name)
		_ = adminDB.Close()
	})
	return db, client
}

func remoteDatabaseName(t *testing.T) string {
	t.Helper()
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate isolated database suffix: %v", err)
	}
	return "sub2api_overlay_verify_" + hex.EncodeToString(bytes)
}

func cleanupRemoteDatabase(t *testing.T, ctx context.Context, adminDB *sql.DB, name string) {
	t.Helper()
	if _, err := adminDB.ExecContext(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, name); err != nil {
		t.Errorf("terminate isolated verification database connections: %v", err)
	}
	if _, err := adminDB.ExecContext(ctx, "DROP DATABASE IF EXISTS "+quoteIdentifier(name)); err != nil {
		t.Errorf("drop isolated verification database %q: %v", name, err)
	}
}

func cleanupStaleRemoteVerificationDatabases(t *testing.T, ctx context.Context, adminDB *sql.DB) {
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
		if !remoteDatabaseNamePattern.MatchString(name) {
			continue
		}
		t.Logf("remote verification: cleaning stale isolated database %s", name)
		cleanupRemoteDatabase(t, ctx, adminDB, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate stale isolated verification databases: %v", err)
	}
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func relationExists(t *testing.T, ctx context.Context, db *sql.DB, relation string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+relation).Scan(&exists); err != nil {
		t.Fatalf("check relation %s: %v", relation, err)
	}
	return exists
}

func migrationLedger(t *testing.T, ctx context.Context, db *sql.DB, table string) []ledgerRow {
	t.Helper()
	rows, err := db.QueryContext(ctx, "SELECT filename, checksum FROM "+quoteIdentifier(table)+" ORDER BY filename")
	if err != nil {
		t.Fatalf("read %s: %v", table, err)
	}
	defer func() { _ = rows.Close() }()
	ledger := make([]ledgerRow, 0)
	for rows.Next() {
		var row ledgerRow
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

func createUser(t *testing.T, ctx context.Context, client *dbent.Client, label string, balance float64) *dbent.User {
	t.Helper()
	suffix := remoteDatabaseName(t)[len("sub2api_overlay_verify_"):]
	item, err := client.User.Create().
		SetEmail(fmt.Sprintf("%s-%s@remoteverify.invalid", label, suffix)).
		SetPasswordHash("remoteverify-password-hash").
		SetUsername(label + "-" + suffix).
		SetBalance(balance).
		SetStatus("active").
		Save(ctx)
	if err != nil {
		t.Fatalf("create isolated user %s: %v", label, err)
	}
	return item
}

func rewardDeliveryInput(t *testing.T, prizeID, userID, sourceID int64) rewards.CreateDelivery {
	t.Helper()
	snapshot, err := json.Marshal(rewards.Snapshot{
		PrizeID: prizeID, PrizeName: "remoteverify balance prize", Rarity: rewards.RarityCommon,
		RewardType: rewards.RewardTypeBalance, RewardValue: 3, StreakDays: 1,
	})
	if err != nil {
		t.Fatalf("marshal reward snapshot: %v", err)
	}
	return rewards.CreateDelivery{
		SourceType: rewards.SourceCheckinBlindbox, SourceID: sourceID, UserID: userID, PrizeID: &prizeID,
		RewardSnapshot: snapshot, RewardType: rewards.RewardTypeBalance, RewardValue: 3,
		RuleVersion: rewards.CheckinBlindboxRuleV1, IdempotencyKey: fmt.Sprintf("remoteverify-reward:%d", sourceID),
	}
}

func assertBalance(t *testing.T, ctx context.Context, client *dbent.Client, userID int64, want float64) {
	t.Helper()
	item, err := client.User.Query().Where(user.IDEQ(userID)).Only(ctx)
	if err != nil {
		t.Fatalf("read user %d balance: %v", userID, err)
	}
	if math.Abs(item.Balance-want) > 1e-8 {
		t.Fatalf("user %d balance = %.8f, want %.8f", userID, item.Balance, want)
	}
}
