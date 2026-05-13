//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestOpsDashboardOverviewExcludesClientDisconnect499FromRawAndPreagg(t *testing.T) {
	ctx := context.Background()
	repo, groupID, windowStart, windowEnd := seedOpsSuccessWithClientDisconnect499(t)

	raw, err := repo.GetDashboardOverview(ctx, &service.OpsDashboardFilter{
		StartTime: windowStart,
		EndTime:   windowEnd,
		GroupID:   &groupID,
		QueryMode: service.OpsQueryModeRaw,
	})
	require.NoError(t, err)
	assertOpsSingleSuccessNoError(t, raw)

	require.NoError(t, repo.UpsertHourlyMetrics(ctx, windowStart, windowEnd))

	preagg, err := repo.GetDashboardOverview(ctx, &service.OpsDashboardFilter{
		StartTime: windowStart,
		EndTime:   windowEnd,
		GroupID:   &groupID,
		QueryMode: service.OpsQueryModePreagg,
	})
	require.NoError(t, err)
	assertOpsSingleSuccessNoError(t, preagg)
}

func TestOpsTrendAggregatesExcludeClientDisconnect499(t *testing.T) {
	ctx := context.Background()
	repo, groupID, windowStart, windowEnd := seedOpsSuccessWithClientDisconnect499(t)

	windowStats, err := repo.GetWindowStats(ctx, &service.OpsDashboardFilter{
		StartTime: windowStart,
		EndTime:   windowEnd,
		GroupID:   &groupID,
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, windowStats.SuccessCount)
	require.EqualValues(t, 0, windowStats.ErrorCountTotal)

	trend, err := repo.GetThroughputTrend(ctx, &service.OpsDashboardFilter{
		StartTime: windowStart,
		EndTime:   windowEnd,
		GroupID:   &groupID,
	}, 3600)
	require.NoError(t, err)
	require.Len(t, trend.Points, 1)
	require.EqualValues(t, 1, trend.Points[0].RequestCount)
	require.EqualValues(t, 0, trend.Points[0].SwitchCount)

	errorTrend, err := repo.GetErrorTrend(ctx, &service.OpsDashboardFilter{
		StartTime: windowStart,
		EndTime:   windowEnd,
		GroupID:   &groupID,
	}, 3600)
	require.NoError(t, err)
	require.Len(t, errorTrend.Points, 1)
	require.EqualValues(t, 0, errorTrend.Points[0].ErrorCountTotal)
	require.EqualValues(t, 0, errorTrend.Points[0].ErrorCountSLA)

	errorDistribution, err := repo.GetErrorDistribution(ctx, &service.OpsDashboardFilter{
		StartTime: windowStart,
		EndTime:   windowEnd,
		GroupID:   &groupID,
	})
	require.NoError(t, err)
	require.Zero(t, errorDistribution.Total)
	require.Empty(t, errorDistribution.Items)
}

func seedOpsSuccessWithClientDisconnect499(t *testing.T) (*opsRepository, int64, time.Time, time.Time) {
	t.Helper()

	client := testEntClient(t)
	usageRepo := newUsageLogRepositoryWithSQL(client, integrationDB)
	opsRepo := NewOpsRepository(integrationDB).(*opsRepository)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	user := mustCreateUser(t, client, &service.User{Email: "ops-499-user-" + suffix + "@example.com"})
	group := mustCreateGroup(t, client, &service.Group{
		Name:     "ops-499-group-" + suffix,
		Platform: service.PlatformAnthropic,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name:     "ops-499-account-" + suffix,
		Platform: service.PlatformAnthropic,
	})
	mustBindAccountToGroup(t, client, account.ID, group.ID, 1)
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:  user.ID,
		Key:     "sk-ops-499-" + suffix,
		Name:    "ops-499",
		GroupID: &group.ID,
	})

	windowStart := time.Now().UTC().Add(-4 * time.Hour).Truncate(time.Hour)
	windowEnd := windowStart.Add(time.Hour)
	createdAt := windowStart.Add(10 * time.Minute)
	requestID := "ops-499-request-" + suffix

	durationMs := 2200
	firstTokenMs := 180
	responseLatencyMs := 2100

	inserted, err := usageRepo.Create(context.Background(), &service.UsageLog{
		UserID:            user.ID,
		APIKeyID:          apiKey.ID,
		AccountID:         account.ID,
		GroupID:           &group.ID,
		RequestID:         requestID,
		Model:             "claude-sonnet-4-5",
		RequestedModel:    "claude-sonnet-4-5",
		InputTokens:       120,
		OutputTokens:      80,
		TotalCost:         1.23,
		ActualCost:        1.23,
		DurationMs:        &durationMs,
		FirstTokenMs:      &firstTokenMs,
		ResponseLatencyMs: &responseLatencyMs,
		CreatedAt:         createdAt,
	})
	require.NoError(t, err)
	require.True(t, inserted)

	_, err = opsRepo.InsertErrorLog(context.Background(), &service.OpsInsertErrorLogInput{
		RequestID:      requestID,
		UserID:         &user.ID,
		APIKeyID:       &apiKey.ID,
		AccountID:      &account.ID,
		GroupID:        &group.ID,
		Platform:       service.PlatformAnthropic,
		Model:          "claude-sonnet-4-5",
		RequestedModel: "claude-sonnet-4-5",
		ErrorPhase:     "network",
		ErrorType:      "client_disconnected",
		Severity:       "warn",
		StatusCode:     499,
		ErrorMessage:   "client disconnected",
		ErrorSource:    "gateway",
		ErrorOwner:     "client",
		CreatedAt:      createdAt.Add(2 * time.Second),
	})
	require.NoError(t, err)

	return opsRepo, group.ID, windowStart, windowEnd
}

func assertOpsSingleSuccessNoError(t *testing.T, overview *service.OpsDashboardOverview) {
	t.Helper()
	require.NotNil(t, overview)
	require.EqualValues(t, 1, overview.SuccessCount)
	require.EqualValues(t, 0, overview.ErrorCountTotal)
	require.EqualValues(t, 0, overview.ErrorCountSLA)
	require.EqualValues(t, 1, overview.RequestCountTotal)
	require.EqualValues(t, 1, overview.RequestCountSLA)
}
