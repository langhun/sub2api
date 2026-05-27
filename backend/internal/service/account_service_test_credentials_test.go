//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type accountCredentialRepoStub struct {
	account *Account
	getErr  error
}

func (s *accountCredentialRepoStub) Create(ctx context.Context, account *Account) error {
	panic("unexpected Create call")
}

func (s *accountCredentialRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.account, nil
}

func (s *accountCredentialRepoStub) GetByIDs(ctx context.Context, ids []int64) ([]*Account, error) {
	panic("unexpected GetByIDs call")
}

func (s *accountCredentialRepoStub) ExistsByID(ctx context.Context, id int64) (bool, error) {
	panic("unexpected ExistsByID call")
}

func (s *accountCredentialRepoStub) GetByCRSAccountID(ctx context.Context, crsAccountID string) (*Account, error) {
	panic("unexpected GetByCRSAccountID call")
}

func (s *accountCredentialRepoStub) FindByExtraField(ctx context.Context, key string, value any) ([]Account, error) {
	panic("unexpected FindByExtraField call")
}

func (s *accountCredentialRepoStub) ListCRSAccountIDs(ctx context.Context) (map[string]int64, error) {
	panic("unexpected ListCRSAccountIDs call")
}

func (s *accountCredentialRepoStub) Update(ctx context.Context, account *Account) error {
	panic("unexpected Update call")
}

func (s *accountCredentialRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *accountCredentialRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *accountCredentialRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode, tier string) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (s *accountCredentialRepoStub) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	panic("unexpected ListByGroup call")
}

func (s *accountCredentialRepoStub) ListActive(ctx context.Context) ([]Account, error) {
	panic("unexpected ListActive call")
}

func (s *accountCredentialRepoStub) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListByPlatform call")
}

func (s *accountCredentialRepoStub) UpdateLastUsed(ctx context.Context, id int64) error {
	panic("unexpected UpdateLastUsed call")
}

func (s *accountCredentialRepoStub) BatchUpdateLastUsed(ctx context.Context, updates map[int64]time.Time) error {
	panic("unexpected BatchUpdateLastUsed call")
}

func (s *accountCredentialRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	panic("unexpected SetError call")
}

func (s *accountCredentialRepoStub) ClearError(ctx context.Context, id int64) error {
	panic("unexpected ClearError call")
}

func (s *accountCredentialRepoStub) SetSchedulable(ctx context.Context, id int64, schedulable bool) error {
	panic("unexpected SetSchedulable call")
}

func (s *accountCredentialRepoStub) AutoPauseExpiredAccounts(ctx context.Context, now time.Time) (int64, error) {
	panic("unexpected AutoPauseExpiredAccounts call")
}

func (s *accountCredentialRepoStub) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	panic("unexpected BindGroups call")
}

func (s *accountCredentialRepoStub) ListSchedulable(ctx context.Context) ([]Account, error) {
	panic("unexpected ListSchedulable call")
}

func (s *accountCredentialRepoStub) ListSchedulableByGroupID(ctx context.Context, groupID int64) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupID call")
}

func (s *accountCredentialRepoStub) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatform call")
}

func (s *accountCredentialRepoStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatform call")
}

func (s *accountCredentialRepoStub) ListSchedulableByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableByPlatforms call")
}

func (s *accountCredentialRepoStub) ListSchedulableByGroupIDAndPlatforms(ctx context.Context, groupID int64, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableByGroupIDAndPlatforms call")
}

func (s *accountCredentialRepoStub) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatform call")
}

func (s *accountCredentialRepoStub) ListSchedulableUngroupedByPlatforms(ctx context.Context, platforms []string) ([]Account, error) {
	panic("unexpected ListSchedulableUngroupedByPlatforms call")
}

func (s *accountCredentialRepoStub) SetRateLimited(ctx context.Context, id int64, resetAt time.Time) error {
	panic("unexpected SetRateLimited call")
}

func (s *accountCredentialRepoStub) SetModelRateLimit(ctx context.Context, id int64, scope string, resetAt time.Time, reason ...string) error {
	panic("unexpected SetModelRateLimit call")
}

func (s *accountCredentialRepoStub) SetOverloaded(ctx context.Context, id int64, until time.Time) error {
	panic("unexpected SetOverloaded call")
}

func (s *accountCredentialRepoStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	panic("unexpected SetTempUnschedulable call")
}

func (s *accountCredentialRepoStub) ClearTempUnschedulable(ctx context.Context, id int64) error {
	panic("unexpected ClearTempUnschedulable call")
}

func (s *accountCredentialRepoStub) ClearRateLimit(ctx context.Context, id int64) error {
	panic("unexpected ClearRateLimit call")
}

func (s *accountCredentialRepoStub) ClearAntigravityQuotaScopes(ctx context.Context, id int64) error {
	panic("unexpected ClearAntigravityQuotaScopes call")
}

func (s *accountCredentialRepoStub) ClearModelRateLimits(ctx context.Context, id int64) error {
	panic("unexpected ClearModelRateLimits call")
}

func (s *accountCredentialRepoStub) UpdateSessionWindow(ctx context.Context, id int64, start, end *time.Time, status string) error {
	panic("unexpected UpdateSessionWindow call")
}

func (s *accountCredentialRepoStub) UpdateExtra(ctx context.Context, id int64, updates map[string]any) error {
	panic("unexpected UpdateExtra call")
}

func (s *accountCredentialRepoStub) BulkUpdate(ctx context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	panic("unexpected BulkUpdate call")
}

func (s *accountCredentialRepoStub) IncrementQuotaUsed(ctx context.Context, id int64, amount float64) error {
	panic("unexpected IncrementQuotaUsed call")
}

func (s *accountCredentialRepoStub) ResetQuotaUsed(ctx context.Context, id int64) error {
	panic("unexpected ResetQuotaUsed call")
}

func TestAccountService_TestCredentials_RepoError(t *testing.T) {
	repoErr := errors.New("db down")
	svc := NewAccountService(&accountCredentialRepoStub{getErr: repoErr}, nil)

	err := svc.TestCredentials(context.Background(), 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "get account")
	require.ErrorIs(t, err, repoErr)
}

func TestAccountService_TestCredentials_NotFound(t *testing.T) {
	svc := NewAccountService(&accountCredentialRepoStub{account: nil}, nil)
	err := svc.TestCredentials(context.Background(), 2)
	require.ErrorIs(t, err, ErrAccountNotFound)
}

func TestAccountService_TestCredentials_OpenAIAPIKey(t *testing.T) {
	account := &Account{
		ID:       10,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "sk-valid",
		},
	}
	svc := NewAccountService(&accountCredentialRepoStub{account: account}, nil)
	require.NoError(t, svc.TestCredentials(context.Background(), account.ID))

	account.Credentials["api_key"] = "   "
	err := svc.TestCredentials(context.Background(), account.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "openai.api_key")
}

func TestAccountService_TestCredentials_OpenAIOAuth(t *testing.T) {
	account := &Account{
		ID:       11,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token",
		},
	}
	svc := NewAccountService(&accountCredentialRepoStub{account: account}, nil)
	require.NoError(t, svc.TestCredentials(context.Background(), account.ID))

	account.Credentials["access_token"] = ""
	err := svc.TestCredentials(context.Background(), account.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "openai.access_token")
}

func TestAccountService_TestCredentials_AnthropicBedrock(t *testing.T) {
	account := &Account{
		ID:       12,
		Platform: PlatformAnthropic,
		Type:     AccountTypeBedrock,
		Credentials: map[string]any{
			"aws_access_key_id":     "AKIA123",
			"aws_secret_access_key": "secret",
			"aws_region":            "us-east-1",
		},
	}
	svc := NewAccountService(&accountCredentialRepoStub{account: account}, nil)
	require.NoError(t, svc.TestCredentials(context.Background(), account.ID))

	account.Credentials["aws_region"] = ""
	err := svc.TestCredentials(context.Background(), account.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "anthropic.bedrock.aws_region")
}

func TestAccountService_TestCredentials_GeminiAndAntigravity(t *testing.T) {
	gemini := &Account{
		ID:       13,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "gemini-key",
		},
	}
	svc := NewAccountService(&accountCredentialRepoStub{account: gemini}, nil)
	require.NoError(t, svc.TestCredentials(context.Background(), gemini.ID))

	gemini.Credentials["api_key"] = ""
	err := svc.TestCredentials(context.Background(), gemini.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "gemini.api_key")

	ag := &Account{
		ID:       14,
		Platform: PlatformAntigravity,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "ag-token",
		},
	}
	svc = NewAccountService(&accountCredentialRepoStub{account: ag}, nil)
	require.NoError(t, svc.TestCredentials(context.Background(), ag.ID))

	ag.Credentials["access_token"] = ""
	err = svc.TestCredentials(context.Background(), ag.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "antigravity.access_token")
}

func TestAccountService_TestCredentials_UnsupportedCases(t *testing.T) {
	cases := []Account{
		{ID: 15, Platform: PlatformOpenAI, Type: AccountTypeServiceAccount, Credentials: map[string]any{"access_token": "x"}},
		{ID: 16, Platform: PlatformAnthropic, Type: AccountTypeServiceAccount, Credentials: map[string]any{"access_token": "x"}},
		{ID: 17, Platform: "unknown", Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "x"}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Platform+"-"+tc.Type, func(t *testing.T) {
			svc := NewAccountService(&accountCredentialRepoStub{account: &tc}, nil)
			err := svc.TestCredentials(context.Background(), tc.ID)
			require.Error(t, err)
			if tc.Platform == "unknown" {
				require.Contains(t, err.Error(), "unsupported platform")
			} else {
				require.Contains(t, err.Error(), "unsupported")
			}
		})
	}
}
