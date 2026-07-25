package redpacket

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Handler is route-neutral. Runtime binds these methods under the existing
// authenticated /redpacket prefix, preserving the legacy public endpoints.
type Handler struct{ service Service }

func NewHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c *gin.Context) {
	userID, ok := activityUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var request struct {
		TotalAmount float64 `json:"total_amount" binding:"required,gt=0"`
		Count       int     `json:"count" binding:"required,gt=0"`
		Type        Type    `json:"redpacket_type"`
		Memo        *string `json:"memo"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.Type == "" {
		request.Type = TypeEqual
	}
	executeRedPacketIdempotent(c, "redpacket_create", request, func(ctx context.Context) (any, error) {
		return h.ready().Create(ctx, CreateRequest{SenderID: userID, TotalAmount: request.TotalAmount,
			Count: request.Count, Type: request.Type, Memo: request.Memo, IdempotencyKey: c.GetHeader("Idempotency-Key")})
	})
}

func (h *Handler) Claim(c *gin.Context) {
	userID, ok := activityUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var request struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	executeRedPacketIdempotent(c, "redpacket_claim", request, func(ctx context.Context) (any, error) {
		return h.ready().Claim(ctx, ClaimRequest{UserID: userID, Code: request.Code, IdempotencyKey: c.GetHeader("Idempotency-Key")})
	})
}

func (h *Handler) Detail(c *gin.Context) {
	userID, ok := activityUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	packet, claims, err := h.ready().GetForParticipant(c.Request.Context(), userID, id)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"redpacket": packet, "claims": claims})
}

func (h *Handler) Mine(c *gin.Context) {
	userID, ok := activityUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	page, pageSize := redPacketPagination(c)
	role := c.DefaultQuery("role", "sent")
	var (
		items []RedPacket
		total int
		err   error
	)
	switch role {
	case "sent":
		items, total, err = h.ready().ListCreatedBy(c.Request.Context(), userID, page, pageSize)
	case "received":
		items, total, err = h.ready().ListClaimedBy(c.Request.Context(), userID, page, pageSize)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be sent or received"})
		return
	}
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

// ListAll is an admin-only route adapter. The route registrar must apply its
// existing admin authorization middleware before binding this method.
func (h *Handler) ListAll(c *gin.Context) {
	page, pageSize := redPacketPagination(c)
	items, total, err := h.ready().ListAll(c.Request.Context(), page, pageSize)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *Handler) ready() Service {
	if h == nil || h.service == nil {
		return unavailableService{}
	}
	return h.service
}

func activityUserID(c *gin.Context) (int64, bool) {
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok && subject.UserID > 0 {
		return subject.UserID, true
	}
	if value, exists := c.Get("user_id"); exists {
		switch value := value.(type) {
		case int64:
			return value, value > 0
		case float64:
			return int64(value), value > 0
		}
	}
	return 0, false
}

func redPacketPagination(c *gin.Context) (int, int) {
	page, pageSize := 1, 20
	if value, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && value > 0 {
		page = value
	}
	if value, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && value > 0 && value <= 100 {
		pageSize = value
	}
	return page, pageSize
}

func writeHTTPError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrUnavailable):
		status = http.StatusServiceUnavailable
	case errors.Is(err, ErrDisabled):
		status = http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrDetailForbidden):
		status = http.StatusForbidden
	case errors.Is(err, ErrInvalidAmount), errors.Is(err, ErrInvalidCount), errors.Is(err, ErrInvalidType), errors.Is(err, ErrAmountTooSmall),
		errors.Is(err, ErrExpired), errors.Is(err, ErrExhausted), errors.Is(err, ErrAlreadyClaimed), errors.Is(err, ErrSelfClaim), errors.Is(err, ErrInvalidPagination), errors.Is(err, ErrInsufficient):
		status = http.StatusBadRequest
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func executeRedPacketIdempotent(c *gin.Context, scope string, payload any, execute func(context.Context) (any, error)) {
	coordinator := service.DefaultIdempotencyCoordinator()
	if coordinator == nil {
		result, err := execute(c.Request.Context())
		if err != nil {
			writeHTTPError(c, err)
			return
		}
		response.Success(c, result)
		return
	}
	actorScope := "user:0"
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok {
		actorScope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}
	result, err := coordinator.Execute(c.Request.Context(), service.IdempotencyExecuteOptions{
		Scope: scope, ActorScope: actorScope, Method: c.Request.Method, Route: c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"), Payload: payload, RequireKey: true, TTL: 24 * time.Hour,
	}, execute)
	if err != nil {
		if infraerrors.Code(err) == infraerrors.Code(service.ErrIdempotencyStoreUnavail) {
			service.RecordIdempotencyStoreUnavailable(c.FullPath(), scope, "handler_fail_close")
			logger.LegacyPrintf("activity.redpacket.idempotency", "red-packet idempotency store unavailable: route=%s", c.FullPath())
		}
		if retryAfter := service.RetryAfterSecondsFromError(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		if isRedPacketError(err) {
			writeHTTPError(c, err)
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, result.Data)
}

func isRedPacketError(err error) bool {
	return errors.Is(err, ErrUnavailable) || errors.Is(err, ErrDisabled) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrDetailForbidden) ||
		errors.Is(err, ErrInvalidAmount) || errors.Is(err, ErrInvalidCount) || errors.Is(err, ErrInvalidType) || errors.Is(err, ErrAmountTooSmall) ||
		errors.Is(err, ErrExpired) || errors.Is(err, ErrExhausted) || errors.Is(err, ErrAlreadyClaimed) || errors.Is(err, ErrSelfClaim) ||
		errors.Is(err, ErrInvalidPagination) || errors.Is(err, ErrInsufficient)
}

type unavailableService struct{}

func (unavailableService) Create(context.Context, CreateRequest) (*RedPacket, error) {
	return nil, ErrUnavailable
}
func (unavailableService) Claim(context.Context, ClaimRequest) (*Claim, error) {
	return nil, ErrUnavailable
}
func (unavailableService) RefundExpired(context.Context) (ExpiryRunResult, error) {
	return ExpiryRunResult{}, ErrUnavailable
}
func (unavailableService) Get(context.Context, int64) (*RedPacket, error) { return nil, ErrUnavailable }
func (unavailableService) GetForParticipant(context.Context, int64, int64) (*RedPacket, []Claim, error) {
	return nil, nil, ErrUnavailable
}
func (unavailableService) ListCreatedBy(context.Context, int64, int, int) ([]RedPacket, int, error) {
	return nil, 0, ErrUnavailable
}
func (unavailableService) ListClaimedBy(context.Context, int64, int, int) ([]RedPacket, int, error) {
	return nil, 0, ErrUnavailable
}
func (unavailableService) ListAll(context.Context, int, int) ([]RedPacket, int, error) {
	return nil, 0, ErrUnavailable
}
