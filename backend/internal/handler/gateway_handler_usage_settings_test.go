package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type usageQueryGateStub struct {
	enabled bool
	err     error
}

func (s usageQueryGateStub) UsageQueryEnabled(context.Context) (bool, error) {
	return s.enabled, s.err
}

func TestGatewayUsageUsesInjectedActivitySettingsGate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		gate       UsageQueryGate
		wantStatus int
	}{
		{name: "disabled", gate: usageQueryGateStub{enabled: false}, wantStatus: http.StatusForbidden},
		{name: "unavailable", gate: usageQueryGateStub{err: errors.New("store unavailable")}, wantStatus: http.StatusServiceUnavailable},
		{name: "missing", gate: nil, wantStatus: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

			handler := &GatewayHandler{}
			handler.SetUsageQueryGate(tt.gate)
			handler.Usage(ctx)

			require.Equal(t, tt.wantStatus, recorder.Code)
		})
	}
}
