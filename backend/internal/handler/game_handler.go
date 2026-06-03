package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type gamePlayRequest struct {
	GameType  string  `json:"game_type" binding:"required"`
	BetAmount float64 `json:"bet_amount" binding:"required,gt=0"`
}

type GameHandler struct {
	gameService *service.GameService
}

func NewGameHandler(gameService *service.GameService) *GameHandler {
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
			RequestID:      c.GetHeader("Idempotency-Key"),
		})
	})
}
