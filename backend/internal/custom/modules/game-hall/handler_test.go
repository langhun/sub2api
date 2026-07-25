package gamehall

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserStatusRejectsMissingIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(NewGameHallService(nil, nil))
	passthrough := func(c *gin.Context) { c.Next() }
	module.RegisterRoutes(
		router,
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/game-hall/status", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestAdminTransactionsRejectsInvalidUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(NewGameHallService(nil, nil))
	passthrough := func(c *gin.Context) { c.Next() }
	module.RegisterRoutes(
		router,
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/game-hall/transactions?user_id=invalid", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestAdminUserAccessRejectsMissingDisabledFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(NewGameHallService(nil, nil))
	passthrough := func(c *gin.Context) { c.Next() }
	module.RegisterRoutes(
		router,
		middleware.JWTAuthMiddleware(passthrough),
		middleware.AdminAuthMiddleware(passthrough),
		middleware.AuditLogMiddleware(passthrough),
		nil,
	)

	request := httptest.NewRequest(http.MethodPut, "/api/v1/admin/game-hall/users/7/access", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusBadRequest, response.Code)
}
