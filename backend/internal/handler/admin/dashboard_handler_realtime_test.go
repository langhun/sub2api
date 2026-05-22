package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type dashboardRealtimeUsageRepoStub struct {
	service.UsageLogRepository
	stats *usagestats.DashboardStats
}

func (s *dashboardRealtimeUsageRepoStub) GetDashboardStats(_ context.Context) (*usagestats.DashboardStats, error) {
	if s.stats == nil {
		return &usagestats.DashboardStats{}, nil
	}
	return s.stats, nil
}

type dashboardMonitoringSummaryStub struct {
	summary *service.MonitoringSummary
	err     error
}

func (s *dashboardMonitoringSummaryStub) GetSummary(ctx context.Context) (*service.MonitoringSummary, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.summary, nil
}

func TestDashboardHandler_GetRealtimeMetrics_UsesMonitoringSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &dashboardRealtimeUsageRepoStub{
		stats: &usagestats.DashboardStats{
			ActiveUsers:       3,
			Rpm:               2,
			AverageDurationMs: 9.9,
			TotalAccounts:     10,
			ErrorAccounts:     1,
		},
	}
	dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
	handler := NewDashboardHandler(dashboardSvc, nil)
	handler.SetMonitoringService(&dashboardMonitoringSummaryStub{
		summary: &service.MonitoringSummary{
			TotalRequests: 120,
			ErrorCount:    30,
			AvgLatencyMs:  321.5,
			HourlyStats: []service.HourlyStats{
				{Hour: "2026-05-22T10:00:00Z", Total: 360, Success: 300},
			},
		},
	})

	router := gin.New()
	router.GET("/admin/dashboard/realtime", handler.GetRealtimeMetrics)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/realtime", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.Equal(t, float64(120), data["active_requests"])
	require.Equal(t, float64(6), data["requests_per_minute"])
	require.Equal(t, 321.5, data["average_response_time"])
	require.InDelta(t, 0.25, data["error_rate"], 1e-9)

	// Verify response is no longer fixed mock payload.
	require.NotEqual(t, float64(0), data["active_requests"])
	require.NotEqual(t, float64(0), data["requests_per_minute"])
	require.NotEqual(t, float64(0), data["average_response_time"])
	require.NotEqual(t, float64(0), data["error_rate"])
}

func TestDashboardHandler_GetRealtimeMetrics_FallsBackToDashboardStatsWhenMonitoringUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &dashboardRealtimeUsageRepoStub{
		stats: &usagestats.DashboardStats{
			ActiveUsers:       11,
			Rpm:               17,
			AverageDurationMs: 88.5,
			TotalAccounts:     4,
			ErrorAccounts:     1,
		},
	}
	dashboardSvc := service.NewDashboardService(repo, nil, nil, nil)
	handler := NewDashboardHandler(dashboardSvc, nil)
	handler.SetMonitoringService(&dashboardMonitoringSummaryStub{
		err: errors.New("monitoring unavailable"),
	})

	router := gin.New()
	router.GET("/admin/dashboard/realtime", handler.GetRealtimeMetrics)

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/realtime", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)

	var data map[string]any
	require.NoError(t, json.Unmarshal(resp.Data, &data))
	require.Equal(t, float64(11), data["active_requests"])
	require.Equal(t, float64(17), data["requests_per_minute"])
	require.Equal(t, 88.5, data["average_response_time"])
	require.InDelta(t, 0.25, data["error_rate"], 1e-9)
}
