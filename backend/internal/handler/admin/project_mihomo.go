package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type mihomoRequest struct {
	Protocol         string   `json:"protocol"`
	TargetHost       string   `json:"target_host"`
	StartPort        int      `json:"start_port"`
	ListenerCount    int      `json:"listener_count"`
	ControllerURL    string   `json:"controller_url"`
	ControllerSecret string   `json:"controller_secret"`
	ProxyNamePrefix  string   `json:"proxy_name_prefix"`
	ListenerRegions  []string `json:"listener_regions"`
	AutoOptimize     bool     `json:"auto_optimize"`
	CountryFilter    string   `json:"country_filter"`
}

func (h *ProxyHandler) GetMihomo(c *gin.Context) {
	if h.mihomoService == nil {
		response.NotFound(c, "Mihomo service is not configured")
		return
	}
	status, err := h.mihomoService.GetStatus(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, status)
}

func (h *ProxyHandler) UpdateMihomo(c *gin.Context) {
	if h.mihomoService == nil {
		response.NotFound(c, "Mihomo service is not configured")
		return
	}
	var req mihomoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	settings, err := h.mihomoService.SetSettings(c.Request.Context(), mihomoSettingsFromRequest(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, settings)
}

func (h *ProxyHandler) SyncMihomo(c *gin.Context) {
	if h.mihomoService == nil {
		response.NotFound(c, "Mihomo service is not configured")
		return
	}
	var req mihomoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	result, err := h.mihomoService.Sync(c.Request.Context(), mihomoSettingsFromRequest(req))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}

func mihomoSettingsFromRequest(req mihomoRequest) *service.MihomoSettings {
	return &service.MihomoSettings{
		Protocol:         req.Protocol,
		TargetHost:       req.TargetHost,
		StartPort:        req.StartPort,
		ListenerCount:    req.ListenerCount,
		ControllerURL:    req.ControllerURL,
		ControllerSecret: req.ControllerSecret,
		ProxyNamePrefix:  req.ProxyNamePrefix,
		ListenerRegions:  req.ListenerRegions,
		AutoOptimize:     req.AutoOptimize,
		CountryFilter:    req.CountryFilter,
	}
}
