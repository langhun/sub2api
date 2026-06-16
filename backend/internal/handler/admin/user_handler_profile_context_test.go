//go:build unit

package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUserHandlerGetByIDIncludesSignupSourceAndInviter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Date(2026, 6, 13, 0, 12, 59, 0, time.UTC)
	adminSvc := newStubAdminService()
	adminSvc.users = []service.User{
		{
			ID:           3,
			Email:        "3023208139@tiu.edu.cn",
			Username:     "",
			Role:         service.RoleUser,
			Status:       service.StatusActive,
			SignupSource: "google",
			InviterUser: &service.User{
				ID:       88,
				Email:    "inviter@example.com",
				Username: "boss-user",
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	handler := NewUserHandler(adminSvc, nil, nil, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/3", nil)

	handler.GetByID(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			SignupSource string `json:"signup_source"`
			InviterUser  *struct {
				ID       int64  `json:"id"`
				Email    string `json:"email"`
				Username string `json:"username"`
			} `json:"inviter_user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "google", resp.Data.SignupSource)
	require.NotNil(t, resp.Data.InviterUser)
	require.Equal(t, int64(88), resp.Data.InviterUser.ID)
	require.Equal(t, "inviter@example.com", resp.Data.InviterUser.Email)
	require.Equal(t, "boss-user", resp.Data.InviterUser.Username)
}

func TestUserHandlerGetBalanceHistoryIncludesAmountSources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	adminSvc := newStubAdminService()
	adminSvc.redeems = []service.RedeemCode{
		{
			ID:        1,
			Code:      "REG-001",
			Type:      service.AdjustmentTypeRegistration,
			Value:     50,
			Status:    service.StatusUsed,
			CreatedAt: time.Date(2026, 6, 13, 0, 12, 59, 0, time.UTC),
		},
	}
	adminSvc.balanceSourceSummary = &service.UserBalanceSourceSummary{
		Recharge:          120.5,
		RegistrationBonus: 50,
		InvitationBonus:   18,
		CheckinBonus:      9.6,
		AffiliateTransfer: 33.3,
		AdminAdjustment:   -5,
		TotalCredited:     226.4,
	}
	handler := NewUserHandler(adminSvc, nil, nil, nil)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Params = gin.Params{{Key: "id", Value: "3"}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users/3/balance-history", nil)

	handler.GetBalanceHistory(c)

	require.Equal(t, http.StatusOK, recorder.Code)

	var resp struct {
		Code int `json:"code"`
		Data struct {
			AmountSources *struct {
				Recharge          float64 `json:"recharge"`
				RegistrationBonus float64 `json:"registration_bonus"`
				InvitationBonus   float64 `json:"invitation_bonus"`
				CheckinBonus      float64 `json:"checkin_bonus"`
				AffiliateTransfer float64 `json:"affiliate_transfer"`
				AdminAdjustment   float64 `json:"admin_adjustment"`
				TotalCredited     float64 `json:"total_credited"`
			} `json:"amount_sources"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.NotNil(t, resp.Data.AmountSources)
	require.InDelta(t, 120.5, resp.Data.AmountSources.Recharge, 1e-9)
	require.InDelta(t, 50.0, resp.Data.AmountSources.RegistrationBonus, 1e-9)
	require.InDelta(t, 18.0, resp.Data.AmountSources.InvitationBonus, 1e-9)
	require.InDelta(t, 9.6, resp.Data.AmountSources.CheckinBonus, 1e-9)
	require.InDelta(t, 33.3, resp.Data.AmountSources.AffiliateTransfer, 1e-9)
	require.InDelta(t, -5.0, resp.Data.AmountSources.AdminAdjustment, 1e-9)
	require.InDelta(t, 226.4, resp.Data.AmountSources.TotalCredited, 1e-9)
}
