package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ProxySubscriptionHandler struct {
	adminService service.AdminService
}

func NewProxySubscriptionHandler(adminService service.AdminService) *ProxySubscriptionHandler {
	return &ProxySubscriptionHandler{adminService: adminService}
}

type CreateProxySubscriptionSourceRequest struct {
	Name                 string `json:"name" binding:"required"`
	URL                  string `json:"url" binding:"required"`
	SourceFormat         string `json:"source_format"`
	Enabled              *bool  `json:"enabled"`
	RefreshIntervalHours int    `json:"refresh_interval_hours"`
	AutoAddToPool        bool   `json:"auto_add_to_pool"`
}

type UpdateProxySubscriptionSourceRequest struct {
	Name                 *string `json:"name"`
	URL                  *string `json:"url"`
	SourceFormat         *string `json:"source_format"`
	Enabled              *bool   `json:"enabled"`
	RefreshIntervalHours *int    `json:"refresh_interval_hours"`
	AutoAddToPool        *bool   `json:"auto_add_to_pool"`
}

func (h *ProxySubscriptionHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	var enabled *bool
	if raw := strings.TrimSpace(c.Query("enabled")); raw != "" {
		value := raw == "true"
		enabled = &value
	}
	items, total, err := h.adminService.ListProxySubscriptionSources(c.Request.Context(), page, pageSize, search, enabled)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.ProxySubscriptionSource, 0, len(items))
	for i := range items {
		out = append(out, *dto.ProxySubscriptionSourceFromService(&items[i]))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *ProxySubscriptionHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid source ID")
		return
	}
	item, err := h.adminService.GetProxySubscriptionSource(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxySubscriptionSourceFromService(item))
}

func (h *ProxySubscriptionHandler) Create(c *gin.Context) {
	var req CreateProxySubscriptionSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item, err := h.adminService.CreateProxySubscriptionSource(c.Request.Context(), &service.CreateProxySubscriptionSourceInput{
		Name:                 strings.TrimSpace(req.Name),
		URL:                  strings.TrimSpace(req.URL),
		SourceFormat:         strings.TrimSpace(req.SourceFormat),
		Enabled:              enabled,
		RefreshIntervalHours: req.RefreshIntervalHours,
		AutoAddToPool:        req.AutoAddToPool,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxySubscriptionSourceFromService(item))
}

func (h *ProxySubscriptionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid source ID")
		return
	}
	var req UpdateProxySubscriptionSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.adminService.UpdateProxySubscriptionSource(c.Request.Context(), id, &service.UpdateProxySubscriptionSourceInput{
		Name:                 req.Name,
		URL:                  req.URL,
		SourceFormat:         req.SourceFormat,
		Enabled:              req.Enabled,
		RefreshIntervalHours: req.RefreshIntervalHours,
		AutoAddToPool:        req.AutoAddToPool,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxySubscriptionSourceFromService(item))
}

func (h *ProxySubscriptionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid source ID")
		return
	}
	if err := h.adminService.DeleteProxySubscriptionSource(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Proxy subscription source deleted successfully"})
}

func (h *ProxySubscriptionHandler) Refresh(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid source ID")
		return
	}
	result, err := h.adminService.RefreshProxySubscriptionSource(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ProxySubscriptionRefreshResultFromService(result))
}

func (h *ProxySubscriptionHandler) ListNodes(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid source ID")
		return
	}
	items, err := h.adminService.ListProxySubscriptionNodes(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.ProxySubscriptionNode, 0, len(items))
	for i := range items {
		out = append(out, *dto.ProxySubscriptionNodeFromService(&items[i]))
	}
	response.Success(c, out)
}

func (h *ProxySubscriptionHandler) ListProxies(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid source ID")
		return
	}
	items, err := h.adminService.ListMaterializedProxiesBySubscriptionSource(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]dto.AdminProxy, 0, len(items))
	for i := range items {
		out = append(out, *dto.ProxyFromServiceAdmin(&items[i]))
	}
	response.Success(c, out)
}
