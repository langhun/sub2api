package walletextension

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// AdminHandler serves the transfer-only administrative compatibility API.
type AdminHandler struct{ compatibility *LegacyCompatibility }

// NewAdminHandler constructs the transfer administration adapter.
func NewAdminHandler(compatibility *LegacyCompatibility) *AdminHandler {
	return &AdminHandler{compatibility: compatibility}
}

// ListTransfers preserves the existing paginated transfer management response.
func (h *AdminHandler) ListTransfers(c *gin.Context) {
	if h == nil || h.compatibility == nil || h.compatibility.legacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "transfer administration is unavailable"})
		return
	}
	page, pageSize := adminTransferPage(c)
	filter := &service.TransferFilter{Status: c.DefaultQuery("status", ""), TransferType: c.DefaultQuery("transfer_type", "")}
	if value := c.Query("user_id"); value != "" {
		if userID, err := strconv.ParseInt(value, 10, 64); err == nil && userID > 0 {
			filter.UserID = &userID
		}
	}
	if value := c.Query("start_time"); value != "" {
		if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
			filter.StartTime = timestamp
		}
	}
	if value := c.Query("end_time"); value != "" {
		if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
			filter.EndTime = timestamp
		}
	}
	items, total, err := h.compatibility.legacy.GetAllTransfers(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total, "page": page, "page_size": pageSize})
}

// FeeStats preserves the existing 30-day default transfer-fee report.
func (h *AdminHandler) FeeStats(c *gin.Context) {
	if h == nil || h.compatibility == nil || h.compatibility.legacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "transfer administration is unavailable"})
		return
	}
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)
	if value := c.Query("start_time"); value != "" {
		if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
			startTime = timestamp
		}
	}
	if value := c.Query("end_time"); value != "" {
		if timestamp, err := time.Parse(time.RFC3339, value); err == nil {
			endTime = timestamp
		}
	}
	result, err := h.compatibility.legacy.GetFeeStats(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// FreezeTransfer preserves the existing transfer freeze semantics.
func (h *AdminHandler) FreezeTransfer(c *gin.Context) {
	if h == nil || h.compatibility == nil || h.compatibility.legacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "transfer administration is unavailable"})
		return
	}
	transferID, ok := adminTransferID(c)
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&request)
	if err := h.compatibility.legacy.FreezeTransfer(c.Request.Context(), directTransferUserID(c), transferID); err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "transfer frozen"})
}

// RevokeTransfer preserves the existing transfer revocation semantics.
func (h *AdminHandler) RevokeTransfer(c *gin.Context) {
	if h == nil || h.compatibility == nil || h.compatibility.legacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "transfer administration is unavailable"})
		return
	}
	transferID, ok := adminTransferID(c)
	if !ok {
		return
	}
	var request struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&request)
	if err := h.compatibility.legacy.RevokeTransfer(c.Request.Context(), directTransferUserID(c), transferID, request.Reason); err != nil {
		writeDirectTransferError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "transfer revoked"})
}

// BatchDistribute preserves the existing administrator-authorized distribution API.
func (h *AdminHandler) BatchDistribute(c *gin.Context) {
	if h == nil || h.compatibility == nil || h.compatibility.legacy == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "transfer administration is unavailable"})
		return
	}
	var request struct {
		Targets []service.BatchDistributeTarget `json:"targets" binding:"required"`
		Memo    *string                         `json:"memo"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := h.compatibility.legacy.BatchDistribute(c.Request.Context(), directTransferUserID(c), request.Targets, request.Memo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "count": len(items)})
}

func adminTransferPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return normalizeDirectTransferPage(page, pageSize)
}

func adminTransferID(c *gin.Context) (int64, bool) {
	transferID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if transferID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, false
	}
	return transferID, true
}
