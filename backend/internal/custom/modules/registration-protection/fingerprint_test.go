package registrationprotection

import (
	"context"
	"database/sql"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestFingerprintHashNormalizesMetadataWithoutPersistingRawUserAgent(t *testing.T) {
	first, ok := FingerprintHash(" 2001:0DB8::1 ", "Mozilla/5.0\t(Windows)  ")
	second, ok2 := FingerprintHash("2001:db8::1", "mozilla/5.0 (windows)")
	require.True(t, ok)
	require.True(t, ok2)
	require.Equal(t, first, second)
	require.Len(t, first, 64)
	_, ok = FingerprintHash("203.0.113.10", "")
	require.False(t, ok, "an empty User-Agent must not create a reusable fingerprint")
	_, ok = FingerprintHash("", "Mozilla/5.0")
	require.False(t, ok, "an empty client IP must not create a reusable fingerprint")
	if stringsContainInsensitive(first, "mozilla") {
		t.Fatal("fingerprint must not contain raw User-Agent text")
	}
	withDevice, ok := FingerprintHashWithDevice("203.0.113.10", "Mozilla/5.0", "device-a")
	withoutDevice, ok2 := FingerprintHashWithDevice("203.0.113.10", "Mozilla/5.0", "device-b")
	require.True(t, ok)
	require.True(t, ok2)
	require.NotEqual(t, withDevice, withoutDevice)
}

func TestStoreBeginCommitFirstRegistration(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer db.Close()
	store := NewStore(db)
	hash, ok := FingerprintHash("203.0.113.10", "Mozilla/5.0")
	require.True(t, ok)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM custom_registration_fingerprint_reservations")).WithArgs(hash).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO custom_registration_fingerprint_reservations")).WithArgs(hash, sqlmock.AnyArg(), "web").WillReturnRows(sqlmock.NewRows([]string{"reservation_token"}).AddRow("reservation-1"))
	mock.ExpectCommit()

	reservation, err := store.Begin(context.Background(), BeginInput{ClientIP: "203.0.113.10", UserAgent: "Mozilla/5.0", Source: "WEB"})
	require.NoError(t, err)
	require.False(t, reservation.RewardBlocked())
	require.Equal(t, hash, reservation.FingerprintHash())

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM custom_registration_fingerprint_reservations")).WithArgs(hash, "reservation-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO custom_registration_fingerprint_history")).WithArgs(hash, int64(41), "web").WillReturnRows(sqlmock.NewRows([]string{"registration_count"}).AddRow(int64(1)))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO custom_registration_fingerprint_events")).WithArgs(hash, int64(41), "web", false, false).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	result, err := store.Commit(context.Background(), reservation, 41)
	require.NoError(t, err)
	require.True(t, result.FirstRegistration)
	require.False(t, result.RewardBlocked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreExistingFingerprintIsBlockedAndRecorded(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer db.Close()
	store := NewStore(db)
	hash, ok := FingerprintHash("203.0.113.11", "Mozilla/5.0")
	require.True(t, ok)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectCommit()
	reservation, err := store.Begin(context.Background(), BeginInput{ClientIP: "203.0.113.11", UserAgent: "Mozilla/5.0"})
	require.NoError(t, err)
	require.True(t, reservation.RewardBlocked())

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO custom_registration_fingerprint_history")).WithArgs(hash, int64(42), "unknown").WillReturnRows(sqlmock.NewRows([]string{"registration_count"}).AddRow(int64(2)))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO custom_registration_fingerprint_events")).WithArgs(hash, int64(42), "unknown", true, true).WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	result, err := store.Commit(context.Background(), reservation, 42)
	require.NoError(t, err)
	require.True(t, result.RewardBlocked)
	require.False(t, result.FirstRegistration)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(BOOL_OR")).WithArgs(int64(42)).WillReturnRows(sqlmock.NewRows([]string{"blocked"}).AddRow(true))
	blocked, err := store.IsRewardBlocked(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, blocked)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreAbortReleasesReservationAndCommitIsIdempotent(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer db.Close()
	store := NewStore(db)
	hash, ok := FingerprintHash("203.0.113.12", "Mozilla/5.0")
	require.True(t, ok)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT EXISTS")).WithArgs(hash).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM custom_registration_fingerprint_reservations")).WithArgs(hash).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO custom_registration_fingerprint_reservations")).WithArgs(hash, sqlmock.AnyArg(), "unknown").WillReturnRows(sqlmock.NewRows([]string{"reservation_token"}).AddRow("reservation-2"))
	mock.ExpectCommit()
	reservation, err := store.Begin(context.Background(), BeginInput{ClientIP: "203.0.113.12", UserAgent: "Mozilla/5.0"})
	require.NoError(t, err)

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM custom_registration_fingerprint_reservations")).WithArgs(hash, "reservation-2").WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, store.Abort(context.Background(), reservation))
	require.NoError(t, store.Abort(context.Background(), reservation), "abort is idempotent")

	// An aborted reservation is not committed, so no history/event statements
	// are emitted. The caller may create a new reservation for a retry.
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestStoreMissingMetadataIsNoOp(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer db.Close()
	store := NewStore(db)
	reservation, err := store.Begin(context.Background(), BeginInput{ClientIP: "203.0.113.13"})
	require.NoError(t, err)
	result, err := store.Commit(context.Background(), reservation, 43)
	require.NoError(t, err)
	require.True(t, result.FirstRegistration)
	require.False(t, result.RewardBlocked)
	require.NoError(t, store.Abort(context.Background(), reservation))
	require.NoError(t, mock.ExpectationsWereMet())
}

func newFingerprintSQLMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	return db, mock
}

func stringsContainInsensitive(value, fragment string) bool {
	return len(fragment) > 0 && len(value) >= len(fragment) &&
		strings.Contains(strings.ToLower(value), strings.ToLower(fragment))
}
