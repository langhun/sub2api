package openairt

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

type oauthServiceStub struct {
	validated string
	clientID  string
	info      *service.OpenAITokenInfo
	err       error
}

func (s *oauthServiceStub) RefreshTokenWithClientID(_ context.Context, token, _ string, clientID string) (*service.OpenAITokenInfo, error) {
	s.validated = token
	s.clientID = clientID
	if s.err != nil {
		return nil, s.err
	}
	return s.info, nil
}
func (s *oauthServiceStub) BuildAccountCredentials(info *service.OpenAITokenInfo) map[string]any {
	credentials := map[string]any{"access_token": info.AccessToken, "plan_type": info.PlanType}
	if info.RefreshToken != "" {
		credentials["refresh_token"] = info.RefreshToken
	}
	return credentials
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

func TestGetRefreshTokenReturnsNoStoreAndToken(t *testing.T) {
	account := &service.Account{
		ID:       1,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "rt-old-secret",
			"auth_mode":     "oauth",
		},
	}
	module := NewModule(&accountStoreStub{account: account}, &oauthServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/1/refresh-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "no-store", resp.Header().Get("Cache-Control"))
	var body struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &body))
	require.Equal(t, "rt-old-secret", body.Data.RefreshToken)
}

func TestReplaceRefreshTokenValidatesBeforeUpdatingAndPreservesSettings(t *testing.T) {
	account := &service.Account{
		ID:       2,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "rt-old-secret",
			"auth_mode":     "oauth",
			"client_id":     "client-1",
			"model_mapping": map[string]string{"gpt": "gpt-4"},
		},
	}
	store := &accountStoreStub{account: account}
	oauth := &oauthServiceStub{info: &service.OpenAITokenInfo{AccessToken: "at-new-secret", RefreshToken: "rt-rotated-secret", PlanType: "pro"}}
	invalidator := &invalidatorStub{}
	module := NewModule(store, oauth, invalidator)

	// Keep the test body explicit so no production token can be copied into fixtures.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/admin/openai/accounts/2/refresh-token", strings.NewReader(`{"refresh_token":"rt-new-secret"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "rt-new-secret", oauth.validated)
	require.Equal(t, "client-1", oauth.clientID)
	require.Equal(t, "at-new-secret", account.Credentials["access_token"])
	require.Equal(t, "rt-rotated-secret", account.Credentials["refresh_token"])
	require.Equal(t, map[string]string{"gpt": "gpt-4"}, account.Credentials["model_mapping"])
	require.NotNil(t, store.updatedInput)
	require.Equal(t, 1, store.clearCalls)
	require.Equal(t, 1, invalidator.calls)
}

func TestReplaceRefreshTokenUsesSubmittedTokenWhenProviderDoesNotRotate(t *testing.T) {
	account := &service.Account{
		ID:          3,
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Credentials: map[string]any{"refresh_token": "rt-old-secret", "auth_mode": "oauth"},
	}
	store := &accountStoreStub{account: account}
	oauth := &oauthServiceStub{info: &service.OpenAITokenInfo{AccessToken: "at-new-secret"}}
	module := NewModule(store, oauth, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/admin/openai/accounts/3/refresh-token", strings.NewReader(`{"refresh_token":"rt-new-secret"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "rt-new-secret", account.Credentials["refresh_token"])
}

func TestReplaceRefreshTokenDoesNotWriteWhenValidationFails(t *testing.T) {
	account := &service.Account{
		ID:       3,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "rt-old-secret",
			"auth_mode":     "oauth",
		},
	}
	store := &accountStoreStub{account: account}
	oauth := &oauthServiceStub{err: errors.New("invalid token")}
	module := NewModule(store, oauth, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/admin/openai/accounts/3/refresh-token", strings.NewReader(`{"refresh_token":"rt-invalid"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)

	require.Equal(t, http.StatusInternalServerError, resp.Code)
	require.Nil(t, store.updatedInput)
	require.Equal(t, "rt-old-secret", account.Credentials["refresh_token"])
}

func TestRTRouteRejectsPATAccount(t *testing.T) {
	account := &service.Account{ID: 4, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"refresh_token": "rt-secret", "access_token": "at-secret", "auth_mode": "personalAccessToken"}}
	module := NewModule(&accountStoreStub{account: account}, &oauthServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/4/refresh-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestRTRouteRejectsCredentialShadowAccount(t *testing.T) {
	parentID := int64(8)
	account := &service.Account{
		ID:              5,
		Platform:        service.PlatformOpenAI,
		Type:            service.AccountTypeOAuth,
		ParentAccountID: &parentID,
		Credentials: map[string]any{
			"refresh_token": "rt-shadow-secret",
			"auth_mode":     "oauth",
		},
	}
	module := NewModule(&accountStoreStub{account: account}, &oauthServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/5/refresh-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestRTRouteRejectsMissingRefreshToken(t *testing.T) {
	account := &service.Account{ID: 6, Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Credentials: map[string]any{"auth_mode": "oauth"}}
	module := NewModule(&accountStoreStub{account: account}, &oauthServiceStub{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/custom/admin/openai/accounts/6/refresh-token", nil)
	resp := httptest.NewRecorder()
	testRouter(module).ServeHTTP(resp, req)
	require.Equal(t, http.StatusBadRequest, resp.Code)
}
