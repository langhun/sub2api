package admin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type AssignProxyAccountsRequest struct {
	ProxyIDs []int64                          `json:"proxy_ids" binding:"required,min=1"`
	DryRun   bool                             `json:"dry_run"`
	Filters  AssignProxyAccountsFilterRequest `json:"filters"`
}

type AssignProxyAccountsFilterRequest struct {
	Platforms []string `json:"platforms"`
	GroupIDs  []int64  `json:"group_ids"`
	Statuses  []string `json:"statuses"`
}

// AssignAccounts previews or applies average proxy assignment to unproxied accounts.
func (h *ProxyHandler) AssignAccounts(c *gin.Context) {
	var req AssignProxyAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	run := func(ctx context.Context) (any, error) {
		return h.adminService.AssignProxiesToAccounts(ctx, &service.AssignProxiesToAccountsInput{
			ProxyIDs: req.ProxyIDs,
			DryRun:   req.DryRun,
			Filters: service.ProxyAssignmentAccountFilters{
				Platforms: req.Filters.Platforms,
				GroupIDs:  req.Filters.GroupIDs,
				Statuses:  req.Filters.Statuses,
			},
		})
	}

	if req.DryRun {
		result, err := run(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		response.Success(c, result)
		return
	}

	executeAdminIdempotentJSON(c, "admin.proxies.assign_accounts", req, service.DefaultWriteIdempotencyTTL(), run)
}
