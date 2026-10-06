package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *OpenAIGatewayHandler) rejectSmartRoutingForeignForwardModel(
	c *gin.Context,
	selection *service.AccountSelectionResult,
	account *service.Account,
	forwardModel string,
	requireCompact, streamStarted bool,
) bool {
	ctx := c.Request.Context()
	if service.SmartRoutingForwardModelCompatible(ctx, forwardModel) {
		actualModel := service.ResolveOpenAIAccountUpstreamModelForRequest(account, forwardModel, requireCompact)
		if service.SmartRoutingForwardModelCompatible(ctx, actualModel) {
			return false
		}
	}
	if selection != nil && selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
	h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", streamStarted)
	return true
}
