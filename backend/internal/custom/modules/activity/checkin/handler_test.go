package checkin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

type handlerProtectorStub struct {
	userID   int64
	remoteIP string
	proof    CaptchaProof
	err      error
}

func (s *handlerProtectorStub) Verify(_ context.Context, userID int64, remoteIP string, proof CaptchaProof) error {
	s.userID = userID
	s.remoteIP = remoteIP
	s.proof = proof
	return s.err
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

func TestHandlerCheckinVerifiesProtectionBeforeReward(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serviceStub := &handlerServiceStub{}
	protectorStub := &handlerProtectorStub{}
	router := gin.New()
	handler := NewHandler(serviceStub, nil, protectorStub)
	router.POST("/checkin", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		handler.Checkin(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/checkin", strings.NewReader(`{"turnstile_token":"proof-token"}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	require.Equal(t, int64(42), protectorStub.userID)
	require.Equal(t, "proof-token", protectorStub.proof.TurnstileToken)
	require.Equal(t, int64(42), serviceStub.checkinUserID)
}

func TestHandlerCheckinUsesTrustedProxyClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serviceStub := &handlerServiceStub{}
	protectorStub := &handlerProtectorStub{}
	router := gin.New()
	require.NoError(t, router.SetTrustedProxies(nil))
	handler := NewHandler(serviceStub, nil, protectorStub)
	router.POST("/checkin", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		handler.Checkin(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/checkin", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("CF-Connecting-IP", "1.1.1.1")
	request.Header.Set("X-Forwarded-For", "8.8.8.8")
	request.RemoteAddr = "203.0.113.20:4321"
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	require.Equal(t, "203.0.113.20", protectorStub.remoteIP)
}

func TestHandlerCheckinStopsBeforeRewardWhenProtectionFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serviceStub := &handlerServiceStub{}
	protectorStub := &handlerProtectorStub{err: &rateLimitError{
		err:        ErrCheckinRateLimited,
		retryAfter: 75 * time.Second,
	}}
	router := gin.New()
	handler := NewHandler(serviceStub, nil, protectorStub)
	router.POST("/checkin", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		handler.Checkin(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/checkin", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusTooManyRequests, writer.Code)
	require.Equal(t, "75", writer.Header().Get("Retry-After"))
	require.Zero(t, serviceStub.checkinUserID)
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

func TestHandlerLuckCheckinBindsCaptchaProofBeforeReward(t *testing.T) {
	gin.SetMode(gin.TestMode)
	serviceStub := &handlerServiceStub{}
	protectorStub := &handlerProtectorStub{}
	router := gin.New()
	handler := NewHandler(serviceStub, nil, protectorStub)
	router.POST("/checkin/luck", func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		handler.LuckCheckin(c)
	})

	request := httptest.NewRequest(http.MethodPost, "/checkin/luck", strings.NewReader(`{
		"bet_amount": 2.5,
		"use_max_balance": true,
		"tencent_captcha_ticket": "ticket-proof",
		"tencent_captcha_randstr": "@randstr"
	}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)

	require.Equal(t, http.StatusOK, writer.Code)
	require.Equal(t, int64(42), protectorStub.userID)
	require.Equal(t, "ticket-proof", protectorStub.proof.TencentTicket)
	require.Equal(t, "@randstr", protectorStub.proof.TencentRandstr)
	require.Equal(t, int64(42), serviceStub.luckUserID)
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
