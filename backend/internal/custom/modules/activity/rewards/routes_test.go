package rewards

import (
	"net/http"
	"net/http/httptest"
	"testing"

	legacyhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/gin-gonic/gin"
)

func TestModuleRegistersEstablishedAdminBlindboxAndDeliveryRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(legacyhandler.NewBlindboxHandler(nil))
	module.RegisterRoutes(router.Group("/api/v1/admin"))

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, route := range []string{
		"GET /api/v1/admin/blindbox/prize-items",
		"POST /api/v1/admin/blindbox/prize-items",
		"PUT /api/v1/admin/blindbox/prize-items/:id",
		"DELETE /api/v1/admin/blindbox/prize-items/:id",
		"GET /api/v1/admin/blindbox/stats",
		"GET /api/v1/admin/reward-deliveries",
		"POST /api/v1/admin/reward-deliveries/:id/retry",
		"POST /api/v1/admin/reward-deliveries/:id/compensate",
	} {
		if _, ok := routes[route]; !ok {
			t.Errorf("missing module route %s", route)
		}
	}
}

func TestModuleForwardsExistingHandlerValidationResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewModule(legacyhandler.NewBlindboxHandler(nil)).RegisterRoutes(router.Group("/api/v1/admin"))

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/blindbox/prize-items/not-an-id", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("DELETE invalid prize ID status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
