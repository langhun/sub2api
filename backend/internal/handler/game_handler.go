package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type gameHallHandlerService interface {
	GetHallStatus(ctx context.Context, userID int64) (*service.GameHallStatus, error)
	Exchange(ctx context.Context, input service.GameExchangeInput) (*service.GameExchangeResult, error)
	Play(ctx context.Context, input service.GamePlayInput) (*service.GamePlayResult, error)
}

type gameExchangeRequest struct {
	Direction string  `json:"direction" binding:"required"`
	Amount    float64 `json:"amount" binding:"required,gt=0"`
}

type gamePlayRequest struct {
	GameType  string  `json:"game_type" binding:"required"`
	BetAmount float64 `json:"bet_amount" binding:"required,gt=0"`
}

type GameHandler struct {
	gameService gameHallHandlerService
}

func NewGameHandler(gameService gameHallHandlerService) *GameHandler {
	return &GameHandler{gameService: gameService}
}

func (h *GameHandler) GetHallStatus(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	status, err := h.gameService.GetHallStatus(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, status)
}

func (h *GameHandler) Exchange(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req gameExchangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: direction and amount are required")
		return
	}

	executeUserIdempotentJSON(c, "user.games.exchange", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.gameService.Exchange(ctx, service.GameExchangeInput{
			UserID:         subject.UserID,
			Direction:      req.Direction,
			Amount:         req.Amount,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
		})
	})
}

func (h *GameHandler) Play(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req gamePlayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: game_type and bet_amount are required")
		return
	}

	executeUserIdempotentJSON(c, "user.games.play", req, service.DefaultWriteIdempotencyTTL(), func(ctx context.Context) (any, error) {
		return h.gameService.Play(ctx, service.GamePlayInput{
			UserID:         subject.UserID,
			GameType:       req.GameType,
			BetAmount:      req.BetAmount,
			IdempotencyKey: c.GetHeader("Idempotency-Key"),
		})
	})
}
