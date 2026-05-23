package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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
