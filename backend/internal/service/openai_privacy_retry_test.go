package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/imroc/req/v3"
	"github.com/stretchr/testify/require"
)

type openAIPrivacyRetryAccountRepoStub struct{}

func (s *openAIPrivacyRetryAccountRepoStub) Create(context.Context, *Account) error { return nil }
func (s *openAIPrivacyRetryAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return nil, errors.New("not implemented")
}
func (s *openAIPrivacyRetryAccountRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	return false, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) Update(context.Context, *Account) error { return nil }
func (s *openAIPrivacyRetryAccountRepoStub) Delete(context.Context, int64) error    { return nil }
func (s *openAIPrivacyRetryAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string, string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListActive(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (s *openAIPrivacyRetryAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) SetError(context.Context, int64, string) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ClearError(context.Context, int64) error { return nil }
func (s *openAIPrivacyRetryAccountRepoStub) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) BindGroups(context.Context, int64, []int64) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ClearRateLimit(context.Context, int64) error { return nil }
func (s *openAIPrivacyRetryAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ClearModelRateLimits(context.Context, int64) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (s *openAIPrivacyRetryAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (s *openAIPrivacyRetryAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

type openAIPrivacyRetryTokenRefreshRepoStub struct {
	openAIPrivacyRetryAccountRepoStub
}

func TestAdminService_EnsureOpenAIPrivacy_RetriesNonSuccessModes(t *testing.T) {
	t.Parallel()

	for _, mode := range []string{"", PrivacyModeFailed, PrivacyModeCFBlocked} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			privacyCalls := 0
			svc := &adminServiceImpl{
				accountRepo: &openAIPrivacyRetryAccountRepoStub{},
				privacyClientFactory: func(proxyURL string) (*req.Client, error) {
					privacyCalls++
					return nil, errors.New("factory failed")
				},
			}

			account := &Account{
				ID:       101,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"access_token": "token-1",
				},
				Extra: map[string]any{
					"privacy_mode": mode,
				},
			}

			got := svc.EnsureOpenAIPrivacy(context.Background(), account)

			require.Equal(t, PrivacyModeFailed, got)
			require.Equal(t, 1, privacyCalls)
		})
	}
}

func TestTokenRefreshService_ensureOpenAIPrivacy_RetriesNonSuccessModes(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		TokenRefresh: config.TokenRefreshConfig{
			MaxRetries:          1,
			RetryBackoffSeconds: 0,
		},
	}

	for _, mode := range []string{"", PrivacyModeFailed, PrivacyModeCFBlocked} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()

			service := NewTokenRefreshService(&openAIPrivacyRetryTokenRefreshRepoStub{}, nil, nil, nil, nil, nil, nil, cfg, nil)
			privacyCalls := 0
			service.SetPrivacyDeps(func(proxyURL string) (*req.Client, error) {
				privacyCalls++
				return nil, errors.New("factory failed")
			}, nil)

			account := &Account{
				ID:       202,
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"access_token": "token-2",
				},
				Extra: map[string]any{
					"privacy_mode": mode,
				},
			}

			service.ensureOpenAIPrivacy(context.Background(), account)

			require.Equal(t, 1, privacyCalls)
		})
	}
}
