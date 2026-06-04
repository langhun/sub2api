package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterUserRoutesRegistersFinanceAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		&handler.Handlers{
			BankCenter: handler.NewBankCenterHandler(nil),
		},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	paths := []string{
		"/api/v1/bank/account",
		"/api/v1/bank/transactions",
		"/api/v1/finance/account",
		"/api/v1/finance/transactions",
	}

	for _, path := range paths {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equalf(t, http.StatusUnauthorized, w.Code, "path=%s should be registered and reach auth check", path)
	}
}
