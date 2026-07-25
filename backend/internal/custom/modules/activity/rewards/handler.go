package rewards

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler keeps the legacy blind-box administration API stable while routing
// all business operations through the activity module boundary.
type Handler struct{ service AdminOperations }

func NewHandler(service AdminOperations) *Handler { return &Handler{service: service} }

func (h *Handler) ListPrizeItems(c *gin.Context) {
	items, err := h.ready().ListPrizeItems(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if items == nil {
		items = []Prize{}
	}
	response.Success(c, items)
}

func (h *Handler) CreatePrizeItem(c *gin.Context) {
	var request CreatePrizeItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.ready().CreatePrizeItem(c.Request.Context(), request)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) UpdatePrizeItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var request UpdatePrizeItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.ready().UpdatePrizeItem(c.Request.Context(), id, request)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *Handler) DeletePrizeItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.ready().DeletePrizeItem(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.ready().GetStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *Handler) ListRewardDeliveries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter := DeliveryFilter{Status: c.Query("status"), SourceType: c.Query("source_type"), Page: page, PageSize: pageSize}
	if rawUserID := c.Query("user_id"); rawUserID != "" {
		userID, err := strconv.ParseInt(rawUserID, 10, 64)
		if err != nil || userID <= 0 {
			response.BadRequest(c, "invalid user_id")
			return
		}
		filter.UserID = &userID
	}
	items, total, err := h.ready().ListRewardDeliveries(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *Handler) RetryRewardDelivery(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return
	}
	delivery, err := h.ready().RetryRewardDelivery(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, delivery)
}

func (h *Handler) CompensateRewardDelivery(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || len(request.Reason) > 500 {
		response.BadRequest(c, "invalid compensation reason")
		return
	}
	delivery, err := h.ready().CompensateRewardDelivery(c.Request.Context(), id, request.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, delivery)
}

func (h *Handler) ready() AdminOperations {
	if h == nil || h.service == nil {
		return unavailableAdminService{}
	}
	return h.service
}

type unavailableAdminService struct{}

func (unavailableAdminService) ListPrizeItems(c context.Context) ([]Prize, error) {
	return nil, ErrUnavailable
}
func (unavailableAdminService) CreatePrizeItem(context.Context, CreatePrizeItemRequest) (*Prize, error) {
	return nil, ErrUnavailable
}
func (unavailableAdminService) UpdatePrizeItem(context.Context, int64, UpdatePrizeItemRequest) (*Prize, error) {
	return nil, ErrUnavailable
}
func (unavailableAdminService) DeletePrizeItem(context.Context, int64) error { return ErrUnavailable }
func (unavailableAdminService) GetStats(context.Context) (PrizeStats, error) {
	return PrizeStats{}, ErrUnavailable
}
func (unavailableAdminService) ListRewardDeliveries(context.Context, DeliveryFilter) ([]Delivery, int64, error) {
	return nil, 0, ErrUnavailable
}
func (unavailableAdminService) RetryRewardDelivery(context.Context, int64) (*Delivery, error) {
	return nil, ErrUnavailable
}
func (unavailableAdminService) CompensateRewardDelivery(context.Context, int64, string) (*Delivery, error) {
	return nil, ErrUnavailable
}

var _ AdminRouteHandler = (*Handler)(nil)
