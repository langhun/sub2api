package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type GameHallHandler struct{ service *service.GameHallService }

func NewGameHallHandler(gameHallService *service.GameHallService) *GameHallHandler {
	return &GameHallHandler{service: gameHallService}
}

func (h *GameHallHandler) Transactions(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	page, pageSize := response.ParsePagination(c)
	userID, ok := optionalGameHallUserID(c)
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

func (h *GameHallHandler) Rounds(c *gin.Context) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "game hall is unavailable")
		return
	}
	page, pageSize := response.ParsePagination(c)
	userID, ok := optionalGameHallUserID(c)
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

func optionalGameHallUserID(c *gin.Context) (*int64, bool) {
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
