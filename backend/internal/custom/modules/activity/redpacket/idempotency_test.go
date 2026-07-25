package redpacket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExecuteRedPacketIdempotentUsesInjectedPlatformCoordinator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	coordinator := &idempotencyCoordinatorStub{available: true, result: &platform.IdempotencyResult{Data: map[string]string{"status": "ok"}, Replayed: true}}
	router := gin.New()
	router.POST("/api/v1/redpacket", func(c *gin.Context) {
		executeRedPacketIdempotent(c, coordinator, "redpacket_create", map[string]string{"kind": "equal"}, func(context.Context) (any, error) {
			return map[string]string{"status": "executed"}, nil
		})
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/redpacket", nil)
	request.Header.Set("Idempotency-Key", "create-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "true", response.Header().Get("X-Idempotency-Replayed"))
	require.Equal(t, "redpacket_create", coordinator.options.Scope)
	require.Equal(t, "user:0", coordinator.options.ActorScope)
	require.Equal(t, http.MethodPost, coordinator.options.Method)
	require.Equal(t, "/api/v1/redpacket", coordinator.options.Route)
	require.Equal(t, "create-1", coordinator.options.IdempotencyKey)
	require.True(t, coordinator.options.RequireKey)
	require.Equal(t, 24*time.Hour, coordinator.options.TTL)
}

type idempotencyCoordinatorStub struct {
	available bool
	options   platform.IdempotencyOptions
	result    *platform.IdempotencyResult
	err       error
}

func (s *idempotencyCoordinatorStub) Available() bool { return s.available }
func (s *idempotencyCoordinatorStub) Execute(_ context.Context, options platform.IdempotencyOptions, _ func(context.Context) (any, error)) (*platform.IdempotencyResult, error) {
	s.options = options
	return s.result, s.err
}
func (*idempotencyCoordinatorStub) IsStoreUnavailable(error) bool                 { return false }
func (*idempotencyCoordinatorStub) RecordStoreUnavailable(string, string, string) {}
func (*idempotencyCoordinatorStub) RetryAfterSeconds(error) int                   { return 0 }
