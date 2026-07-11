package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BalanceTransferHandler struct {
	transferService *service.BalanceTransferService
}

func NewBalanceTransferHandler(transferService *service.BalanceTransferService) *BalanceTransferHandler {
	return &BalanceTransferHandler{transferService: transferService}
}

func (h *BalanceTransferHandler) Transfer(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		ReceiverID int64   `json:"receiver_id" binding:"required"`
		Amount     float64 `json:"amount" binding:"required,gt=0"`
		Memo       *string `json:"memo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	executeUserIdempotentJSON(c, "balance_transfer", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.transferService.Transfer(ctx, userID, req.ReceiverID, req.Amount, req.Memo)
	})
}

func (h *BalanceTransferHandler) ValidateTransfer(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		ReceiverID int64   `json:"receiver_id" binding:"required"`
		Amount     float64 `json:"amount" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	validation, err := h.transferService.ValidateTransfer(c.Request.Context(), userID, req.ReceiverID, req.Amount)
	if err != nil {
		WriteAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, validation)
}

func (h *BalanceTransferHandler) GetHistory(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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
		WriteAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": records, "total": total, "page": page, "page_size": pageSize})
}

func (h *BalanceTransferHandler) GetStats(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	sent, received, feePaid, err := h.transferService.GetTransferStats(c.Request.Context(), userID)
	if err != nil {
		WriteAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"total_sent": sent, "total_received": received, "total_fee_paid": feePaid})
}

func (h *BalanceTransferHandler) CreateRedPacket(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		TotalAmount   float64 `json:"total_amount" binding:"required,gt=0"`
		Count         int     `json:"count" binding:"required,gt=0"`
		RedPacketType string  `json:"redpacket_type"`
		Memo          *string `json:"memo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.RedPacketType == "" {
		req.RedPacketType = "equal"
	}
	executeUserIdempotentJSON(c, "redpacket_create", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.transferService.CreateRedPacket(ctx, userID, req.TotalAmount, req.Count, req.RedPacketType, req.Memo)
	})
}

func (h *BalanceTransferHandler) ClaimRedPacket(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	executeUserIdempotentJSON(c, "redpacket_claim", req, 24*time.Hour, func(ctx context.Context) (any, error) {
		return h.transferService.ClaimRedPacket(ctx, userID, req.Code)
	})
}

func (h *BalanceTransferHandler) GetRedPacketDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	rp, claims, err := h.transferService.GetRedPacketDetailForUser(c.Request.Context(), userID, id)
	if err != nil {
		WriteAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"redpacket": rp, "claims": claims})
}

func (h *BalanceTransferHandler) GetMyRedPackets(c *gin.Context) {
	userID := GetUserIDAware(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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
	role := c.DefaultQuery("role", "sent")
	if role != "sent" && role != "received" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be sent or received"})
		return
	}
	records, total, err := h.transferService.GetMyRedPackets(c.Request.Context(), userID, role, page, pageSize)
	if err != nil {
		WriteAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": records, "total": total, "page": page, "page_size": pageSize})
}

func (h *BalanceTransferHandler) GetLeaderboard(c *gin.Context) {
	period := c.DefaultQuery("period", "day")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	entries, err := h.transferService.GetLeaderboard(c.Request.Context(), period, limit)
	if err != nil {
		WriteAppError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

func GetUserIDAware(c *gin.Context) int64 {
	id, exists := c.Get("user_id")
	if !exists {
		return 0
	}
	switch v := id.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func WriteAppError(c *gin.Context, err error) {
	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		c.JSON(int(appErr.Code), gin.H{"error": appErr.Message, "code": appErr.Reason})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
