package channelmonitorv2activeprobes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingsStub struct {
	values map[string]string
	err    error
	writes map[string]string
}

func (s *settingsStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.values, nil
}

func (s *settingsStub) SetMultiple(_ context.Context, values map[string]string) error {
	if s.err != nil {
		return s.err
	}
	s.writes = values
	return nil
}

func enabledSettings() *settingsStub {
	return &settingsStub{values: map[string]string{settingKeyEnabled: "true"}}
}

type schedulerStub struct{ calls int }

func (s *schedulerStub) TriggerScheduledChecks() { s.calls++ }

type runtimeReaderStub struct{ runtime service.ChannelMonitorRuntime }

func (s runtimeReaderStub) GetChannelMonitorRuntime(context.Context) service.ChannelMonitorRuntime {
	return s.runtime
}

func activeProbeConfig() service.ChannelMonitorV2Config {
	return service.ChannelMonitorV2Config{
		Enabled: true,
		Platforms: []service.ChannelMonitorV2PlatformConfig{
			{Platform: "openai", Enabled: true},
			{Platform: "deepseek", Enabled: true},
		},
		HealthThresholds: service.DefaultChannelMonitorV2HealthThresholds(),
	}
}

func activeProbeFilter(now time.Time) service.ChannelMonitorV2Filter {
	return service.ChannelMonitorV2Filter{
		Start:  now.Add(-90 * time.Minute),
		End:    now,
		Bucket: 5 * time.Minute,
	}
}

func expectFactsQuery(mock sqlmock.Sqlmock, filter service.ChannelMonitorV2Filter, rows *sqlmock.Rows) {
	mock.ExpectQuery(regexp.QuoteMeta("FROM custom_channel_monitor_v2_active_probe_results")).
		WithArgs(filter.Start.UTC(), filter.End.UTC()).
		WillReturnRows(rows)
}

func TestAllowChannelMonitorCheck(t *testing.T) {
	module := NewModule(nil, enabledSettings(), nil, nil)
	v2 := service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV2}

	require.True(t, module.AllowChannelMonitorCheck(context.Background(), v2, service.MonitorCheckModeProbe))
	require.True(t, module.AllowChannelMonitorCheck(context.Background(), v2, service.MonitorCheckModeQuotaProbe))
	require.True(t, module.AllowChannelMonitorCheck(context.Background(), v2, ""), "legacy empty mode is a probe")
	require.False(t, module.AllowChannelMonitorCheck(context.Background(), v2, service.MonitorCheckModeQuota))
	require.False(t, module.AllowChannelMonitorCheck(context.Background(), service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV1}, service.MonitorCheckModeProbe))

	module.settings = &settingsStub{values: map[string]string{settingKeyEnabled: "false"}}
	require.False(t, module.AllowChannelMonitorCheck(context.Background(), v2, service.MonitorCheckModeProbe))
	module.settings = &settingsStub{err: errors.New("settings unavailable")}
	require.False(t, module.AllowChannelMonitorCheck(context.Background(), v2, service.MonitorCheckModeProbe))
}

func TestStartRestoresEnabledV2ScheduleWithoutDuplicatingV1(t *testing.T) {
	scheduler := &schedulerStub{}
	module := NewModule(nil, enabledSettings(), scheduler, runtimeReaderStub{runtime: service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV2}})
	module.Start(context.Background())
	require.Equal(t, 1, scheduler.calls)

	module.runtime = runtimeReaderStub{runtime: service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV1}}
	module.Start(context.Background())
	require.Equal(t, 1, scheduler.calls)

	module.runtime = runtimeReaderStub{runtime: service.ChannelMonitorRuntime{Enabled: false, Mode: service.ChannelMonitorModeV2}}
	module.Start(context.Background())
	require.Equal(t, 1, scheduler.calls)
}

func TestUpdateConfigTriggersExistingTargetsOnlyWhenSwitchTurnsOn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := &settingsStub{values: map[string]string{settingKeyEnabled: "false"}}
	scheduler := &schedulerStub{}
	module := NewModule(nil, settings, scheduler, nil)

	request := func(body string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/config", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		module.updateConfig(ctx)
		return recorder
	}

	recorder := request(`{"enabled":true}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "true", settings.writes[settingKeyEnabled])
	require.Equal(t, 1, scheduler.calls)

	settings.values[settingKeyEnabled] = "true"
	recorder = request(`{"enabled":true}`)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, scheduler.calls, "saving an already-enabled switch must not duplicate probes")
}

func TestObserveChannelMonitorCheckPersistsOnlySanitizedProbeFacts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})

	checkedAt := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	latency := 321
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO custom_channel_monitor_v2_active_probe_results").
		WithArgs(int64(7), "deepseek", "deepseek-chat", service.MonitorStatusOperational, &latency, 60, checkedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	module := NewModule(db, enabledSettings(), nil, nil)
	module.lastCleanupUnix.Store(time.Now().Unix())
	module.ObserveChannelMonitorCheck(context.Background(), service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV2}, &service.ChannelMonitor{
		ID: 7, Provider: " DeepSeek ", CheckMode: service.MonitorCheckModeProbe, IntervalSeconds: 60,
		APIKey: "must-never-be-persisted", Endpoint: "https://example.test/v1",
	}, []*service.CheckResult{{Model: "deepseek-chat", Status: service.MonitorStatusOperational, LatencyMs: &latency, CheckedAt: checkedAt, Message: "must-never-be-persisted"}})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestObserveChannelMonitorCheckSkipsV1AndPureQuota(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	module := NewModule(db, enabledSettings(), nil, nil)
	result := []*service.CheckResult{{Model: "quota", Status: service.MonitorStatusOperational, CheckedAt: time.Now()}}

	module.ObserveChannelMonitorCheck(context.Background(), service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV1}, &service.ChannelMonitor{ID: 1, CheckMode: service.MonitorCheckModeProbe}, result)
	module.ObserveChannelMonitorCheck(context.Background(), service.ChannelMonitorRuntime{Enabled: true, Mode: service.ChannelMonitorModeV2}, &service.ChannelMonitor{ID: 2, CheckMode: service.MonitorCheckModeQuota}, result)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupExpiredFactsIsRateLimited(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	mock.ExpectExec("DELETE FROM custom_channel_monitor_v2_active_probe_results").
		WillReturnResult(sqlmock.NewResult(0, 4))
	module := NewModule(db, enabledSettings(), nil, nil)
	module.now = func() time.Time { return now }

	module.cleanupExpiredFacts()
	module.cleanupExpiredFacts()

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExtendModelsUsesAnyHealthyLatestTargetWithoutTokenUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	filter := activeProbeFilter(now)
	rows := sqlmock.NewRows([]string{"monitor_id", "platform", "model", "status", "latency_ms", "interval_seconds", "checked_at"}).
		AddRow(int64(1), "openai", "gpt-5", service.MonitorStatusFailed, int64(900), 600, now.Add(-5*time.Minute)).
		AddRow(int64(2), "openai", "gpt-5", service.MonitorStatusOperational, int64(300), 600, now.Add(-10*time.Minute))
	expectFactsQuery(mock, filter, rows)

	list := &service.ChannelMonitorV2List[service.ChannelMonitorV2ModelRow]{}
	module := NewModule(db, enabledSettings(), nil, nil)
	module.now = func() time.Time { return now }
	err = module.ExtendChannelMonitorV2Models(context.Background(), filter, activeProbeConfig(), list, false)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	row := list.Items[0]
	require.Equal(t, int64(2), row.Metrics.RequestCount)
	require.Equal(t, int64(1), row.Metrics.SuccessRequests)
	require.Equal(t, int64(1), row.Metrics.ErrorRequests)
	require.Zero(t, row.Metrics.TokenCount)
	require.Zero(t, row.Metrics.CacheRateNumerator)
	require.Zero(t, row.Metrics.CacheRateDenominator)
	require.Equal(t, "healthy", row.Health.Overall)
	require.Equal(t, "unknown", row.Health.Cache)
	require.Nil(t, row.Health.CacheScore)
	require.NotNil(t, row.Health.Score)
	require.Equal(t, float64(100), *row.Health.Score)
	source, ok := row.Extensions[extensionKey].(probeSource)
	require.True(t, ok)
	require.True(t, source.ActiveProbe)
	require.Equal(t, service.MonitorStatusOperational, source.ActiveProbeStatus)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSourceForFactsPreservesDegradedWhenNoTargetIsOperational(t *testing.T) {
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	source := sourceForFacts([]probeFact{
		{MonitorID: 1, MappedModel: "gpt-5", Status: service.MonitorStatusFailed, CheckedAt: now.Add(-time.Minute)},
		{MonitorID: 2, MappedModel: "gpt-5", Status: service.MonitorStatusDegraded, CheckedAt: now},
	}, false)
	require.Equal(t, service.MonitorStatusDegraded, source.ActiveProbeStatus)
	health := activeOnlyHealth(source, service.DefaultChannelMonitorV2HealthThresholds())
	require.Equal(t, "warning", health.Overall)
	require.NotNil(t, health.Score)
	require.Equal(t, float64(50), *health.Score)
}

func TestExtendModelsKeepsFreshStatusOutsideSelectedWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	filter := activeProbeFilter(now)
	rows := sqlmock.NewRows([]string{"monitor_id", "platform", "model", "status", "latency_ms", "interval_seconds", "checked_at"}).
		AddRow(int64(3), "deepseek", "deepseek-chat", service.MonitorStatusOperational, int64(250), 3600, now.Add(-2*time.Hour))
	expectFactsQuery(mock, filter, rows)

	list := &service.ChannelMonitorV2List[service.ChannelMonitorV2ModelRow]{}
	module := NewModule(db, enabledSettings(), nil, nil)
	module.now = func() time.Time { return now }
	err = module.ExtendChannelMonitorV2Models(context.Background(), filter, activeProbeConfig(), list, false)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Zero(t, list.Items[0].Metrics.RequestCount, "out-of-window freshness must not inflate selected-window volume")
	require.Equal(t, "healthy", list.Items[0].Health.Overall)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExtendModelsExpiresStatusAfterThreeProbeIntervals(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	filter := activeProbeFilter(now)
	rows := sqlmock.NewRows([]string{"monitor_id", "platform", "model", "status", "latency_ms", "interval_seconds", "checked_at"}).
		AddRow(int64(8), "deepseek", "deepseek-chat", service.MonitorStatusOperational, int64(250), 60, now.Add(-4*time.Minute))
	expectFactsQuery(mock, filter, rows)

	module := NewModule(db, enabledSettings(), nil, nil)
	module.now = func() time.Time { return now }
	list := &service.ChannelMonitorV2List[service.ChannelMonitorV2ModelRow]{}
	err = module.ExtendChannelMonitorV2Models(context.Background(), filter, activeProbeConfig(), list, false)
	require.NoError(t, err)
	require.Len(t, list.Items, 1)
	require.Equal(t, int64(1), list.Items[0].Metrics.RequestCount, "historical probe volume remains in the selected window")
	require.Equal(t, "unknown", list.Items[0].Health.Overall)
	source, ok := list.Items[0].Extensions[extensionKey].(probeSource)
	require.True(t, ok)
	require.False(t, source.ActiveProbe)
	require.Nil(t, source.LastActiveProbeAt)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExtendModelsMergesRequestsButPreservesPassiveTokenAndCacheData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	now := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC)
	filter := activeProbeFilter(now)
	rows := sqlmock.NewRows([]string{"monitor_id", "platform", "model", "status", "latency_ms", "interval_seconds", "checked_at"}).
		AddRow(int64(4), "openai", "gpt-5", service.MonitorStatusOperational, int64(200), 60, now.Add(-time.Minute))
	expectFactsQuery(mock, filter, rows)

	list := &service.ChannelMonitorV2List[service.ChannelMonitorV2ModelRow]{Items: []service.ChannelMonitorV2ModelRow{{
		Platform: "openai", Model: "gpt-5",
		Metrics: service.ChannelMonitorV2Metric{RequestCount: 10, SuccessRequests: 9, ErrorRequests: 1, TokenCount: 500, InputTokens: 400, OutputTokens: 100, CacheRateNumerator: 50, CacheRateDenominator: 200, CacheRate: .25},
	}}}
	module := NewModule(db, enabledSettings(), nil, nil)
	module.now = func() time.Time { return now }
	err = module.ExtendChannelMonitorV2Models(context.Background(), filter, activeProbeConfig(), list, true)
	require.NoError(t, err)
	row := list.Items[0]
	require.Equal(t, int64(11), row.Metrics.RequestCount)
	require.Equal(t, int64(500), row.Metrics.TokenCount)
	require.Equal(t, int64(50), row.Metrics.CacheRateNumerator)
	require.Equal(t, int64(200), row.Metrics.CacheRateDenominator)
	require.Equal(t, .25, row.Metrics.CacheRate)
	source, ok := row.Extensions[extensionKey].(probeSource)
	require.True(t, ok)
	require.True(t, source.PassiveTraffic)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExtensionsDoNotInventGroupAttribution(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, db.Close())
	})
	module := NewModule(db, enabledSettings(), nil, nil)
	filter := activeProbeFilter(time.Now().UTC())
	filter.GroupIDs = []int64{9}

	require.NoError(t, module.ExtendChannelMonitorV2Dimensions(context.Background(), filter, activeProbeConfig(), &service.ChannelMonitorV2Dimensions{}))
	require.NoError(t, module.ExtendChannelMonitorV2Snapshot(context.Background(), filter, activeProbeConfig(), &service.ChannelMonitorV2Snapshot{}, false))
	require.NoError(t, module.ExtendChannelMonitorV2Models(context.Background(), filter, activeProbeConfig(), &service.ChannelMonitorV2List[service.ChannelMonitorV2ModelRow]{}, false))
	filter.GroupIDs = nil
	require.NoError(t, module.ExtendChannelMonitorV2Matrix(context.Background(), filter, activeProbeConfig(), service.ChannelMonitorV2GroupByPlatformGroup, &service.ChannelMonitorV2Matrix{}, false))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestMetricForFactsDoesNotSynthesizeUsage(t *testing.T) {
	latency := int64(120)
	metric := metricForFacts([]probeFact{{Status: service.MonitorStatusOperational, LatencyMs: &latency}}, 1, false)
	require.Equal(t, int64(1), metric.RequestCount)
	require.Equal(t, int64(1), metric.Duration.SampleCount)
	require.Zero(t, metric.InputTokens)
	require.Zero(t, metric.OutputTokens)
	require.Zero(t, metric.TokenCount)
	require.Zero(t, metric.CacheRateNumerator)
	require.Zero(t, metric.CacheRateDenominator)
}

var _ Settings = (*settingsStub)(nil)
