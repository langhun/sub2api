package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type setPrivacyAdminService struct {
	*stubAdminService
	account                 service.Account
	forceAntigravityPrivacy string
}

func (s *setPrivacyAdminService) GetAccount(_ context.Context, id int64) (*service.Account, error) {
	if s.account.ID == id {
		acc := s.account
		return &acc, nil
	}
	return s.stubAdminService.GetAccount(context.Background(), id)
}

func (s *setPrivacyAdminService) ForceAntigravityPrivacy(ctx context.Context, account *service.Account) string {
	if s.forceAntigravityPrivacy != "" {
		return s.forceAntigravityPrivacy
	}
	return s.stubAdminService.ForceAntigravityPrivacy(ctx, account)
}

func setupSetPrivacyRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/:id/set-privacy", handler.SetPrivacy)
	return router
}

func TestSetPrivacy_OpenAICFBlocked_ReturnsBadGateway(t *testing.T) {
	adminSvc := &setPrivacyAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       11,
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "openai-oauth",
		},
	}
	adminSvc.forceOpenAIPrivacyMode = service.PrivacyModeCFBlocked
	router := setupSetPrivacyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/11/set-privacy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Contains(t, rec.Body.String(), "Cloudflare")
}

func TestSetPrivacy_OpenAIFailed_ReturnsBadGateway(t *testing.T) {
	adminSvc := &setPrivacyAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       12,
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "openai-oauth",
		},
	}
	adminSvc.forceOpenAIPrivacyMode = service.PrivacyModeFailed
	router := setupSetPrivacyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/12/set-privacy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Contains(t, rec.Body.String(), "upstream request failed")
}

func TestSetPrivacy_AntigravityFailed_ReturnsBadGateway(t *testing.T) {
	adminSvc := &setPrivacyAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       13,
			Platform: service.PlatformAntigravity,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "antigravity-oauth",
		},
		forceAntigravityPrivacy: service.AntigravityPrivacyFailed,
	}
	router := setupSetPrivacyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/13/set-privacy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Contains(t, rec.Body.String(), "upstream request failed")
}

func TestSetPrivacy_OpenAINearExpiry_RefreshesBeforeSettingPrivacy(t *testing.T) {
	adminSvc := &setPrivacyAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       14,
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "openai-oauth",
			Credentials: map[string]any{
				"access_token":  "access-token-old",
				"refresh_token": "refresh-token-1",
				"expires_at":    time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	adminSvc.refreshedAccounts = map[int64]service.Account{
		14: {
			ID:       14,
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "openai-oauth",
			Credentials: map[string]any{
				"access_token":  "access-token-new",
				"refresh_token": "refresh-token-1",
				"expires_at":    time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	adminSvc.forceOpenAIPrivacyMode = service.PrivacyModeTrainingOff
	router := setupSetPrivacyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/14/set-privacy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, []int64{14}, adminSvc.refreshedAccountIDs)
	require.Equal(t, []int64{14}, adminSvc.forcedPrivacyIDs)
}

func TestSetPrivacy_OpenAIExpiredWithoutRefreshToken_ReturnsClearBadRequest(t *testing.T) {
	adminSvc := &setPrivacyAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       15,
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "openai-oauth",
			Credentials: map[string]any{
				"access_token": "access-token-old",
				"expires_at":   time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	router := setupSetPrivacyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/15/set-privacy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "refresh_token is missing")
	require.Empty(t, adminSvc.forcedPrivacyIDs)
	require.Empty(t, adminSvc.refreshedAccountIDs)
}

func TestSetPrivacy_OpenAIRefreshFailure_ReturnsRefreshError(t *testing.T) {
	adminSvc := &setPrivacyAdminService{
		stubAdminService: newStubAdminService(),
		account: service.Account{
			ID:       16,
			Platform: service.PlatformOpenAI,
			Type:     service.AccountTypeOAuth,
			Status:   service.StatusActive,
			Name:     "openai-oauth",
			Credentials: map[string]any{
				"access_token":  "access-token-old",
				"refresh_token": "refresh-token-1",
				"expires_at":    time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339),
			},
		},
	}
	adminSvc.refreshAccountErr = infraerrors.BadRequest("OPENAI_OAUTH_REFRESH_FAILED", "refresh_token_reused")
	router := setupSetPrivacyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/16/set-privacy", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "refresh_token_reused")
	require.Equal(t, []int64{16}, adminSvc.refreshedAccountIDs)
	require.Empty(t, adminSvc.forcedPrivacyIDs)
}
