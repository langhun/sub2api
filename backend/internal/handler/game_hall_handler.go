package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GameHallHandler struct{ service *service.GameHallService }

func NewGameHallHandler(gameHallService *service.GameHallService) *GameHallHandler {
	return &GameHallHandler{service: gameHallService}
}

func (h *GameHallHandler) Status(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := GetUserIDAware(c)
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

func (h *GameHallHandler) Exchange(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := GetUserIDAware(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req gameExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	executeUserIdempotentJSON(c, "game_hall_exchange", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.service.Exchange(ctx, service.GameExchangeInput{UserID: userID, Direction: req.Direction, Amount: req.Amount, IdempotencyKey: c.GetHeader("Idempotency-Key")})
	})
}

type gamePlayRequest struct {
	GameType  string  `json:"game_type" binding:"required"`
	BetAmount float64 `json:"bet_amount" binding:"required,gt=0"`
}

func (h *GameHallHandler) Play(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := GetUserIDAware(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req gamePlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	executeUserIdempotentJSON(c, "game_hall_play", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.service.Play(ctx, service.GamePlayInput{UserID: userID, GameType: req.GameType, BetAmount: req.BetAmount, IdempotencyKey: c.GetHeader("Idempotency-Key")})
	})
}

func (h *GameHallHandler) Transactions(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := GetUserIDAware(c)
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

func (h *GameHallHandler) Rounds(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	userID := GetUserIDAware(c)
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
