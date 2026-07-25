package gamehall

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"

	"github.com/gin-gonic/gin"
)

// UserHandler serves authenticated game-hall users.
type UserHandler struct {
	service     *GameHallService
	idempotency platform.IdempotencyCoordinator
}

func NewUserHandler(gameHallService *GameHallService) *UserHandler {
	return NewUserHandlerWithIdempotency(gameHallService, nil)
}

func NewUserHandlerWithIdempotency(gameHallService *GameHallService, coordinator platform.IdempotencyCoordinator) *UserHandler {
	return &UserHandler{service: gameHallService, idempotency: coordinator}
}

func (h *UserHandler) Status(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := userID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	result, err := h.service.GetHallStatus(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

type gameExchangeRequest struct {
	Direction string  `json:"direction" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
}

func (h *UserHandler) Exchange(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := userID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req gameExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.executeUserIdempotentJSON(c, "game_hall_exchange", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.service.Exchange(ctx, GameExchangeInput{
			UserID:         userID,
			Direction:      req.Direction,
			Amount:         req.Amount,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
		})
	})
}

type gamePlayRequest struct {
	GameType  string  `json:"game_type" binding:"required"`
	BetAmount float64 `json:"bet_amount" binding:"required,gt=0"`
}

func (h *UserHandler) Play(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := userID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req gamePlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	h.executeUserIdempotentJSON(c, "game_hall_play", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.service.Play(ctx, GamePlayInput{
			UserID:         userID,
			GameType:       req.GameType,
			BetAmount:      req.BetAmount,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
		})
	})
}

func (h *UserHandler) Transactions(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := userID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.service.ListUserTransactions(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *UserHandler) Rounds(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := userID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, total, err := h.service.ListUserRounds(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

// AdminHandler serves the game-hall operational queries.
type AdminHandler struct{ service *GameHallService }

func NewAdminHandler(gameHallService *GameHallService) *AdminHandler {
	return &AdminHandler{service: gameHallService}
}

func (h *AdminHandler) Transactions(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	page, pageSize := response.ParsePagination(c)
	userID, ok := optionalUserID(c)
	if !ok {
		return
	}
	items, total, err := h.service.ListAdminTransactions(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *AdminHandler) Rounds(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	page, pageSize := response.ParsePagination(c)
	userID, ok := optionalUserID(c)
	if !ok {
		return
	}
	items, total, err := h.service.ListAdminRounds(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *AdminHandler) UserAccess(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID, ok := requiredUserID(c)
	if !ok {
		return
	}
	access, err := h.service.GetUserAccess(c.Request.Context(), userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, access)
}

type updateUserAccessRequest struct {
	Disabled *bool `json:"disabled" binding:"required"`
}

func (h *AdminHandler) UpdateUserAccess(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID, ok := requiredUserID(c)
	if !ok {
		return
	}
	var req updateUserAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	access, err := h.service.SetUserAccessDisabled(c.Request.Context(), userID, *req.Disabled)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, access)
}

func optionalUserID(c *gin.Context) (*int64, bool) {
	raw := c.Query("user_id")
	if raw == "" {
		return nil, true
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user_id")
		return nil, false
	}
	return &id, true
}

func requiredUserID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid user_id")
		return 0, false
	}
	return id, true
}

func userID(c *gin.Context) int64 {
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok {
		return subject.UserID
	}

	// Keep the legacy key fallback for callers that have not migrated yet.
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	switch value := id.(type) {
	case int64:
		return value
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func (h *UserHandler) executeUserIdempotentJSON(
	c *gin.Context,
	scope string,
	payload any,
	ttl time.Duration,
	execute func(context.Context) (any, error),
) {
	coordinator := h.idempotency
	if coordinator == nil || !coordinator.Available() {
		data, err := execute(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, data)
		return
	}

	actorScope := "user:0"
	if subject, ok := servermiddleware.GetAuthSubjectFromContext(c); ok {
		actorScope = "user:" + strconv.FormatInt(subject.UserID, 10)
	}

	result, err := coordinator.Execute(c.Request.Context(), platform.IdempotencyOptions{
		Scope:          scope,
		ActorScope:     actorScope,
		Method:         c.Request.Method,
		Route:          c.FullPath(),
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Payload:        payload,
		RequireKey:     true,
		TTL:            ttl,
	}, execute)
	if err != nil {
		if coordinator.IsStoreUnavailable(err) {
			coordinator.RecordStoreUnavailable(c.FullPath(), scope, "handler_fail_close")
			logger.LegacyPrintf("handler.idempotency", "[Idempotency] store unavailable: method=%s route=%s scope=%s strategy=fail_close", c.Request.Method, c.FullPath(), scope)
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
