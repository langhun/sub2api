package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegisterUserRoutesRegistersLotteryRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	v1 := router.Group("/api/v1")

	RegisterUserRoutes(
		v1,
		&handler.Handlers{
			Lottery: handler.NewLotteryHandler(nil),
		},
		servermiddleware.JWTAuthMiddleware(func(c *gin.Context) {
			c.Next()
		}),
		nil,
	)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/v1/lottery/current"},
		{method: http.MethodPost, path: "/api/v1/lottery/bet", body: `{"red_balls":[1,8,12,18,25,33],"blue_ball":9}`},
		{method: http.MethodGet, path: "/api/v1/lottery/orders"},
		{method: http.MethodGet, path: "/api/v1/lottery/results"},
		{method: http.MethodGet, path: "/api/v1/lottery/results/2026062"},
	}

	for _, tt := range tests {
		var body *strings.Reader
		if tt.body == "" {
			body = strings.NewReader("")
		} else {
			body = strings.NewReader(tt.body)
		}
		req := httptest.NewRequest(tt.method, tt.path, body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equalf(t, http.StatusUnauthorized, w.Code, "%s %s should be registered and reach auth check", tt.method, tt.path)
	}
}
