package openaiat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type accountStoreStub struct {
	account      *service.Account
	updatedInput *service.UpdateAccountInput
	clearCalls   int
}

func (s *accountStoreStub) GetAccount(context.Context, int64) (*service.Account, error) {
	if s.account == nil {
		return nil, errors.New("not found")
	}
	return s.account, nil
}
func (s *accountStoreStub) GetProxy(context.Context, int64) (*service.Proxy, error) { return nil, nil }
func (s *accountStoreStub) UpdateAccount(_ context.Context, _ int64, input *service.UpdateAccountInput) (*service.Account, error) {
	s.updatedInput = input
	for key, value := range input.Credentials {
		s.account.Credentials[key] = value
	}
	return s.account, nil
}
func (s *accountStoreStub) ClearAccountError(context.Context, int64) (*service.Account, error) {
	s.clearCalls++
	return s.account, nil
}

type patServiceStub struct {
	validated string
	info      *service.OpenAITokenInfo
	err       error
}

func (s *patServiceStub) ValidateCodexPersonalAccessToken(_ context.Context, token, _ string) (*service.OpenAITokenInfo, error) {
	s.validated = token
	if s.err != nil {
		return nil, s.err
	}
	return s.info, nil
}
func (s *patServiceStub) BuildAccountCredentials(info *service.OpenAITokenInfo) map[string]any {
	return map[string]any{
		"access_token": info.AccessToken,
		"auth_mode":    "personalAccessToken",
		"plan_type":    info.PlanType,
	}
}

type invalidatorStub struct{ calls int }

func (s *invalidatorStub) InvalidateToken(context.Context, *service.Account) error {
	s.calls++
	return nil
}

func testRouter(module *Module) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/custom/admin")
	module.RegisterRoutes(group)
	return r
}

func TestGetAccessTokenReturnsNoStoreAndToken(t *testing.T) {
	account := &service.Account{
		ID:       1,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "at-old-secret",
			"auth_mode":    "personalAccessToken",
		},
	}
	module := NewModule(&accountStoreStub{account: account}, &patServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/1/access-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
	var body struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "at-old-secret", body.Data.AccessToken)
}

func TestReplaceAccessTokenValidatesBeforeUpdatingAndPreservesSettings(t *testing.T) {
	account := &service.Account{
		ID:       2,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "at-old-secret",
			"auth_mode":    "personalAccessToken",
			"model_mapping": map[string]string{"gpt": "gpt-4"},
		},
	}
	store := &accountStoreStub{account: account}
	pat := &patServiceStub{info: &service.OpenAITokenInfo{AccessToken: "at-new-secret", AuthMode: service.OpenAIAuthModePersonalAccessToken, PlanType: "pro"}}
	invalidator := &invalidatorStub{}
	module := NewModule(store, pat, invalidator)

	// Keep the test body explicit so no production token can be copied into fixtures.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/admin/openai/accounts/2/access-token", strings.NewReader(`{"access_token":"at-new-secret"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "at-new-secret", pat.validated)
	require.Equal(t, "at-new-secret", account.Credentials["access_token"])
	require.Equal(t, map[string]string{"gpt": "gpt-4"}, account.Credentials["model_mapping"])
	require.NotNil(t, store.updatedInput)
	require.Equal(t, 1, store.clearCalls)
	require.Equal(t, 1, invalidator.calls)
}

func TestReplaceAccessTokenDoesNotWriteWhenValidationFails(t *testing.T) {
	account := &service.Account{
		ID:       3,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "at-old-secret",
			"auth_mode":    "personalAccessToken",
		},
	}
	store := &accountStoreStub{account: account}
	pat := &patServiceStub{err: errors.New("invalid token")}
	module := NewModule(store, pat, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/admin/openai/accounts/3/access-token", strings.NewReader(`{"access_token":"at-invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Nil(t, store.updatedInput)
	require.Equal(t, "at-old-secret", account.Credentials["access_token"])
}

func TestPATRouteRejectsNonPATAccount(t *testing.T) {
	account := &service.Account{ID: 4, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"auth_mode": "oauth"}}
	module := NewModule(&accountStoreStub{account: account}, &patServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/4/access-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestPATRouteRejectsCredentialShadowAccount(t *testing.T) {
	parentID := int64(8)
	account := &service.Account{
		ID:              5,
		Platform:        service.PlatformOpenAI,
		Type:            service.AccountTypeOAuth,
		ParentAccountID: &parentID,
		Credentials: map[string]any{
			"access_token": "at-shadow-secret",
			"auth_mode":    "personalAccessToken",
		},
	}
	module := NewModule(&accountStoreStub{account: account}, &patServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/5/access-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}
