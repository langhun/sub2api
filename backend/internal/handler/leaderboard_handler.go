package handler

import (
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

func (h *LeaderboardHandler) enabled(c *gin.Context, enabled func(service.BalanceFeatureSettings) bool) bool {
	settings := h.settingService.GetLeaderboardSettings(c.Request.Context())
	if !settings.LeaderboardEnabled || !enabled(settings) {
		response.NotFound(c, "Leaderboard is disabled")
		return false
	}
	return true
}

func (h *LeaderboardHandler) GetBalanceLeaderboard(c *gin.Context) {
	if !h.enabled(c, func(s service.BalanceFeatureSettings) bool { return s.LeaderboardBalanceEnabled }) {
		return
	}
	page, pageSize := response.ParsePagination(c)

	result, err := h.leaderboardService.GetBalanceLeaderboard(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalError(c, "Failed to get balance leaderboard")
		return
	}

	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}

func (h *LeaderboardHandler) GetConsumptionLeaderboard(c *gin.Context) {
	if !h.enabled(c, func(s service.BalanceFeatureSettings) bool { return s.LeaderboardConsumptionEnabled }) {
		return
	}
	period := c.DefaultQuery("period", "daily")
	if period != "daily" && period != "weekly" && period != "monthly" {
		response.BadRequest(c, "Invalid period, must be daily, weekly or monthly")
		return
	}

	page, pageSize := response.ParsePagination(c)

	result, err := h.leaderboardService.GetConsumptionLeaderboard(c.Request.Context(), period, page, pageSize)
	if err != nil {
		response.InternalError(c, "Failed to get consumption leaderboard")
		return
	}

	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}

func (h *LeaderboardHandler) GetCheckinLeaderboard(c *gin.Context) {
	if !h.enabled(c, func(s service.BalanceFeatureSettings) bool { return s.LeaderboardCheckinEnabled }) {
		return
	}
	page, pageSize := response.ParsePagination(c)

	result, err := h.leaderboardService.GetCheckinLeaderboard(c.Request.Context(), page, pageSize)
	if err != nil {
		response.InternalError(c, "Failed to get checkin leaderboard")
		return
	}

	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}

func (h *LeaderboardHandler) GetTransferLeaderboard(c *gin.Context) {
	if !h.enabled(c, func(s service.BalanceFeatureSettings) bool { return s.LeaderboardTransferEnabled }) {
		return
	}
	period := c.DefaultQuery("period", "daily")
	if period != "daily" && period != "weekly" && period != "monthly" {
		response.BadRequest(c, "Invalid period, must be daily, weekly or monthly")
		return
	}
	page, pageSize := response.ParsePagination(c)
	result, err := h.leaderboardService.GetTransferLeaderboard(c.Request.Context(), period, page, pageSize)
	if err != nil {
		response.InternalError(c, "Failed to get transfer leaderboard")
		return
	}
	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}
