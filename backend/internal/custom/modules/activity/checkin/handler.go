package checkin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// Handler owns the activity check-in HTTP behavior.
type Handler struct {
	service        contract.CheckinService
	blindboxReader contract.BlindboxRecordsReader
}

func NewHandler(service contract.CheckinService, blindboxReader contract.BlindboxRecordsReader) *Handler {
	return &Handler{service: service, blindboxReader: blindboxReader}
}

func (h *Handler) Checkin(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "activity check-in is unavailable")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	result, err := h.service.Checkin(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type luckCheckinRequest struct {
	BetAmount     float64 `json:"bet_amount" binding:"required,gt=0"`
	UseMaxBalance bool    `json:"use_max_balance"`
}

func (h *Handler) LuckCheckin(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "activity check-in is unavailable")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	var req luckCheckinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: bet_amount is required and must be greater than 0")
		return
	}
	result, err := h.service.LuckCheckin(c.Request.Context(), userID, req.BetAmount, req.UseMaxBalance)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) GetStatus(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "activity check-in is unavailable")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	status, err := h.service.GetStatus(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *Handler) GetCalendar(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "activity check-in is unavailable")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	calendar, err := h.service.GetCalendar(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, calendar)
}

func (h *Handler) GetBlindboxRecords(c *gin.Context) {
	if h == nil || h.blindboxReader == nil {
		response.Error(c, http.StatusServiceUnavailable, "activity check-in is unavailable")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	page, pageSize := checkinPagination(c)
	records, err := h.blindboxReader.GetUserRecords(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, records)
}

func currentUserID(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		return 0, false
	}
	return subject.UserID, true
}

func checkinPagination(c *gin.Context) (int, int) {
	page, pageSize := 1, 20
	if value, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && value > 0 {
		page = value
	}
	if value, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && value > 0 && value <= 100 {
		pageSize = value
	}
	return page, pageSize
}
