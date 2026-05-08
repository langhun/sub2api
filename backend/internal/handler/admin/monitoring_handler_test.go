package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type monitoringServiceStub struct {
	overview    *service.MonitoringOverview
	overviewErr error
	summary     *service.MonitoringSummary
	summaryErr  error
}

func (s *monitoringServiceStub) GetOverview(ctx context.Context) (*service.MonitoringOverview, error) {
	return s.overview, s.overviewErr
}

func (s *monitoringServiceStub) GetSummary(ctx context.Context) (*service.MonitoringSummary, error) {
	return s.summary, s.summaryErr
}

func (s *monitoringServiceStub) GetGroupModels(ctx context.Context) (*service.MonitoringGroupModels, error) {
	return nil, errors.New("unexpected GetGroupModels call")
}

func (s *monitoringServiceStub) GetModelLatency(ctx context.Context) (*service.MonitoringModelLatency, error) {
	return nil, errors.New("unexpected GetModelLatency call")
}

func TestMonitoringHandler_GetPublicSummary_StripsSensitiveFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewMonitoringHandler(&monitoringServiceStub{
		summary: &service.MonitoringSummary{
			Groups: []service.GroupHealth{
				{
					GroupID:        7,
					GroupName:      "public-group",
					TotalAccounts:  12,
					ActiveAccounts: 10,
					ErrorAccounts:  1,
					RateLimited:    2,
					Overload:       3,
					Disabled:       4,
				},
			},
			ErrorAccounts: []service.ErrorAccount{
				{
					AccountID:     42,
					AccountName:   "internal-account",
					GroupName:     "public-group",
					Status:        "error",
					ErrorMessage:  "upstream auth failed",
					RateLimitedAt: "2026-05-08 08:00:00",
				},
			},
			HourlyStats: []service.HourlyStats{
				{Hour: "2026-05-08T08:00:00Z", Total: 100, Success: 90},
			},
			TotalRequests: 100,
			SuccessCount:  90,
			ErrorCount:    10,
			AvgLatencyMs:  321.5,
		},
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/summary", nil)

	handler.GetPublicSummary(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.NotContains(t, data, "error_accounts")

	groups, ok := data["groups"].([]any)
	require.True(t, ok)
	require.Len(t, groups, 1)

	group, ok := groups[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(7), group["group_id"])
	require.Equal(t, "public-group", group["group_name"])
	require.NotContains(t, group, "total_accounts")
	require.NotContains(t, group, "active_accounts")
	require.NotContains(t, group, "error_accounts")
	require.NotContains(t, group, "rate_limited")
	require.NotContains(t, group, "overload")
	require.NotContains(t, group, "disabled")

	require.Equal(t, float64(100), data["total_requests_today"])
	require.Equal(t, float64(90), data["success_count_today"])
	require.Equal(t, float64(10), data["error_count_today"])
	require.Equal(t, 321.5, data["avg_latency_ms_today"])
}

func TestMonitoringHandler_GetPublicOverview_StripsSensitiveFieldsAndKeepsPublicCharts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewMonitoringHandler(&monitoringServiceStub{
		overview: &service.MonitoringOverview{
			Groups: []service.GroupHealth{
				{
					GroupID:        3,
					GroupName:      "group-a",
					TotalAccounts:  8,
					ActiveAccounts: 6,
					ErrorAccounts:  1,
				},
			},
			GroupModels: []service.GroupModelStats{
				{
					GroupID:      3,
					GroupName:    "group-a",
					Model:        "gpt-4.1",
					RequestCount: 12,
					SuccessCount: 11,
					ErrorCount:   1,
					AvgLatencyMs: 800,
				},
			},
			ModelLatencies: []service.ModelLatency{
				{
					Model:           "gpt-4.1",
					RequestCount:    12,
					SuccessCount:    11,
					ErrorCount:      1,
					AvgLatencyMs:    800,
					P95LatencyMs:    1200,
					P99LatencyMs:    1500,
					AvgFirstTokenMs: 220,
				},
			},
			ErrorAccounts: []service.ErrorAccount{
				{
					AccountID:    88,
					AccountName:  "secret",
					ErrorMessage: "sensitive detail",
				},
			},
			HourlyStats: []service.HourlyStats{
				{Hour: "2026-05-08T08:00:00Z", Total: 20, Success: 18},
			},
			ModelHourlyStats: []service.ModelHourlyStats{
				{GroupID: 3, Model: "gpt-4.1", Hour: "2026-05-08T08:00:00Z", Total: 2, Success: 2},
			},
			TotalRequests: 20,
			SuccessCount:  18,
			ErrorCount:    2,
			AvgLatencyMs:  456.7,
		},
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/overview", nil)

	handler.GetPublicOverview(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.NotContains(t, data, "error_accounts")

	groupModels, ok := data["group_models"].([]any)
	require.True(t, ok)
	require.Len(t, groupModels, 1)

	modelLatencies, ok := data["model_latencies"].([]any)
	require.True(t, ok)
	require.Len(t, modelLatencies, 1)

	modelHourlyStats, ok := data["model_hourly_stats"].([]any)
	require.True(t, ok)
	require.Len(t, modelHourlyStats, 1)
}
