package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type projectMihomoRequest struct {
	SubscriptionURL  string   `json:"subscription_url"`
	SubscriptionURLs []string `json:"subscription_urls"`
	SubscriptionUA   string   `json:"subscription_user_agent"`
	UpdateInterval   int      `json:"update_interval"`
	Protocol         string   `json:"protocol"`
	TargetHost       string   `json:"target_host"`
	StartPort        int      `json:"start_port"`
	ListenerCount    int      `json:"listener_count"`
	ControllerURL    string   `json:"controller_url"`
	ControllerSecret string   `json:"controller_secret"`
	ProxyNamePrefix  string   `json:"proxy_name_prefix"`
	ListenerRegions  []string `json:"listener_regions"`
}

func (h *ProxyHandler) GetProjectMihomo(c *gin.Context) {
	if h.projectMihomoService == nil {
		response.NotFound(c, "Project Mihomo service is not configured")
		return
	}
	status, err := h.projectMihomoService.GetStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *ProxyHandler) UpdateProjectMihomo(c *gin.Context) {
	if h.projectMihomoService == nil {
		response.NotFound(c, "Project Mihomo service is not configured")
		return
	}
	var req projectMihomoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings, err := h.projectMihomoService.SetSettings(c.Request.Context(), projectMihomoSettingsFromRequest(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *ProxyHandler) SyncProjectMihomo(c *gin.Context) {
	if h.projectMihomoService == nil {
		response.NotFound(c, "Project Mihomo service is not configured")
		return
	}
	var req projectMihomoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.projectMihomoService.Sync(c.Request.Context(), projectMihomoSettingsFromRequest(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func projectMihomoSettingsFromRequest(req projectMihomoRequest) *service.ProjectMihomoSettings {
	return &service.ProjectMihomoSettings{
		SubscriptionURL:  req.SubscriptionURL,
		SubscriptionURLs: req.SubscriptionURLs,
		SubscriptionUA:   req.SubscriptionUA,
		UpdateInterval:   req.UpdateInterval,
		Protocol:         req.Protocol,
		TargetHost:       req.TargetHost,
		StartPort:        req.StartPort,
		ListenerCount:    req.ListenerCount,
		ControllerURL:    req.ControllerURL,
		ControllerSecret: req.ControllerSecret,
		ProxyNamePrefix:  req.ProxyNamePrefix,
		ListenerRegions:  req.ListenerRegions,
	}
}
