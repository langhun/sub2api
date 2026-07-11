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
