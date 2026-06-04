package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type lotteryHandlerServiceStub struct {
	currentIssue *service.Issue
	jackpot      *service.LotteryJackpotView
	betResult    *service.LotteryBetResult
	orders       []service.LotteryOrderView

	createInput service.LotteryBetInput
	orderQuery  service.LotteryOrderQuery
}

func (s *lotteryHandlerServiceStub) CreateBet(_ context.Context, input service.LotteryBetInput) (*service.LotteryBetResult, error) {
	s.createInput = input
	return s.betResult, nil
}

func (s *lotteryHandlerServiceStub) GetCurrentIssue(_ context.Context, lotteryType string) (*service.Issue, error) {
	return s.currentIssue, nil
}

func (s *lotteryHandlerServiceStub) GetMyOrders(_ context.Context, query service.LotteryOrderQuery) ([]service.LotteryOrderView, error) {
	s.orderQuery = query
	return s.orders, nil
}

func (s *lotteryHandlerServiceStub) GetJackpot(_ context.Context, lotteryType string) (*service.LotteryJackpotView, error) {
	return s.jackpot, nil
}

func TestLotteryHandlerGetCurrentSuccess(t *testing.T) {
	service.SetDefaultIdempotencyCoordinator(nil)
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 6, 4, 21, 15, 0, 0, time.UTC)
	stub := &lotteryHandlerServiceStub{
		currentIssue: &service.Issue{
			LotteryType: service.LotteryTypeSSQ,
			IssueNo:     "2026060",
			OpenTime:    now,
			CutoffTime:  now.Add(-10 * time.Minute),
			IsClosed:    false,
		},
		jackpot: &service.LotteryJackpotView{
			LotteryType: service.LotteryTypeSSQ,
			Balance:     decimal.NewFromInt(10000000),
		},
	}
	h := &LotteryHandler{lotteryService: stub}
	w, c := lotteryHandlerTestContext(http.MethodGet, "/api/v1/lottery/current", "", 42)

	h.GetCurrent(c)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			IssueNo        string `json:"issue_no"`
			LotteryType    string `json:"lottery_type"`
			JackpotBalance string `json:"jackpot_balance"`
			IsClosed       bool   `json:"is_closed"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Equal(t, "2026060", body.Data.IssueNo)
	require.Equal(t, service.LotteryTypeSSQ, body.Data.LotteryType)
	require.Equal(t, "10000000", body.Data.JackpotBalance)
	require.False(t, body.Data.IsClosed)
}

func TestLotteryHandlerBetSuccess(t *testing.T) {
	service.SetDefaultIdempotencyCoordinator(nil)
	gin.SetMode(gin.TestMode)
	stub := &lotteryHandlerServiceStub{
		betResult: &service.LotteryBetResult{
			LotteryType: service.LotteryTypeSSQ,
			IssueNo:     "2026060",
			OrderIDs:    []int64{88},
			Cost:        decimal.NewFromInt(100),
			Status:      "pending",
		},
	}
	h := &LotteryHandler{lotteryService: stub}
	w, c := lotteryHandlerTestContext(http.MethodPost, "/api/v1/lottery/bet", `{"red_balls":[1,8,12,18,25,33],"blue_ball":9}`, 42)
	c.Request.Header.Set("Idempotency-Key", "lottery-bet-1")

	h.Bet(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(42), stub.createInput.UserID)
	require.Equal(t, service.LotteryTypeSSQ, stub.createInput.LotteryType)
	require.Equal(t, []string{"1", "8", "12", "18", "25", "33"}, stub.createInput.RedBalls)
	require.Equal(t, "9", stub.createInput.BlueBall)
	require.Equal(t, "lottery-bet-1", stub.createInput.IdempotencyKey)

	var body struct {
		Code int `json:"code"`
		Data struct {
			OrderID     int64   `json:"order_id"`
			OrderIDs    []int64 `json:"order_ids"`
			IssueNo     string  `json:"issue_no"`
			LotteryType string  `json:"lottery_type"`
			Cost        string  `json:"cost"`
			Status      string  `json:"status"`
			CreatedAt   string  `json:"created_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, int64(88), body.Data.OrderID)
	require.Equal(t, []int64{88}, body.Data.OrderIDs)
	require.Equal(t, "2026060", body.Data.IssueNo)
	require.Equal(t, service.LotteryTypeSSQ, body.Data.LotteryType)
	require.Equal(t, "100", body.Data.Cost)
	require.Equal(t, "pending", body.Data.Status)
	require.NotEmpty(t, body.Data.CreatedAt)
}

func TestLotteryHandlerBetRejectsInvalidPayload(t *testing.T) {
	service.SetDefaultIdempotencyCoordinator(nil)
	gin.SetMode(gin.TestMode)
	stub := &lotteryHandlerServiceStub{}
	h := &LotteryHandler{lotteryService: stub}
	w, c := lotteryHandlerTestContext(http.MethodPost, "/api/v1/lottery/bet", `{"red_balls":["bad"],"blue_ball":9}`, 42)

	h.Bet(c)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Zero(t, stub.createInput.UserID)
}

func TestLotteryHandlerGetOrdersSuccess(t *testing.T) {
	service.SetDefaultIdempotencyCoordinator(nil)
	gin.SetMode(gin.TestMode)
	createdAt := time.Date(2026, 6, 4, 9, 30, 0, 0, time.UTC)
	stub := &lotteryHandlerServiceStub{
		orders: []service.LotteryOrderView{
			{
				ID:          99,
				LotteryType: service.LotteryTypeSSQ,
				IssueNo:     "2026060",
				RedBalls:    []string{"01", "08", "12", "18", "25", "33"},
				BlueBall:    "09",
				Cost:        decimal.NewFromInt(100),
				Reward:      decimal.Zero,
				Status:      "pending",
				CreatedAt:   createdAt,
			},
		},
	}
	h := &LotteryHandler{lotteryService: stub}
	w, c := lotteryHandlerTestContext(http.MethodGet, "/api/v1/lottery/orders?issue_no=2026060", "", 42)

	h.GetOrders(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, int64(42), stub.orderQuery.UserID)
	require.Equal(t, service.LotteryTypeSSQ, stub.orderQuery.LotteryType)
	require.Equal(t, "2026060", stub.orderQuery.IssueNo)

	var body struct {
		Code int `json:"code"`
		Data []struct {
			OrderID     int64    `json:"order_id"`
			IssueNo     string   `json:"issue_no"`
			LotteryType string   `json:"lottery_type"`
			RedBalls    []string `json:"red_balls"`
			BlueBall    string   `json:"blue_ball"`
			Cost        string   `json:"cost"`
			Status      string   `json:"status"`
			Reward      string   `json:"reward"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body.Data, 1)
	require.Equal(t, int64(99), body.Data[0].OrderID)
	require.Equal(t, []string{"01", "08", "12", "18", "25", "33"}, body.Data[0].RedBalls)
	require.Equal(t, "09", body.Data[0].BlueBall)
	require.Equal(t, "100", body.Data[0].Cost)
	require.Equal(t, "pending", body.Data[0].Status)
}

func TestLotteryHandlerUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &LotteryHandler{}
	w, c := lotteryHandlerTestContext(http.MethodGet, "/api/v1/lottery/current", "", 0)

	h.GetCurrent(c)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func lotteryHandlerTestContext(method, path, body string, userID int64) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body == "" {
		c.Request = httptest.NewRequest(method, path, nil)
	} else {
		c.Request = httptest.NewRequest(method, path, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	if userID > 0 {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: userID})
	}
	return w, c
}
