package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestRewardDeliveryRepositoryCreatePending(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	prizeID := int64(3)
	input := service.CreateRewardDelivery{
		SourceType:     " checkin ",
		SourceID:       19,
		UserID:         23,
		PrizeItemID:    &prizeID,
		RewardSnapshot: json.RawMessage(`{"name":"Daily prize"}`),
		RewardType:     " balance ",
		RewardValue:    1.123456789,
		RuleVersion:    " blindbox-v1 ",
		IdempotencyKey: " checkin:19 ",
	}

	mock.ExpectQuery("INSERT INTO reward_deliveries").
		WithArgs("checkin", input.SourceID, input.UserID, input.PrizeItemID, sqlmock.AnyArg(),
			"balance", 1.12345679, "", "blindbox-v1", "checkin:19").
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusPending, 0))

	delivery, err := repo.CreatePending(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, int64(41), delivery.ID)
	require.Equal(t, 1.12345679, delivery.RewardValue)
	require.JSONEq(t, `{"name":"Daily prize"}`, string(delivery.RewardSnapshot))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryCreatePendingReadsConcurrentWinner(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	prizeID := int64(3)
	input := service.CreateRewardDelivery{
		SourceType: "checkin", SourceID: 19, UserID: 23, PrizeItemID: &prizeID,
		RewardSnapshot: json.RawMessage(`{"name":"Daily prize"}`), RewardType: "balance",
		RewardValue: 1.123456789, RuleVersion: "blindbox-v1", IdempotencyKey: "checkin:19",
	}

	mock.ExpectQuery("(?s)INSERT INTO reward_deliveries.*ON CONFLICT DO NOTHING").
		WillReturnRows(sqlmock.NewRows(rewardDeliveryColumnNames()))
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE idempotency_key = \\$1 LIMIT 1").
		WithArgs(input.IdempotencyKey).
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusDelivered, 1))

	delivery, err := repo.CreatePending(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, service.RewardDeliveryStatusDelivered, delivery.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryRejectsNonObjectSnapshot(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)

	_, err := repo.CreatePending(context.Background(), service.CreateRewardDelivery{
		SourceType: "checkin", SourceID: 19, UserID: 23,
		RewardSnapshot: json.RawMessage(`["mutable"]`), RewardType: "balance",
		RuleVersion: "blindbox-v1", IdempotencyKey: "checkin:19",
	})
	require.EqualError(t, err, "reward snapshot must be a JSON object")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryRejectsKeyConflictWithDifferentPayload(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	input := service.CreateRewardDelivery{
		SourceType: "checkin", SourceID: 19, UserID: 999,
		RewardSnapshot: json.RawMessage(`{"name":"Daily prize"}`), RewardType: "balance",
		RewardValue: 1.123456789, RuleVersion: "blindbox-v1", IdempotencyKey: "checkin:19",
	}

	mock.ExpectQuery("INSERT INTO reward_deliveries").
		WillReturnRows(sqlmock.NewRows(rewardDeliveryColumnNames()))
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE idempotency_key = \\$1 LIMIT 1").
		WithArgs(input.IdempotencyKey).
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusPending, 0))

	_, err := repo.CreatePending(context.Background(), input)
	require.ErrorIs(t, err, service.ErrRewardDeliveryIdempotencyConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryRejectsSourceConflictWithDifferentKey(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	prizeID := int64(3)
	input := service.CreateRewardDelivery{
		SourceType: "checkin", SourceID: 19, UserID: 23, PrizeItemID: &prizeID,
		RewardSnapshot: json.RawMessage(`{"name":"Daily prize"}`), RewardType: "balance",
		RewardValue: 1.123456789, RuleVersion: "blindbox-v1", IdempotencyKey: "different-key",
	}

	mock.ExpectQuery("(?s)INSERT INTO reward_deliveries.*ON CONFLICT DO NOTHING").
		WillReturnRows(sqlmock.NewRows(rewardDeliveryColumnNames()))
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE idempotency_key = \\$1 LIMIT 1").
		WithArgs(input.IdempotencyKey).
		WillReturnRows(sqlmock.NewRows(rewardDeliveryColumnNames()))
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE source_type = \\$1 AND source_id = \\$2 LIMIT 1").
		WithArgs(input.SourceType, input.SourceID).
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusPending, 0))

	_, err := repo.CreatePending(context.Background(), input)
	require.ErrorIs(t, err, service.ErrRewardDeliveryIdempotencyConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryClaimDueUsesLockedAtomicTransition(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)

	mock.ExpectQuery("(?s)WITH due AS .*FOR UPDATE SKIP LOCKED.*UPDATE reward_deliveries.*attempts = delivery.attempts \\+ 1").
		WithArgs(service.RewardDeliveryStatusPending, now, 25, service.RewardDeliveryStatusDelivering).
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusDelivering, 1))

	deliveries, err := repo.ClaimDue(context.Background(), now, 25)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	require.Equal(t, 1, deliveries[0].Attempts)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryMarkFailedSchedulesRetryOrTerminalFailure(t *testing.T) {
	t.Run("retry", func(t *testing.T) {
		client, db, mock := newRewardDeliverySQLMock(t)
		repo := NewRewardDeliveryRepository(client, db)
		nextRetry := time.Date(2026, 7, 11, 8, 1, 0, 0, time.UTC)
		mock.ExpectExec("UPDATE reward_deliveries").
			WithArgs(service.RewardDeliveryStatusPending, "temporary", &nextRetry, int64(7), service.RewardDeliveryStatusDelivering).
			WillReturnResult(sqlmock.NewResult(0, 1))
		require.NoError(t, repo.MarkFailed(context.Background(), 7, "temporary", &nextRetry))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("terminal", func(t *testing.T) {
		client, db, mock := newRewardDeliverySQLMock(t)
		repo := NewRewardDeliveryRepository(client, db)
		mock.ExpectExec("UPDATE reward_deliveries").
			WithArgs(service.RewardDeliveryStatusFailed, "permanent", nil, int64(8), service.RewardDeliveryStatusDelivering).
			WillReturnResult(sqlmock.NewResult(0, 1))
		require.NoError(t, repo.MarkFailed(context.Background(), 8, "permanent", nil))
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRewardDeliveryRepositoryRetryAndCompensateFailedDelivery(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db).(service.RewardDeliveryAdminStore)

	mock.ExpectExec("(?s)UPDATE reward_deliveries.*last_error = NULL.*WHERE id = \\$2 AND status = \\$3").
		WithArgs(service.RewardDeliveryStatusPending, int64(11), service.RewardDeliveryStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Retry(context.Background(), 11))

	mock.ExpectExec("(?s)UPDATE reward_deliveries.*compensated_at = NOW\\(\\).*WHERE id = \\$3 AND status = \\$4").
		WithArgs(service.RewardDeliveryStatusCompensated, "manual credit", int64(12), service.RewardDeliveryStatusFailed).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, repo.Compensate(context.Background(), 12, " manual credit "))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryProcessClaimedIsAtomic(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE id = \\$1 AND status = \\$2 FOR UPDATE").
		WithArgs(int64(41), service.RewardDeliveryStatusDelivering).
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusDelivering, 1))
	mock.ExpectExec("UPDATE reward_deliveries").
		WithArgs(service.RewardDeliveryStatusDelivered, "credited", sqlmock.AnyArg(), int64(41), service.RewardDeliveryStatusDelivering).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ProcessClaimed(context.Background(), 41, func(ctx context.Context, delivery service.RewardDelivery) (string, error) {
		require.NotNil(t, dbent.TxFromContext(ctx))
		require.Equal(t, int64(41), delivery.ID)
		return "credited", nil
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryProcessClaimedRollsBackApplyFailure(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	now := time.Date(2026, 7, 11, 8, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE id = \\$1 AND status = \\$2 FOR UPDATE").
		WithArgs(int64(41), service.RewardDeliveryStatusDelivering).
		WillReturnRows(rewardDeliveryRows(now, service.RewardDeliveryStatusDelivering, 1))
	mock.ExpectRollback()

	applyErr := errors.New("credit failed")
	err := repo.ProcessClaimed(context.Background(), 41, func(ctx context.Context, delivery service.RewardDelivery) (string, error) {
		require.NotNil(t, dbent.TxFromContext(ctx))
		return "", applyErr
	})
	require.ErrorIs(t, err, applyErr)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryRepositoryRecoverStale(t *testing.T) {
	client, db, mock := newRewardDeliverySQLMock(t)
	repo := NewRewardDeliveryRepository(client, db)
	staleBefore := time.Date(2026, 7, 11, 7, 55, 0, 0, time.UTC)
	nextRetry := staleBefore.Add(time.Minute)
	mock.ExpectExec("(?s)UPDATE reward_deliveries.*locked_at < \\$5").
		WithArgs(service.RewardDeliveryStatusPending, "delivery lock expired and was recovered", nextRetry,
			service.RewardDeliveryStatusDelivering, staleBefore).
		WillReturnResult(sqlmock.NewResult(0, 3))

	count, err := repo.RecoverStale(context.Background(), staleBefore, nextRetry)
	require.NoError(t, err)
	require.Equal(t, 3, count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRewardDeliveryFilterWhereUsesBoundParameters(t *testing.T) {
	userID := int64(12)
	where, args := rewardDeliveryFilterWhere(service.RewardDeliveryFilter{
		Status: " failed ", SourceType: " checkin ", UserID: &userID,
	})
	require.Equal(t, " WHERE status = $1 AND source_type = $2 AND user_id = $3", where)
	require.Equal(t, []any{"failed", "checkin", userID}, args)

	where, args = rewardDeliveryFilterWhere(service.RewardDeliveryFilter{})
	require.Empty(t, where)
	require.Empty(t, args)
}

func newRewardDeliverySQLMock(t *testing.T) (*dbent.Client, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })
	return client, db, mock
}

func rewardDeliveryColumnNames() []string {
	return []string{
		"id", "source_type", "source_id", "user_id", "prize_item_id", "reward_snapshot",
		"reward_type", "reward_value", "reward_detail", "rule_version", "idempotency_key",
		"status", "attempts", "last_error", "next_retry_at", "locked_at", "delivered_at",
		"compensated_at", "created_at", "updated_at",
	}
}

func rewardDeliveryRows(now time.Time, status string, attempts int) *sqlmock.Rows {
	return sqlmock.NewRows(rewardDeliveryColumnNames()).AddRow(
		int64(41), "checkin", int64(19), int64(23), int64(3), []byte(`{"name":"Daily prize"}`),
		"balance", 1.123456789, "", "blindbox-v1", "checkin:19", status, attempts,
		nil, nil, now, nil, nil, now, now,
	)
}
