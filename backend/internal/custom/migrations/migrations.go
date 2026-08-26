// Package migrations applies schema changes owned by the custom Overlay.
package migrations

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed *.sql
var files embed.FS

var historicalMigrationChecksums = map[string][]string{
	// 012 was already applied in production before the sticky-routing field
	// was made backwards-compatible. Keep that deployed checksum accepted so
	// changing the migration source does not brick startup on existing data.
	"012_smart_routing.sql": {
		"f8788dd4063924c194ee09180cd6d9073ba22c02e149ecd3c57a12c99bf6627e",
	},
	// This checksum was recorded by the first production candidate that applied
	// 005 before the final release artifact was built. Keep the migration file
	// immutable and accept only that known deployed revision.
	"005_recharge_promotions_tiered_percentage.sql": {
		"7ecb972b0cef7ba126c2b933385ef8b80ce5820121bacd61d899ee96656224ff",
	},
}

const customMigrationsAdvisoryLockID int64 = 694208311321144028

const customSchemaMigrationsTableDDL = `
CREATE TABLE IF NOT EXISTS custom_schema_migrations (
	filename TEXT PRIMARY KEY,
	checksum TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// Apply runs immutable Overlay migrations after the host application's schema
// has been prepared. Its ledger cannot interfere with upstream migrations.
func Apply(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("nil sql db")
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire custom migrations connection: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", customMigrationsAdvisoryLockID); err != nil {
		return fmt.Errorf("lock custom migrations: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(unlockCtx, "SELECT pg_advisory_unlock($1)", customMigrationsAdvisoryLockID)
	}()

	if _, err := conn.ExecContext(ctx, customSchemaMigrationsTableDDL); err != nil {
		return fmt.Errorf("create custom_schema_migrations: %w", err)
	}

	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return fmt.Errorf("list custom migrations: %w", err)
	}
	sort.Strings(names)

	for _, name := range names {
		body, err := fs.ReadFile(files, name)
		if err != nil {
			return fmt.Errorf("read custom migration %s: %w", name, err)
		}
		content := strings.TrimSpace(string(body))
		if content == "" {
			continue
		}
		checksum, compatibleChecksums := migrationChecksums(content)
		for _, historicalChecksum := range historicalMigrationChecksums[name] {
			compatibleChecksums[historicalChecksum] = struct{}{}
		}

		var existing string
		err = conn.QueryRowContext(ctx, "SELECT checksum FROM custom_schema_migrations WHERE filename = $1", name).Scan(&existing)
		if err == nil {
			if _, compatible := compatibleChecksums[existing]; !compatible {
				return fmt.Errorf("custom migration %s checksum mismatch", name)
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check custom migration %s: %w", name, err)
		}

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin custom migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, content); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply custom migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO custom_schema_migrations (filename, checksum) VALUES ($1, $2)", name, checksum); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record custom migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit custom migration %s: %w", name, err)
		}
	}

	return nil
}

func migrationChecksums(content string) (string, map[string]struct{}) {
	lf := strings.ReplaceAll(strings.TrimSpace(content), "\r\n", "\n")
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	canonical := migrationChecksum(lf)
	return canonical, map[string]struct{}{
		canonical:               {},
		migrationChecksum(crlf): {},
	}
}

func migrationChecksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
