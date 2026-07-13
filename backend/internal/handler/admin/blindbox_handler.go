package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BlindboxHandler struct {
	blindboxService *service.BlindBoxService
}

func NewBlindboxHandler(blindboxService *service.BlindBoxService) *BlindboxHandler {
	return &BlindboxHandler{blindboxService: blindboxService}
}

func (h *BlindboxHandler) ListPrizeItems(c *gin.Context) {
	items, err := h.blindboxService.ListPrizeItems(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if items == nil {
		items = []service.PrizeItem{}
	}
	response.Success(c, items)
}

func (h *BlindboxHandler) CreatePrizeItem(c *gin.Context) {
	var req service.CreatePrizeItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.blindboxService.CreatePrizeItem(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *BlindboxHandler) UpdatePrizeItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var req service.UpdatePrizeItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.blindboxService.UpdatePrizeItem(c.Request.Context(), id, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *BlindboxHandler) DeletePrizeItem(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	if err := h.blindboxService.DeletePrizeItem(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}

func (h *BlindboxHandler) GetStats(c *gin.Context) {
	stats, err := h.blindboxService.GetStats(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *BlindboxHandler) ListRewardDeliveries(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter := service.RewardDeliveryFilter{
		Status:     c.Query("status"),
		SourceType: c.Query("source_type"),
		Page:       page,
		PageSize:   pageSize,
	}
	if rawUserID := c.Query("user_id"); rawUserID != "" {
		userID, err := strconv.ParseInt(rawUserID, 10, 64)
		if err != nil || userID <= 0 {
			response.BadRequest(c, "invalid user_id")
			return
		}
		filter.UserID = &userID
	}
	items, total, err := h.blindboxService.ListRewardDeliveries(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (h *BlindboxHandler) RetryRewardDelivery(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return
	}
	delivery, err := h.blindboxService.RetryRewardDelivery(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, delivery)
}

func (h *BlindboxHandler) CompensateRewardDelivery(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Reason) > 500 {
		response.BadRequest(c, "invalid compensation reason")
		return
	}
	delivery, err := h.blindboxService.CompensateRewardDelivery(c.Request.Context(), id, req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, delivery)
}
