package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BalanceTransferHandler struct {
	transferService *service.BalanceTransferService
}

func NewBalanceTransferHandler(transferService *service.BalanceTransferService) *BalanceTransferHandler {
	return &BalanceTransferHandler{transferService: transferService}
}

func getUserID(c *gin.Context) int64 {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		return 0
	}
	return subject.UserID
}

func (h *BalanceTransferHandler) Transfer(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		ReceiverID int64   `json:"receiver_id" binding:"required"`
		Amount     float64 `json:"amount" binding:"required,gt=0"`
		Memo       *string `json:"memo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	record, err := h.transferService.Transfer(c.Request.Context(), userID, req.ReceiverID, req.Amount, req.Memo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, record)
}

func (h *BalanceTransferHandler) ValidateTransfer(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		ReceiverID int64   `json:"receiver_id" binding:"required"`
		Amount     float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	fee, feeRate, err := h.transferService.ValidateTransfer(c.Request.Context(), userID, req.ReceiverID, req.Amount)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"fee": fee, "fee_rate": feeRate})
}

func (h *BalanceTransferHandler) GetHistory(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	role := c.DefaultQuery("role", "all")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	records, total, err := h.transferService.GetHistory(c.Request.Context(), userID, role, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Paginated(c, records, int64(total), page, pageSize)
}

func (h *BalanceTransferHandler) GetStats(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	sent, received, feePaid, err := h.transferService.GetTransferStats(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"total_sent": sent, "total_received": received, "total_fee_paid": feePaid})
}

func (h *BalanceTransferHandler) CreateRedPacket(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		TotalAmount   float64 `json:"total_amount" binding:"required,gt=0"`
		Count         int     `json:"count" binding:"required,gt=0"`
		RedPacketType string  `json:"redpacket_type"`
		Memo          *string `json:"memo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if req.RedPacketType == "" {
		req.RedPacketType = "equal"
	}
	rp, err := h.transferService.CreateRedPacket(c.Request.Context(), userID, req.TotalAmount, req.Count, req.RedPacketType, req.Memo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rp)
}

func (h *BalanceTransferHandler) ClaimRedPacket(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	claim, err := h.transferService.ClaimRedPacket(c.Request.Context(), userID, req.Code)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, claim)
}

func (h *BalanceTransferHandler) GetRedPacketDetail(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		response.BadRequest(c, "invalid id")
		return
	}
	rp, claims, err := h.transferService.GetRedPacketDetail(c.Request.Context(), userID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"redpacket": rp, "claims": claims})
}

func (h *BalanceTransferHandler) GetMyRedPackets(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	records, total, err := h.transferService.GetMyRedPackets(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Paginated(c, records, int64(total), page, pageSize)
}

func (h *BalanceTransferHandler) GetLeaderboard(c *gin.Context) {
	period := c.DefaultQuery("period", "day")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	entries, err := h.transferService.GetLeaderboard(c.Request.Context(), period, limit)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, entries)
}

func (h *BalanceTransferHandler) SearchUsers(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		response.Unauthorized(c, "unauthorized")
		return
	}
	q := c.Query("q")
	results, err := h.transferService.SearchUsers(c.Request.Context(), q)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if results == nil {
		results = []*service.UserSearchResult{}
	}
	response.Success(c, results)
}

func GetUserIDAware(c *gin.Context) int64 {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		return 0
	}
	return subject.UserID
}
