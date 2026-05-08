package admin

import (
	"context"
	"log"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type monitoringQueryService interface {
	GetOverview(ctx context.Context) (*service.MonitoringOverview, error)
	GetSummary(ctx context.Context) (*service.MonitoringSummary, error)
	GetGroupModels(ctx context.Context) (*service.MonitoringGroupModels, error)
	GetModelLatency(ctx context.Context) (*service.MonitoringModelLatency, error)
}

type MonitoringHandler struct {
	monitoringService monitoringQueryService
}

func NewMonitoringHandler(monitoringService monitoringQueryService) *MonitoringHandler {
	return &MonitoringHandler{monitoringService: monitoringService}
}

type PublicMonitoringGroup struct {
	GroupID   int64  `json:"group_id"`
	GroupName string `json:"group_name"`
}

type PublicMonitoringOverview struct {
	Groups           []PublicMonitoringGroup    `json:"groups"`
	GroupModels      []service.GroupModelStats  `json:"group_models"`
	ModelLatencies   []service.ModelLatency     `json:"model_latencies"`
	HourlyStats      []service.HourlyStats      `json:"hourly_stats"`
	ModelHourlyStats []service.ModelHourlyStats `json:"model_hourly_stats"`
	TotalRequests    int64                      `json:"total_requests_today"`
	SuccessCount     int64                      `json:"success_count_today"`
	ErrorCount       int64                      `json:"error_count_today"`
	AvgLatencyMs     float64                    `json:"avg_latency_ms_today"`
}

type PublicMonitoringSummary struct {
	Groups        []PublicMonitoringGroup `json:"groups"`
	HourlyStats   []service.HourlyStats   `json:"hourly_stats"`
	TotalRequests int64                   `json:"total_requests_today"`
	SuccessCount  int64                   `json:"success_count_today"`
	ErrorCount    int64                   `json:"error_count_today"`
	AvgLatencyMs  float64                 `json:"avg_latency_ms_today"`
}

func (h *MonitoringHandler) GetPublicOverview(c *gin.Context) {
	overview, err := h.monitoringService.GetOverview(c.Request.Context())
	if err != nil {
		log.Printf("[Monitoring] GetPublicOverview failed: %v", err)
		response.InternalError(c, "Failed to get monitoring data")
		return
	}
	response.Success(c, toPublicMonitoringOverview(overview))
}

func (h *MonitoringHandler) GetPublicSummary(c *gin.Context) {
	data, err := h.monitoringService.GetSummary(c.Request.Context())
	if err != nil {
		log.Printf("[Monitoring] GetPublicSummary failed: %v", err)
		response.InternalError(c, "Failed to get summary data")
		return
	}
	response.Success(c, toPublicMonitoringSummary(data))
}

func (h *MonitoringHandler) GetOverview(c *gin.Context) {
	h.GetPublicOverview(c)
}

func (h *MonitoringHandler) GetSummary(c *gin.Context) {
	h.GetPublicSummary(c)
}

func (h *MonitoringHandler) GetGroupModels(c *gin.Context) {
	data, err := h.monitoringService.GetGroupModels(c.Request.Context())
	if err != nil {
		log.Printf("[Monitoring] GetGroupModels failed: %v", err)
		response.InternalError(c, "Failed to get group model data")
		return
	}
	response.Success(c, data)
}

func toPublicMonitoringOverview(overview *service.MonitoringOverview) *PublicMonitoringOverview {
	if overview == nil {
		return &PublicMonitoringOverview{}
	}

	return &PublicMonitoringOverview{
		Groups:           toPublicMonitoringGroups(overview.Groups),
		GroupModels:      overview.GroupModels,
		ModelLatencies:   overview.ModelLatencies,
		HourlyStats:      overview.HourlyStats,
		ModelHourlyStats: overview.ModelHourlyStats,
		TotalRequests:    overview.TotalRequests,
		SuccessCount:     overview.SuccessCount,
		ErrorCount:       overview.ErrorCount,
		AvgLatencyMs:     overview.AvgLatencyMs,
	}
}

func toPublicMonitoringSummary(summary *service.MonitoringSummary) *PublicMonitoringSummary {
	if summary == nil {
		return &PublicMonitoringSummary{}
	}

	return &PublicMonitoringSummary{
		Groups:        toPublicMonitoringGroups(summary.Groups),
		HourlyStats:   summary.HourlyStats,
		TotalRequests: summary.TotalRequests,
		SuccessCount:  summary.SuccessCount,
		ErrorCount:    summary.ErrorCount,
		AvgLatencyMs:  summary.AvgLatencyMs,
	}
}

func toPublicMonitoringGroups(groups []service.GroupHealth) []PublicMonitoringGroup {
	if len(groups) == 0 {
		return nil
	}

	publicGroups := make([]PublicMonitoringGroup, 0, len(groups))
	for _, group := range groups {
		publicGroups = append(publicGroups, PublicMonitoringGroup{
			GroupID:   group.GroupID,
			GroupName: group.GroupName,
		})
	}
	return publicGroups
}

func (h *MonitoringHandler) GetModelLatency(c *gin.Context) {
	data, err := h.monitoringService.GetModelLatency(c.Request.Context())
	if err != nil {
		log.Printf("[Monitoring] GetModelLatency failed: %v", err)
		response.InternalError(c, "Failed to get model latency data")
		return
	}
	response.Success(c, data)
}
