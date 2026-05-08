package handler

import (
	"math"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type LeaderboardHandler struct {
	leaderboardService *service.LeaderboardService
	checkinService     *service.CheckinService
	settingService     *service.SettingService
}

func NewLeaderboardHandler(leaderboardService *service.LeaderboardService, checkinService *service.CheckinService, settingService *service.SettingService) *LeaderboardHandler {
	return &LeaderboardHandler{
		leaderboardService: leaderboardService,
		checkinService:     checkinService,
		settingService:     settingService,
	}
}

func (h *LeaderboardHandler) GetBalanceLeaderboard(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	includeAdmin := h.settingService != nil && h.settingService.IsLeaderboardIncludeAdminEnabled(c.Request.Context())

	result, err := h.leaderboardService.GetBalanceLeaderboard(c.Request.Context(), page, pageSize, includeAdmin)
	if err != nil {
		response.InternalError(c, "Failed to get balance leaderboard")
		return
	}

	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}

func (h *LeaderboardHandler) GetConsumptionLeaderboard(c *gin.Context) {
	period := c.DefaultQuery("period", "daily")
	if period != "daily" && period != "weekly" && period != "monthly" {
		response.BadRequest(c, "Invalid period, must be daily, weekly or monthly")
		return
	}

	page, pageSize := response.ParsePagination(c)
	includeAdmin := h.settingService != nil && h.settingService.IsLeaderboardIncludeAdminEnabled(c.Request.Context())

	result, err := h.leaderboardService.GetConsumptionLeaderboard(c.Request.Context(), period, page, pageSize, includeAdmin)
	if err != nil {
		response.InternalError(c, "Failed to get consumption leaderboard")
		return
	}

	pages := int(math.Ceil(float64(result.Total) / float64(pageSize)))
	if pages < 1 {
		pages = 1
	}

	response.Success(c, struct {
		Items      []service.LeaderboardEntry     `json:"items"`
		Total      int64                          `json:"total"`
		Page       int                            `json:"page"`
		PageSize   int                            `json:"page_size"`
		Pages      int                            `json:"pages"`
		Summary    *service.LeaderboardSummary    `json:"summary,omitempty"`
		ChartItems []service.LeaderboardChartItem `json:"chart_items,omitempty"`
	}{
		Items:      result.Entries,
		Total:      result.Total,
		Page:       page,
		PageSize:   pageSize,
		Pages:      pages,
		Summary:    result.Summary,
		ChartItems: result.ChartItems,
	})
}

func (h *LeaderboardHandler) GetCheckinLeaderboard(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	includeAdmin := h.settingService != nil && h.settingService.IsLeaderboardIncludeAdminEnabled(c.Request.Context())

	result, err := h.leaderboardService.GetCheckinLeaderboard(c.Request.Context(), page, pageSize, includeAdmin)
	if err != nil {
		response.InternalError(c, "Failed to get checkin leaderboard")
		return
	}

	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}
