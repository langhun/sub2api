//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type refreshAccountRepoStub struct {
	mockAccountRepoForGemini
	account             *Account
	updateCredentialsID int64
	updateCredentials   map[string]any
	updateCalls         int
}

func (r *refreshAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.account == nil || r.account.ID != id {
		return nil, errors.New("account not found")
	}
	return r.account, nil
}

func (r *refreshAccountRepoStub) UpdateCredentials(_ context.Context, id int64, credentials map[string]any) error {
	r.updateCalls++
	r.updateCredentialsID = id
	r.updateCredentials = cloneCredentials(credentials)
	if r.account != nil && r.account.ID == id {
		r.account.Credentials = cloneCredentials(credentials)
	}
	return nil
}

type openAIOAuthClientRefreshStub struct {
	refreshResp *openai.TokenResponse
	refreshErr  error
}

func (s *openAIOAuthClientRefreshStub) ExchangeCode(context.Context, string, string, string, string, string) (*openai.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openAIOAuthClientRefreshStub) RefreshToken(context.Context, string, string) (*openai.TokenResponse, error) {
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}
	if s.refreshResp != nil {
		return s.refreshResp, nil
	}
	return &openai.TokenResponse{
		AccessToken:  "new-access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "new-refresh-token",
	}, nil
}

func (s *openAIOAuthClientRefreshStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openai.TokenResponse, error) {
	return s.RefreshToken(ctx, refreshToken, proxyURL)
}

func TestAdminService_RefreshAccountCredentials_OpenAISuccess(t *testing.T) {
	account := &Account{
		ID:       101,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"refresh_token": "old-refresh-token",
			"access_token":  "old-access-token",
			"custom_field":  "keep-me",
		},
	}
	repo := &refreshAccountRepoStub{account: account}
	openaiSvc := NewOpenAIOAuthService(nil, &openAIOAuthClientRefreshStub{})
	svc := &adminServiceImpl{
		accountRepo:           repo,
		openAIOAuthService:    openaiSvc,
		tokenCacheInvalidator: nil,
	}

	updated, err := svc.RefreshAccountCredentials(context.Background(), 101)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, 1, repo.updateCalls)
	require.Equal(t, int64(101), repo.updateCredentialsID)
	require.Equal(t, "new-access-token", updated.GetCredential("access_token"))
	require.Equal(t, "new-refresh-token", updated.GetCredential("refresh_token"))
	require.Equal(t, "keep-me", updated.GetCredential("custom_field"))
	require.NotEmpty(t, updated.GetCredential("expires_at"))
}

func TestAdminService_RefreshAccountCredentials_UnsupportedReturnsError(t *testing.T) {
	repo := &refreshAccountRepoStub{
		account: &Account{
			ID:       202,
			Platform: PlatformOpenAI,
			Type:     AccountTypeAPIKey,
		},
	}
	svc := &adminServiceImpl{
		accountRepo: repo,
	}

	updated, err := svc.RefreshAccountCredentials(context.Background(), 202)
	require.Error(t, err)
	require.Nil(t, updated)
	require.Contains(t, err.Error(), "ACCOUNT_REFRESH_UNSUPPORTED")
	require.Equal(t, 0, repo.updateCalls)
}
