package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type ungroupedAutoTestAccountRepoStub struct {
	accounts              []Account
	listWithFiltersGroup  int64
	listWithFiltersStatus string
	listWithFiltersCalls  int
	updatedExtraByID      map[int64]map[string]any
}

func (s *ungroupedAutoTestAccountRepoStub) Create(context.Context, *Account) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return nil, ErrAccountNotFound
}
func (s *ungroupedAutoTestAccountRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	return false, nil
}
func (s *ungroupedAutoTestAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) Update(context.Context, *Account) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) Delete(context.Context, int64) error    { return nil }
func (s *ungroupedAutoTestAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListWithFilters(_ context.Context, _ pagination.PaginationParams, _, _, status, _ string, groupID int64, _, _ string) ([]Account, *pagination.PaginationResult, error) {
	s.listWithFiltersCalls++
	s.listWithFiltersGroup = groupID
	s.listWithFiltersStatus = status
	out := append([]Account(nil), s.accounts...)
	return out, &pagination.PaginationResult{Total: int64(len(out)), Page: 1, PageSize: len(out)}, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListActive(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetError(context.Context, int64, string) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) ClearError(context.Context, int64) error       { return nil }
func (s *ungroupedAutoTestAccountRepoStub) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (s *ungroupedAutoTestAccountRepoStub) BindGroups(context.Context, int64, []int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ClearRateLimit(context.Context, int64) error { return nil }
func (s *ungroupedAutoTestAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ClearModelRateLimits(context.Context, int64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if s.updatedExtraByID == nil {
		s.updatedExtraByID = make(map[int64]map[string]any)
	}
	copied := make(map[string]any, len(updates))
	for k, v := range updates {
		copied[k] = v
	}
	s.updatedExtraByID[id] = copied
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (s *ungroupedAutoTestAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (s *ungroupedAutoTestAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

func TestShouldRunUngroupedAutoTest(t *testing.T) {
	now := time.Date(2026, 5, 14, 16, 0, 0, 0, time.UTC)

	require.True(t, shouldRunUngroupedAutoTest(now, nil))
	require.True(t, shouldRunUngroupedAutoTest(now, map[string]any{}))
	require.True(t, shouldRunUngroupedAutoTest(now, map[string]any{
		"auto_test_last_at": now.Add(-ungroupedAccountAutoTestRecencyInterval - time.Minute).Format(time.RFC3339),
	}))
	require.False(t, shouldRunUngroupedAutoTest(now, map[string]any{
		"auto_test_last_at": now.Add(-(ungroupedAccountAutoTestRecencyInterval - time.Minute)).Format(time.RFC3339),
	}))
}

func TestUngroupedAccountAutoTestServiceListCandidatesFiltersByRecencyAndSchedulable(t *testing.T) {
	now := time.Now().UTC()
	repo := &ungroupedAutoTestAccountRepoStub{
		accounts: []Account{
			{ID: 1, Status: StatusActive, Schedulable: true},
			{ID: 2, Status: StatusActive, Schedulable: true, Extra: map[string]any{
				"auto_test_last_at": now.Add(-(ungroupedAccountAutoTestRecencyInterval - 10*time.Minute)).Format(time.RFC3339),
			}},
			{ID: 3, Status: StatusActive, Schedulable: false},
			{ID: 4, Status: StatusDisabled, Schedulable: true},
		},
	}
	svc := NewUngroupedAccountAutoTestService(repo, &AccountTestService{}, nil)

	candidates, err := svc.listCandidates(context.Background())
	require.NoError(t, err)
	require.Equal(t, AccountListGroupUngrouped, repo.listWithFiltersGroup)
	require.Equal(t, StatusActive, repo.listWithFiltersStatus)
	require.Equal(t, 1, repo.listWithFiltersCalls)
	require.Len(t, candidates, 1)
	require.Equal(t, int64(1), candidates[0].ID)
}

func TestBuildUngroupedAutoTestExtraUpdates(t *testing.T) {
	started := time.Date(2026, 5, 14, 16, 30, 0, 0, time.UTC)
	statusCode := 403
	result := &ScheduledTestResult{
		Status:         "failed",
		ErrorMessage:   "forbidden",
		HTTPStatusCode: &statusCode,
		LatencyMs:      1234,
		FinishedAt:     started.Add(5 * time.Second),
	}

	updates := buildUngroupedAutoTestExtraUpdates(started, result)
	require.Equal(t, "failed", updates["auto_test_last_status"])
	require.Equal(t, "forbidden", updates["auto_test_last_error"])
	require.Equal(t, 403, updates["auto_test_http_status"])
	require.Equal(t, int64(1234), updates["auto_test_latency_ms"])
	require.Equal(t, started.Format(time.RFC3339), updates["auto_test_last_at"])
	require.Equal(t, started.Add(5*time.Second).Format(time.RFC3339), updates["auto_test_last_finished_at"])
}
