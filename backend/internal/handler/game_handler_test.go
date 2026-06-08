//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type gameHandlerServiceStub struct {
	hallStatus *service.GameHallStatus
	exchange   *service.GameExchangeResult
	lastInput  service.GameExchangeInput
}

func (s *gameHandlerServiceStub) GetHallStatus(_ context.Context, _ int64) (*service.GameHallStatus, error) {
	return s.hallStatus, nil
}

func (s *gameHandlerServiceStub) Exchange(_ context.Context, input service.GameExchangeInput) (*service.GameExchangeResult, error) {
	s.lastInput = input
	return s.exchange, nil
}

func (s *gameHandlerServiceStub) Play(_ context.Context, _ service.GamePlayInput) (*service.GamePlayResult, error) {
	return nil, nil
}

func TestGameHandler_GetHallStatus_Unauthenticated401(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewGameHandler(&gameHandlerServiceStub{})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/games/hall", nil)

	h.GetHallStatus(c)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestGameHandler_GetHallStatus_ReturnsDGAndJackpot(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewGameHandler(&gameHandlerServiceStub{
		hallStatus: &service.GameHallStatus{
			MainBalance:    88,
			DGBalance:      12,
			JackpotBalance: 345,
			Games: []service.GameInfo{
				{Type: service.GameTypeSlots, Name: "Slots"},
			},
		},
	})
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/games/hall", nil)
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 1})

	h.GetHallStatus(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Data struct {
			MainBalance    float64              `json:"main_balance"`
			DGBalance      float64              `json:"dg_balance"`
			JackpotBalance float64              `json:"jackpot_balance"`
			Games          []service.GameInfo   `json:"games"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 88.0, resp.Data.MainBalance)
	require.Equal(t, 12.0, resp.Data.DGBalance)
	require.Equal(t, 345.0, resp.Data.JackpotBalance)
	require.Len(t, resp.Data.Games, 1)
	require.Equal(t, service.GameTypeSlots, resp.Data.Games[0].Type)
}

func TestGameHandler_Exchange_UsesAuthenticatedUserAndRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	stub := &gameHandlerServiceStub{
		exchange: &service.GameExchangeResult{
			Direction:        service.GameExchangeBalanceToDG,
			Amount:           20,
			MainBalanceAfter: 60,
			DGBalanceAfter:   25,
		},
	}
	h := NewGameHandler(stub)
	body := bytes.NewBufferString(`{"direction":"balance_to_dg","amount":20}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/games/exchange", body)
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Idempotency-Key", "exchange-1")
	c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 7})

	h.Exchange(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(7), stub.lastInput.UserID)
	require.Equal(t, service.GameExchangeBalanceToDG, stub.lastInput.Direction)
	require.Equal(t, 20.0, stub.lastInput.Amount)
	require.Equal(t, "exchange-1", stub.lastInput.IdempotencyKey)
}
