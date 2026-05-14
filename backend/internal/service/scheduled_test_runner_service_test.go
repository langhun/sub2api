//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type scheduledTestPlanRepoStub struct {
	lastUpdateID      int64
	lastUpdateRunAt   time.Time
	lastUpdateNextRun time.Time
	updateErr         error
}

type scheduledRunnerAccountRepoStub struct {
	accountRepoStub
	accountsByID    map[int64]*Account
	boundAccountID  int64
	boundGroupIDs   []int64
	bindErr         error
}

func (s *scheduledRunnerAccountRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if account, ok := s.accountsByID[id]; ok {
		return account, nil
	}
	return nil, errors.New("account not found")
}

func (s *scheduledRunnerAccountRepoStub) BindGroups(ctx context.Context, accountID int64, groupIDs []int64) error {
	s.boundAccountID = accountID
	s.boundGroupIDs = append([]int64(nil), groupIDs...)
	return s.bindErr
}

func (s *scheduledTestPlanRepoStub) Create(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	panic("unexpected Create call")
}

func (s *scheduledTestPlanRepoStub) GetByID(ctx context.Context, id int64) (*ScheduledTestPlan, error) {
	panic("unexpected GetByID call")
}

func (s *scheduledTestPlanRepoStub) ListByAccountID(ctx context.Context, accountID int64) ([]*ScheduledTestPlan, error) {
	panic("unexpected ListByAccountID call")
}

func (s *scheduledTestPlanRepoStub) ListDue(ctx context.Context, now time.Time) ([]*ScheduledTestPlan, error) {
	panic("unexpected ListDue call")
}

func (s *scheduledTestPlanRepoStub) Update(ctx context.Context, plan *ScheduledTestPlan) (*ScheduledTestPlan, error) {
	panic("unexpected Update call")
}

func (s *scheduledTestPlanRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *scheduledTestPlanRepoStub) UpdateAfterRun(ctx context.Context, id int64, lastRunAt time.Time, nextRunAt time.Time) error {
	s.lastUpdateID = id
	s.lastUpdateRunAt = lastRunAt
	s.lastUpdateNextRun = nextRunAt
	return s.updateErr
}

type scheduledTestResultRepoStub struct {
	created     *ScheduledTestResult
	prunedPlan  int64
	prunedKeep  int
	createErr   error
	pruneErr    error
	createCalls int
	pruneCalls  int
}

func (s *scheduledTestResultRepoStub) Create(ctx context.Context, result *ScheduledTestResult) (*ScheduledTestResult, error) {
	s.createCalls++
	if s.createErr != nil {
		return nil, s.createErr
	}
	cloned := *result
	s.created = &cloned
	return &cloned, nil
}

func (s *scheduledTestResultRepoStub) ListByPlanID(ctx context.Context, planID int64, limit int) ([]*ScheduledTestResult, error) {
	panic("unexpected ListByPlanID call")
}

func (s *scheduledTestResultRepoStub) PruneOldResults(ctx context.Context, planID int64, keepCount int) error {
	s.pruneCalls++
	s.prunedPlan = planID
	s.prunedKeep = keepCount
	return s.pruneErr
}

func TestScheduledTestService_SaveResultPersistsPlanAndPrunes(t *testing.T) {
	resultRepo := &scheduledTestResultRepoStub{}
	svc := NewScheduledTestService(nil, resultRepo)

	result := &ScheduledTestResult{
		Status:       "failed",
		ResponseText: "partial output",
		ErrorMessage: "API returned 401: bad token",
		LatencyMs:    12,
		StartedAt:    time.Now().Add(-time.Second),
		FinishedAt:   time.Now(),
	}

	err := svc.SaveResult(context.Background(), 123, 7, result)
	require.NoError(t, err)
	require.Equal(t, int64(123), result.PlanID)
	require.NotNil(t, resultRepo.created)
	require.Equal(t, int64(123), resultRepo.created.PlanID)
	require.Equal(t, "failed", resultRepo.created.Status)
	require.Equal(t, "API returned 401: bad token", resultRepo.created.ErrorMessage)
	require.Equal(t, int64(123), resultRepo.prunedPlan)
	require.Equal(t, 7, resultRepo.prunedKeep)
	require.Equal(t, 1, resultRepo.createCalls)
	require.Equal(t, 1, resultRepo.pruneCalls)
}

func TestScheduledTestRunnerService_RunOnePlanSavesResultAndUpdatesNextRun(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(planRepo, resultRepo)
	accountRepo := &mockAccountRepoForGemini{
		accountsByID: map[int64]*Account{
			88: {
				ID:          88,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Concurrency: 1,
				Credentials: map[string]any{"access_token": "test-token"},
			},
		},
	}
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusUnauthorized, `{"error":"bad token"}`),
		},
	}
	accountTestSvc := &AccountTestService{
		accountRepo:  accountRepo,
		httpUpstream: upstream,
	}
	runner := NewScheduledTestRunnerService(planRepo, scheduledSvc, accountTestSvc, nil, accountRepo, nil, nil)

	plan := &ScheduledTestPlan{
		ID:             66,
		AccountID:      88,
		ModelID:        "gpt-5.4",
		CronExpression: "*/5 * * * *",
		MaxResults:     9,
	}

	runner.runOnePlan(context.Background(), plan)

	require.NotNil(t, resultRepo.created)
	require.Equal(t, int64(66), resultRepo.created.PlanID)
	require.Equal(t, "failed", resultRepo.created.Status)
	require.Contains(t, resultRepo.created.ErrorMessage, "API returned 401")
	require.NotNil(t, resultRepo.created.HTTPStatusCode)
	require.Equal(t, http.StatusUnauthorized, *resultRepo.created.HTTPStatusCode)
	require.Equal(t, int64(66), planRepo.lastUpdateID)
	require.False(t, planRepo.lastUpdateRunAt.IsZero())
	require.False(t, planRepo.lastUpdateNextRun.IsZero())
	require.True(t, planRepo.lastUpdateNextRun.After(planRepo.lastUpdateRunAt))
}

func TestScheduledTestRunnerService_401DoubleCheckDeleteContractPendingMainline(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(nil, resultRepo)
	accountRepo := &scheduledRunnerAccountRepoStub{
		accountRepoStub: accountRepoStub{exists: true},
		accountsByID: map[int64]*Account{
			80: {
				ID:          80,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Concurrency: 1,
				Credentials: map[string]any{"access_token": "test-token"},
			},
		},
	}
	upstream := &queuedHTTPUpstream{
		responses: []*http.Response{
			newJSONResponse(http.StatusUnauthorized, `{"error":"bad token"}`),
			newJSONResponse(http.StatusUnauthorized, `{"error":"bad token"}`),
		},
	}
	accountTestSvc := &AccountTestService{
		accountRepo:  accountRepo,
		httpUpstream: upstream,
	}
	runner := NewScheduledTestRunnerService(planRepo, scheduledSvc, accountTestSvc, nil, accountRepo, nil, nil)

	plan := &ScheduledTestPlan{
		ID:                   77,
		AccountID:            80,
		ModelID:              "gpt-5.4",
		CronExpression:       "*/5 * * * *",
		MaxResults:           9,
		DeleteOnConfirmed401: true,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, []int64{80}, accountRepo.deletedIDs)
	require.Equal(t, 2, resultRepo.createCalls)
	require.NotNil(t, resultRepo.created)
	require.Equal(t, 2, resultRepo.created.AttemptNo)
	require.Equal(t, "deleted_account", resultRepo.created.ActionTaken)
	require.NotNil(t, resultRepo.created.HTTPStatusCode)
	require.Equal(t, http.StatusUnauthorized, *resultRepo.created.HTTPStatusCode)
}

func TestScheduledTestRunnerService_429SwitchGroupContractPendingMainline(t *testing.T) {
	planRepo := &scheduledTestPlanRepoStub{}
	resultRepo := &scheduledTestResultRepoStub{}
	scheduledSvc := NewScheduledTestService(nil, resultRepo)
	accountRepo := &scheduledRunnerAccountRepoStub{
		accountRepoStub: accountRepoStub{exists: true},
		accountsByID: map[int64]*Account{
			88: {
				ID:          88,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusError,
				Concurrency: 1,
				Credentials: map[string]any{"access_token": "test-token"},
				GroupIDs:    []int64{1001, 1002},
			},
		},
	}
	upstreamResp := newJSONResponse(http.StatusTooManyRequests, `{"error":{"type":"usage_limit_reached","message":"limit reached","resets_at":1777283883}}`)
	upstreamResp.Header.Set("x-codex-primary-used-percent", "100")
	upstreamResp.Header.Set("x-codex-primary-reset-after-seconds", "604800")
	upstreamResp.Header.Set("x-codex-primary-window-minutes", "10080")
	accountTestSvc := &AccountTestService{
		accountRepo:  accountRepo,
		httpUpstream: &queuedHTTPUpstream{responses: []*http.Response{upstreamResp}},
	}
	runner := NewScheduledTestRunnerService(planRepo, scheduledSvc, accountTestSvc, nil, accountRepo, nil, nil)

	fromID := int64(1002)
	toID := int64(1003)
	plan := &ScheduledTestPlan{
		ID:                88,
		AccountID:         88,
		ModelID:           "gpt-5.4",
		CronExpression:    "*/5 * * * *",
		MaxResults:        9,
		SwitchGroupFromID: &fromID,
		SwitchGroupToID:   &toID,
	}

	runner.runOnePlan(context.Background(), plan)

	require.Equal(t, int64(88), accountRepo.boundAccountID)
	require.Equal(t, []int64{1001, 1003}, accountRepo.boundGroupIDs)
	require.NotNil(t, resultRepo.created)
	require.Equal(t, "switched_group_1002_to_1003", resultRepo.created.ActionTaken)
	require.NotNil(t, resultRepo.created.HTTPStatusCode)
	require.Equal(t, http.StatusTooManyRequests, *resultRepo.created.HTTPStatusCode)
}
