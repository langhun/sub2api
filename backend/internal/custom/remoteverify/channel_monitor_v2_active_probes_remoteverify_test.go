//go:build remoteverify

package remoteverify_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestRemoteDatabaseChannelMonitorV2ActiveProbeMigration(t *testing.T) {
	if os.Getenv(remoteVerifyEnableEnv) != "1" {
		t.Skipf("set %s=1 to permit an isolated remote database verification", remoteVerifyEnableEnv)
	}
	if os.Getenv("DATA_DIR") == "" {
		t.Fatal("DATA_DIR must point at the existing local configuration directory")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Minute)
	defer cancel()
	db, _ := openIsolatedDatabase(t, ctx)

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (
			filename TEXT PRIMARY KEY,
			checksum TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		INSERT INTO schema_migrations (filename, checksum) VALUES ('upstream_sentinel.sql', 'unchanged');
		CREATE TABLE channel_monitors (id BIGSERIAL PRIMARY KEY);`); err != nil {
		t.Fatalf("prepare isolated upstream schema: %v", err)
	}
	before := migrationLedger(t, ctx, db, "schema_migrations")
	migrationPath := filepath.Join("..", "migrations", "017_channel_monitor_v2_active_probes.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read active probe migration: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("first active probe migration application: %v", err)
	}
	if _, err := db.ExecContext(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("second active probe migration application: %v", err)
	}
	after := migrationLedger(t, ctx, db, "schema_migrations")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("custom migrations changed upstream history: before=%d rows after=%d rows", len(before), len(after))
	}
	if !relationExists(t, ctx, db, "custom_channel_monitor_v2_active_probe_results") {
		t.Fatal("active probe Overlay table was not created")
	}

	var monitorID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO channel_monitors DEFAULT VALUES RETURNING id`).Scan(&monitorID); err != nil {
		t.Fatalf("create isolated channel monitor: %v", err)
	}
	checkedAt := time.Now().UTC().Truncate(time.Microsecond)
	const writers = 12
	errs := make(chan error, writers)
	var wg sync.WaitGroup
	for range writers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := db.ExecContext(ctx, `
				INSERT INTO custom_channel_monitor_v2_active_probe_results
					(monitor_id, platform, model, status, latency_ms, interval_seconds, checked_at)
				VALUES ($1, 'openai', 'gpt-5', 'operational', 120, 60, $2)
				ON CONFLICT (monitor_id, model, checked_at) DO NOTHING`, monitorID, checkedAt)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent active probe insert: %v", err)
		}
	}
	var rows int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM custom_channel_monitor_v2_active_probe_results WHERE monitor_id = $1`, monitorID).Scan(&rows); err != nil {
		t.Fatalf("count idempotent active probe facts: %v", err)
	}
	if rows != 1 {
		t.Fatalf("active probe facts = %d, want 1 after concurrent duplicate inserts", rows)
	}
}
