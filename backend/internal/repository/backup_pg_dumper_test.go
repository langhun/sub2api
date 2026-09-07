package repository

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func testShellCommand(ctx context.Context, script string) *exec.Cmd {
	if runtime.GOOS != "windows" {
		return exec.CommandContext(ctx, "sh", "-c", script)
	}
	// The repository's tests run on Windows as well as Linux. Keep the
	// shell snippets equivalent without requiring WSL or a profile-installed
	// Unix shell on the developer machine.
	switch script {
	case "printf backup-data":
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "[Console]::Write('backup-data')")
	case "printf partial-backup; exit 7":
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "[Console]::Write('partial-backup'); exit 7")
	case "true":
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "exit 0")
	default:
		return exec.CommandContext(ctx, "cmd", "/C", script)
	}
}

func newTestPgDumper(t *testing.T, commandContext func(context.Context, string, ...string) *exec.Cmd) (*PgDumper, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return &PgDumper{
		cfg: &config.DatabaseConfig{
			Host:     "db.example.test",
			Port:     5432,
			User:     "sub2api",
			Password: "secret",
			DBName:   "sub2api",
			SSLMode:  "require",
		},
		db:             db,
		commandContext: commandContext,
	}, mock
}

func expectBackupMigrationLock(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock($1)")).
		WithArgs(migrationsAdvisoryLockID).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
}

func expectBackupMigrationUnlock(mock sqlmock.Sqlmock) {
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
		WithArgs(migrationsAdvisoryLockID).
		WillReturnResult(sqlmock.NewResult(0, 1))
}

func TestPgDumperHoldsMigrationLockThroughReaderClose(t *testing.T) {
	var mock sqlmock.Sqlmock
	commandCreated := false
	dumper, createdMock := newTestPgDumper(t, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		commandCreated = true
		require.Equal(t, "pg_dump", name)
		require.Contains(t, args, "--clean")
		require.NoError(t, mock.ExpectationsWereMet(), "migration lock must be acquired before pg_dump is created")
		return testShellCommand(ctx, "printf backup-data")
	})
	mock = createdMock
	expectBackupMigrationLock(mock)

	reader, err := dumper.Dump(context.Background())
	require.NoError(t, err)
	require.True(t, commandCreated)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "backup-data", string(data))

	expectBackupMigrationUnlock(mock)
	require.Error(t, mock.ExpectationsWereMet(), "migration lock was released before the reader closed")
	require.NoError(t, reader.Close())
	require.NoError(t, mock.ExpectationsWereMet())
	require.NoError(t, reader.Close(), "reader close must be idempotent")
}

func TestPgDumperReleasesMigrationLockWhenStdoutPipeSetupFails(t *testing.T) {
	dumper, mock := newTestPgDumper(t, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cmd := testShellCommand(ctx, "true")
		cmd.Stdout = io.Discard
		return cmd
	})
	expectBackupMigrationLock(mock)
	expectBackupMigrationUnlock(mock)

	reader, err := dumper.Dump(context.Background())
	require.Nil(t, reader)
	require.ErrorContains(t, err, "create stdout pipe")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgDumperReleasesMigrationLockWhenProcessStartFails(t *testing.T) {
	dumper, mock := newTestPgDumper(t, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "/path/that/does/not/exist/pg_dump")
	})
	expectBackupMigrationLock(mock)
	expectBackupMigrationUnlock(mock)

	reader, err := dumper.Dump(context.Background())
	require.Nil(t, reader)
	require.ErrorContains(t, err, "start pg_dump")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgDumperReleasesMigrationLockWhenProcessFails(t *testing.T) {
	dumper, mock := newTestPgDumper(t, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return testShellCommand(ctx, "printf partial-backup; exit 7")
	})
	expectBackupMigrationLock(mock)

	reader, err := dumper.Dump(context.Background())
	require.NoError(t, err)
	data, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, "partial-backup", string(data))
	expectBackupMigrationUnlock(mock)
	require.ErrorContains(t, reader.Close(), "pg_dump exited with error")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgDumperReportsUnlockFailureAndDiscardsConnection(t *testing.T) {
	dumper, mock := newTestPgDumper(t, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return testShellCommand(ctx, "printf backup-data")
	})
	expectBackupMigrationLock(mock)

	reader, err := dumper.Dump(context.Background())
	require.NoError(t, err)
	_, err = io.ReadAll(reader)
	require.NoError(t, err)
	mock.ExpectExec(regexp.QuoteMeta("SELECT pg_advisory_unlock($1)")).
		WithArgs(migrationsAdvisoryLockID).
		WillReturnError(errors.New("unlock unavailable"))
	require.ErrorContains(t, reader.Close(), "release backup migration lock")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgDumperDoesNotStartProcessWhenMigrationLockFails(t *testing.T) {
	commandCreated := false
	dumper, mock := newTestPgDumper(t, func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		commandCreated = true
		return testShellCommand(ctx, "true")
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT pg_try_advisory_lock($1)")).
		WithArgs(migrationsAdvisoryLockID).
		WillReturnError(errors.New("database unavailable"))

	reader, err := dumper.Dump(context.Background())
	require.Nil(t, reader)
	require.ErrorContains(t, err, "acquire backup migration lock")
	require.False(t, commandCreated)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPgDumperRejectsNilDatabase(t *testing.T) {
	dumper := &PgDumper{cfg: &config.DatabaseConfig{}}
	reader, err := dumper.Dump(context.Background())
	require.Nil(t, reader)
	require.ErrorContains(t, err, "nil sql db")
}
