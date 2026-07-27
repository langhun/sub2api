package walletextension

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// UserHandler serves the direct-transfer user API.
type UserHandler struct {
	service     *Service
	idempotency platform.IdempotencyCoordinator
}

// NewUserHandler constructs the direct-transfer HTTP adapter.
func NewUserHandler(directTransferService *Service) *UserHandler {
	return NewUserHandlerWithIdempotency(directTransferService, nil)
}

// NewUserHandlerWithIdempotency injects the generic platform coordinator used
// by direct-transfer mutations.
func NewUserHandlerWithIdempotency(directTransferService *Service, coordinator platform.IdempotencyCoordinator) *UserHandler {
	return &UserHandler{service: directTransferService, idempotency: coordinator}
}

type directTransferRequest struct {
	ReceiverID int64   `json:"receiver_id" binding:"required"`
	Amount     float64 `json:"amount" binding:"required,gt=0"`
	Memo       *string `json:"memo"`
}

// Transfer commits one authenticated direct transfer.
func (h *UserHandler) Transfer(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct transfer is unavailable"})
		return
	}
	senderID := directTransferUserID(c)
	if senderID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var request directTransferRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	executeDirectTransferIdempotent(c, h.idempotency, request, func(ctx context.Context) (DirectTransferRecord, error) {
		return h.service.Transfer(ctx, senderID, DirectTransferRequest{
			ReceiverID: request.ReceiverID, Amount: request.Amount, Memo: request.Memo, IdempotencyKey: c.GetHeader("Idempotency-Key"),
		})
	})
}

// Preview returns the direct-transfer fee and daily-limit result.
func (h *UserHandler) Preview(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct transfer is unavailable"})
		return
	}
	senderID := directTransferUserID(c)
	if senderID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var request directTransferRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.service.Preview(c.Request.Context(), senderID, request.ReceiverID, request.Amount)
	if err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ResolveRecipient resolves an eligible recipient by exact identity.
func (h *UserHandler) ResolveRecipient(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct transfer is unavailable"})
		return
	}
	requesterID := directTransferUserID(c)
	if requesterID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.service.ResolveRecipient(c.Request.Context(), requesterID, c.Query("query"))
	if err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"receiver_id": result.Account.ID, "receiver_display": result.DisplayName})
}

// SearchRecipients lists masked eligible recipient candidates.
func (h *UserHandler) SearchRecipients(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct transfer is unavailable"})
		return
	}
	requesterID := directTransferUserID(c)
	if requesterID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	results, err := h.service.SearchRecipients(c.Request.Context(), requesterID, c.Query("query"))
	if err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, results)
}

// History returns an authenticated user's direct-transfer history.
func (h *UserHandler) History(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct transfer is unavailable"})
		return
	}
	accountID := directTransferUserID(c)
	if accountID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	items, total, err := h.service.ListHistory(c.Request.Context(), DirectTransferHistoryQuery{AccountID: accountID, Role: c.DefaultQuery("role", "all"), Page: page, PageSize: pageSize})
	if err != nil {
		writeDirectTransferError(c, err)
		return
	}
	page, pageSize = normalizeDirectTransferPage(page, pageSize)
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

// Stats returns the authenticated user's direct-transfer aggregates.
func (h *UserHandler) Stats(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "direct transfer is unavailable"})
		return
	}
	accountID := directTransferUserID(c)
	if accountID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	result, err := h.service.GetStats(c.Request.Context(), accountID)
	if err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// Leaderboard returns the module-owned direct-transfer leaderboard.
func (h *UserHandler) Leaderboard(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "transfer leaderboard is unavailable"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	result, err := h.service.GetLeaderboard(c.Request.Context(), c.DefaultQuery("period", "day"), limit)
	if err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func executeDirectTransferIdempotent(c *gin.Context, coordinator platform.IdempotencyCoordinator, payload directTransferRequest, execute func(context.Context) (DirectTransferRecord, error)) {
	if coordinator == nil || !coordinator.Available() {
		result, err := execute(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, result)
		return
	}

	actorScope := "user:0"
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok {
		actorScope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}
	result, err := coordinator.Execute(c.Request.Context(), platform.IdempotencyOptions{
		Scope: "balance_transfer", ActorScope: actorScope, Method: c.Request.Method, Route: c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"), Payload: payload, RequireKey: true, TTL: 24 * time.Hour,
	}, func(ctx context.Context) (any, error) {
		return execute(ctx)
	})
	if err != nil {
		if coordinator.IsStoreUnavailable(err) {
			coordinator.RecordStoreUnavailable(c.FullPath(), "balance_transfer", "handler_fail_close")
			logger.LegacyPrintf("wallet-extension.idempotency", "direct transfer idempotency store unavailable: route=%s", c.FullPath())
		}
		if retryAfter := coordinator.RetryAfterSeconds(err); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		response.ErrorFrom(c, err)
		return
	}
	if result != nil && result.Replayed {
		c.Header("X-Idempotency-Replayed", "true")
	}
	response.Success(c, result.Data)
}

func directTransferUserID(c *gin.Context) int64 {
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}
	if value, exists := c.Get("user_id"); exists {
		switch typed := value.(type) {
		case int64:
			return typed
		case float64:
			return int64(typed)
		}
	}
	return 0
}

func writeDirectTransferError(c *gin.Context, err error) {
	var applicationError *infraerrors.ApplicationError
	if errors.As(err, &applicationError) {
		c.JSON(int(applicationError.Code), gin.H{"error": applicationError.Message, "code": applicationError.Reason})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
