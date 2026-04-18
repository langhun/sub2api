package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type MonitoringHandler struct {
	monitoringService *service.MonitoringService
}

func NewMonitoringHandler(monitoringService *service.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{monitoringService: monitoringService}
}

func (h *MonitoringHandler) GetOverview(c *gin.Context) {
	overview, err := h.monitoringService.GetOverview(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to get monitoring data")
		return
	}
	response.Success(c, overview)
}
