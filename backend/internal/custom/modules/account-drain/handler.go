package accountdrain

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) AccountStatus(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	status, err := h.service.AccountStatus(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get account directed-consumption status"})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) EnableAccount(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	status, err := h.service.EnableAccount(c.Request.Context(), accountID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) DisableAccount(c *gin.Context) {
	accountID, ok := parseAccountID(c)
	if !ok {
		return
	}
	if err := h.service.DisableAccount(c.Request.Context(), accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop account directed consumption"})
		return
	}
	c.Status(http.StatusNoContent)
}

func parseAccountID(c *gin.Context) (int64, bool) {
	accountID, err := strconv.ParseInt(c.Param("accountID"), 10, 64)
	if err != nil || accountID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid account ID"})
		return 0, false
	}
	return accountID, true
}
