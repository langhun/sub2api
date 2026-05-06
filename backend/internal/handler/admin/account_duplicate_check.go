package admin

import (
	"errors"
	"io"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AccountDuplicateCheckRequest struct {
	Platforms       []string `json:"platforms"`
	GroupIDs        []int64  `json:"group_ids"`
	Statuses        []string `json:"statuses"`
	IncludeInactive *bool    `json:"include_inactive"`
}

// DuplicateCheck scans matching accounts and reports duplicate groups without modifying data.
func (h *AccountHandler) DuplicateCheck(c *gin.Context) {
	var req AccountDuplicateCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.adminService.CheckDuplicateAccounts(c.Request.Context(), &service.AccountDuplicateCheckInput{
		Platforms:       req.Platforms,
		GroupIDs:        req.GroupIDs,
		Statuses:        req.Statuses,
		IncludeInactive: req.IncludeInactive,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
