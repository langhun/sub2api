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
		sum := sha256.Sum256([]byte(content))
		checksum := hex.EncodeToString(sum[:])

		var existing string
		err = conn.QueryRowContext(ctx, "SELECT checksum FROM custom_schema_migrations WHERE filename = $1", name).Scan(&existing)
		if err == nil {
			if existing != checksum {
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
