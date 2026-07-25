package checkin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerServiceStub struct {
	checkinUserID int64
	luckUserID    int64
	luckBet       float64
	luckUseMax    bool
}

func (s *handlerServiceStub) Checkin(_ context.Context, userID int64) (*contract.CheckinResult, error) {
	s.checkinUserID = userID
	return &contract.CheckinResult{RewardAmount: 2.5}, nil
}

func (s *handlerServiceStub) LuckCheckin(_ context.Context, userID int64, betAmount float64, useMaxBalance bool) (*contract.CheckinResult, error) {
	s.luckUserID = userID
	s.luckBet = betAmount
	s.luckUseMax = useMaxBalance
	return &contract.CheckinResult{RewardAmount: 3.5, BetAmount: betAmount}, nil
}

func (s *handlerServiceStub) GetStatus(context.Context, int64) (*contract.CheckinStatus, error) {
	return &contract.CheckinStatus{CanCheckin: true}, nil
}

func (s *handlerServiceStub) GetCalendar(context.Context, int64) (*contract.CheckinCalendar, error) {
	return &contract.CheckinCalendar{}, nil
}

func TestHandlerCheckinRejectsMissingIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(&handlerServiceStub{}, nil)
	router.POST("/checkin", handler.Checkin)

	request := httptest.NewRequest(http.MethodPost, "/checkin", nil)
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusUnauthorized, writer.Code)
}

func TestHandlerLuckCheckinForwardsAuthenticatedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &handlerServiceStub{}
	router := gin.New()
	handler := NewHandler(stub, nil)
	router.POST("/checkin/luck", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		handler.LuckCheckin(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/checkin/luck", strings.NewReader(`{"bet_amount":2.5,"use_max_balance":true}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	require.Equal(t, int64(42), stub.luckUserID)
	require.Equal(t, 2.5, stub.luckBet)
	require.True(t, stub.luckUseMax)
	require.Contains(t, writer.Body.String(), `"reward_amount":3.5`)
}

func TestHandlerRejectsUnavailableService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/checkin/status", NewHandler(nil, nil).GetStatus)

	request := httptest.NewRequest(http.MethodGet, "/checkin/status", nil)
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusServiceUnavailable, writer.Code)
}

func TestModuleRegistersEstablishedCheckinRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(&handlerServiceStub{}, nil)
	module.RegisterRoutes(router.Group("/api/v1"))

	routes := make(map[string]struct{})
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}

	for _, route := range []string{
		"POST /api/v1/checkin",
		"POST /api/v1/checkin/luck",
		"GET /api/v1/checkin/status",
		"GET /api/v1/checkin/calendar",
		"GET /api/v1/checkin/blindbox/records",
	} {
		_, ok := routes[route]
		require.Truef(t, ok, "missing module route %s", route)
	}
}
