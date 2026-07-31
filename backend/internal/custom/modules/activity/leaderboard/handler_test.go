package leaderboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandlerUsesModuleServiceForBalanceBoard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	reader := &balanceReaderStub{}
	handler := NewHandler(NewService(leaderboardSettingsStub{settings: contract.LeaderboardFeatureSettings{
		Enabled: true, BalanceEnabled: true,
	}}, Readers{Balance: reader}))
	router := gin.New()
	router.GET("/leaderboard/balance", handler.Balance)

	request := httptest.NewRequest(http.MethodGet, "/leaderboard/balance?page=1&page_size=10", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, reader.queries, 1)
	require.Equal(t, 1, reader.queries[0].Page)
	require.Equal(t, 10, reader.queries[0].PageSize)
}

func TestHandlerKeepsDisabledBoardHidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHandler(NewService(leaderboardSettingsStub{}, Readers{Balance: &balanceReaderStub{}}))
	router := gin.New()
	router.GET("/leaderboard/balance", handler.Balance)

	request := httptest.NewRequest(http.MethodGet, "/leaderboard/balance", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestModuleRegistersLegacyCompatiblePublicRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	module := NewModule(leaderboardSettingsStub{settings: contract.LeaderboardFeatureSettings{
		Enabled: true, BalanceEnabled: true, ConsumptionEnabled: true, CheckinEnabled: true,
	}}, Readers{
		Balance:     &balanceReaderStub{},
		Consumption: &consumptionReaderStub{},
		Checkin:     &checkinReaderStub{},
	})
	router := gin.New()
	module.RegisterRoutes(router)

	for _, path := range []string{
		"/api/v1/public/leaderboard/balance",
		"/api/v1/public/leaderboard/consumption",
		"/api/v1/public/leaderboard/checkin",
	} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}
