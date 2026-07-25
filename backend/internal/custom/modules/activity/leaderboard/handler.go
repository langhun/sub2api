package leaderboard

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// Handler is a route-neutral HTTP adapter. The future activity route registrar
// can bind its methods without importing legacy leaderboard handlers.
type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Balance(c *gin.Context) {
	h.list(c, contract.LeaderboardBalance)
}

func (h *Handler) Consumption(c *gin.Context) {
	h.list(c, contract.LeaderboardConsumption)
}

func (h *Handler) Checkin(c *gin.Context) {
	h.list(c, contract.LeaderboardCheckin)
}

func (h *Handler) Transfer(c *gin.Context) {
	h.list(c, contract.LeaderboardTransfer)
}

func (h *Handler) list(c *gin.Context, kind contract.LeaderboardKind) {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "activity leaderboard is unavailable")
		return
	}

	page, pageSize := response.ParsePagination(c)
	result, err := h.service.List(c.Request.Context(), kind, contract.LeaderboardQuery{
		Page:     page,
		PageSize: pageSize,
		Period:   contract.LeaderboardPeriod(c.Query("period")),
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrDisabled):
			response.NotFound(c, "Leaderboard is disabled")
		case errors.Is(err, ErrInvalidPeriod), errors.Is(err, ErrInvalidPage), errors.Is(err, ErrUnsupportedKind):
			response.BadRequest(c, err.Error())
		default:
			response.InternalError(c, "Failed to get leaderboard")
		}
		return
	}

	response.Paginated(c, result.Entries, result.Total, page, pageSize)
}
