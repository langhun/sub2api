package rewards

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestOutboxRepositoryEnqueueUsesModuleTypesAndEstablishedIdempotencyInsert(t *testing.T) {
	client, db, mock := newOutboxSQLMock(t)
	repo := NewOutboxRepository(client, db)
	now := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	input := newDeliveryInput(t)

	mock.ExpectQuery("(?s)INSERT INTO reward_deliveries.*ON CONFLICT DO NOTHING").
		WithArgs(input.SourceType, input.SourceID, input.UserID, input.PrizeID, sqlmock.AnyArg(),
			string(input.RewardType), 1.12345679, "", input.RuleVersion, input.IdempotencyKey).
		WillReturnRows(outboxRows(now, DeliveryStatusPending, 0))

	delivery, err := repo.Enqueue(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, int64(41), delivery.ID)
	require.Equal(t, 1.12345679, delivery.RewardValue)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOutboxRepositoryExecuteClaimedKeepsEffectAndDeliveryStateAtomic(t *testing.T) {
	client, db, mock := newOutboxSQLMock(t)
	repo := NewOutboxRepository(client, db)
	now := time.Date(2026, 7, 25, 8, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM reward_deliveries WHERE id = \\$1 AND status = \\$2 FOR UPDATE").
		WithArgs(int64(41), DeliveryStatusDelivering).
		WillReturnRows(outboxRows(now, DeliveryStatusDelivering, 1))
	mock.ExpectExec("UPDATE reward_deliveries").
		WithArgs(DeliveryStatusDelivered, "credited", sqlmock.AnyArg(), int64(41), DeliveryStatusDelivering).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := repo.ExecuteClaimed(context.Background(), 41, func(ctx context.Context, delivery Delivery) (string, error) {
		require.NotNil(t, dbent.TxFromContext(ctx))
		require.Equal(t, int64(41), delivery.ID)
		return "credited", nil
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewProductionBuildsModuleOwnedHandlerAndRuntime(t *testing.T) {
	module, runtime := NewProduction(ProductionDependencies{}, WorkerOptions{})
	require.NotNil(t, module)
	require.NotNil(t, module.Admin)
	require.NotNil(t, module.Rewards)
	require.NotNil(t, module.Outbox)
	require.NotNil(t, module.Runner)
	require.NotNil(t, runtime)
}

func TestSettingsAdapterIncludesCheckinRewardAndMultiplierRanges(t *testing.T) {
	registry := customsettings.NewRegistry(rewardSettingsStore{values: map[string]string{
		"checkin_enabled":               "true",
		"checkin_min_balance":           "1.25",
		"checkin_max_balance":           "4.5",
		"checkin_luck_enabled":          "true",
		"checkin_luck_min_multiplier":   "0.5",
		"checkin_luck_max_multiplier":   "3.5",
		"checkin_blindbox_enabled":      "true",
		"checkin_blindbox_trigger_type": "total",
		"checkin_blindbox_interval":     "7",
	}})
	settings, err := NewRegistrySettingsAdapter(registry).GetActivitySettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1.25, settings.Checkin.MinimumReward)
	require.Equal(t, 4.5, settings.Checkin.MaximumReward)
	require.Equal(t, 0.5, settings.Checkin.MinimumMultiplier)
	require.Equal(t, 3.5, settings.Checkin.MaximumMultiplier)
	require.True(t, settings.Checkin.Enabled)
	require.True(t, settings.Checkin.LuckEnabled)
	require.True(t, settings.Blindbox.Enabled)
	require.Equal(t, "total", settings.Blindbox.TriggerType)
	require.Equal(t, 7, settings.Blindbox.Interval)
}

func TestInvitationCodeGeneratorUsesNarrowCodeFormatPort(t *testing.T) {
	source := &platformCodeGeneratorStub{code: "INV-9"}
	code, err := codeFormatInvitationGenerator{source: source}.GenerateInvitationCode(context.Background())
	require.NoError(t, err)
	require.Equal(t, "INV-9", code)
	require.Equal(t, "invitation", source.codeType)
}

type rewardSettingsStore struct{ values map[string]string }

func (s rewardSettingsStore) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string)
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			values[key] = value
		}
	}
	return values, nil
}

func (rewardSettingsStore) SetMultiple(context.Context, map[string]string) error { return nil }

type platformCodeGeneratorStub struct {
	code     string
	codeType string
}

func (s *platformCodeGeneratorStub) GenerateCode(_ context.Context, codeType string) (string, error) {
	s.codeType = codeType
	return s.code, nil
}

func newDeliveryInput(t *testing.T) CreateDelivery {
	t.Helper()
	prizeID := int64(3)
	snapshot, err := json.Marshal(Snapshot{
		PrizeID: prizeID, PrizeName: "Daily prize", Rarity: RarityCommon, RewardType: RewardTypeBalance,
		RewardValue: 1.123456789, StreakDays: 2,
	})
	require.NoError(t, err)
	return CreateDelivery{
		SourceType: SourceCheckinBlindbox, SourceID: 19, UserID: 23, PrizeID: &prizeID,
		RewardSnapshot: snapshot, RewardType: RewardTypeBalance, RewardValue: 1.123456789,
		RuleVersion: CheckinBlindboxRuleV1, IdempotencyKey: "checkin_blindbox:19",
	}
}

func newOutboxSQLMock(t *testing.T) (*dbent.Client, *sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(driver))
	t.Cleanup(func() { _ = client.Close() })
	return client, db, mock
}

func outboxColumnNames() []string {
	return []string{
		"id", "source_type", "source_id", "user_id", "prize_item_id", "reward_snapshot",
		"reward_type", "reward_value", "reward_detail", "rule_version", "idempotency_key",
		"status", "attempts", "last_error", "next_retry_at", "locked_at", "delivered_at",
		"compensated_at", "created_at", "updated_at",
	}
}

func outboxRows(now time.Time, status string, attempts int) *sqlmock.Rows {
	return sqlmock.NewRows(outboxColumnNames()).AddRow(
		int64(41), SourceCheckinBlindbox, int64(19), int64(23), int64(3), []byte(`{"prize_item_id":3}`),
		string(RewardTypeBalance), 1.123456789, "", CheckinBlindboxRuleV1, "checkin_blindbox:19", status, attempts,
		nil, nil, now, nil, nil, now, now,
	)
}
