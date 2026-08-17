package checkin

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	ippkg "github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// Handler owns the activity check-in HTTP behavior.
type Handler struct {
	service        contract.CheckinService
	blindboxReader contract.BlindboxRecordsReader
	protector      CheckinProtector
}

func NewHandler(service contract.CheckinService, blindboxReader contract.BlindboxRecordsReader, protector ...CheckinProtector) *Handler {
	var configured CheckinProtector
	if len(protector) > 0 {
		configured = protector[0]
	}
	return &Handler{service: service, blindboxReader: blindboxReader, protector: configured}
}

type captchaRequest struct {
	TurnstileToken        string `json:"turnstile_token"`
	TencentCaptchaTicket  string `json:"tencent_captcha_ticket"`
	TencentCaptchaRandstr string `json:"tencent_captcha_randstr"`
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
	var req captchaRequest
	if err := bindOptionalJSON(c, &req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	if err := h.verifyProtection(c, userID, req); err != nil {
		response.ErrorFrom(c, err)
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
	captchaRequest
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
	if err := h.verifyProtection(c, userID, req.captchaRequest); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	result, err := h.service.LuckCheckin(c.Request.Context(), userID, req.BetAmount, req.UseMaxBalance)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) verifyProtection(c *gin.Context, userID int64, req captchaRequest) error {
	if h.protector == nil {
		return nil
	}
	err := h.protector.Verify(c.Request.Context(), userID, ippkg.GetTrustedClientIP(c), CaptchaProof{
		TurnstileToken: req.TurnstileToken,
		TencentTicket:  req.TencentCaptchaTicket,
		TencentRandstr: req.TencentCaptchaRandstr,
	})
	if retryAfter, ok := retryAfterFromError(err); ok {
		c.Header("Retry-After", strconv.FormatInt(roundedSeconds(retryAfter), 10))
	}
	return err
}

func bindOptionalJSON(c *gin.Context, target any) error {
	if c.Request == nil || c.Request.Body == nil {
		return nil
	}
	err := c.ShouldBindJSON(target)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
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
