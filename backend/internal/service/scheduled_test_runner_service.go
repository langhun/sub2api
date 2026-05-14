package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const scheduledTestDefaultMaxWorkers = 10

// ScheduledTestRunnerService periodically scans due test plans and executes them.
type ScheduledTestRunnerService struct {
	planRepo       ScheduledTestPlanRepository
	scheduledSvc   *ScheduledTestService
	accountTestSvc *AccountTestService
	rateLimitSvc   *RateLimitService
	accountRepo    AccountRepository
	groupRepo      GroupRepository
	cfg            *config.Config

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewScheduledTestRunnerService creates a new runner.
func NewScheduledTestRunnerService(
	planRepo ScheduledTestPlanRepository,
	scheduledSvc *ScheduledTestService,
	accountTestSvc *AccountTestService,
	rateLimitSvc *RateLimitService,
	accountRepo AccountRepository,
	groupRepo GroupRepository,
	cfg *config.Config,
) *ScheduledTestRunnerService {
	return &ScheduledTestRunnerService{
		planRepo:       planRepo,
		scheduledSvc:   scheduledSvc,
		accountTestSvc: accountTestSvc,
		rateLimitSvc:   rateLimitSvc,
		accountRepo:    accountRepo,
		groupRepo:      groupRepo,
		cfg:            cfg,
	}
}

// Start begins the cron ticker (every minute).
func (s *ScheduledTestRunnerService) Start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(scheduledTestCronParser), cron.WithLocation(loc))
		_, err := c.AddFunc("* * * * *", func() { s.runScheduled() })
		if err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] not started (invalid schedule): %v", err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] started (tick=every minute)")
	})
}

// Stop gracefully shuts down the cron scheduler.
func (s *ScheduledTestRunnerService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron != nil {
			ctx := s.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] cron stop timed out")
			}
		}
	})
}

func (s *ScheduledTestRunnerService) runScheduled() {
	// Delay 10s so execution lands at ~:10 of each minute instead of :00.
	time.Sleep(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()
	plans, err := s.planRepo.ListDue(ctx, now)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] ListDue error: %v", err)
		return
	}
	if len(plans) == 0 {
		return
	}

	logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] found %d due plans", len(plans))

	sem := make(chan struct{}, scheduledTestDefaultMaxWorkers)
	var wg sync.WaitGroup

	for _, plan := range plans {
		sem <- struct{}{}
		wg.Add(1)
		go func(p *ScheduledTestPlan) {
			defer wg.Done()
			defer func() { <-sem }()
			s.runOnePlan(ctx, p)
		}(plan)
	}

	wg.Wait()
}

func (s *ScheduledTestRunnerService) runOnePlan(ctx context.Context, plan *ScheduledTestPlan) {
	result, err := s.accountTestSvc.RunTestBackground(ctx, plan.AccountID, plan.ModelID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d RunTestBackground error: %v", plan.ID, err)
		return
	}

	deleteAfterRun, actionErr := s.applyPlanActions(ctx, plan, result)
	if actionErr != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d action error: %v", plan.ID, actionErr)
	}

	if err := s.scheduledSvc.SaveResult(ctx, plan.ID, plan.MaxResults, result); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d SaveResult error: %v", plan.ID, err)
	}

	// Auto-recover account if test succeeded and auto_recover is enabled.
	if result.Status == "success" && plan.AutoRecover {
		s.tryRecoverAccount(ctx, plan.AccountID, plan.ID)
	}

	nextRun, err := computeNextRun(plan.CronExpression, time.Now())
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d computeNextRun error: %v", plan.ID, err)
		return
	}

	if err := s.planRepo.UpdateAfterRun(ctx, plan.ID, time.Now(), nextRun); err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d UpdateAfterRun error: %v", plan.ID, err)
	}

	if deleteAfterRun && s.accountRepo != nil {
		if err := s.accountRepo.Delete(ctx, plan.AccountID); err != nil {
			logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d delete-after-run failed: %v", plan.ID, err)
		}
	}
}

func (s *ScheduledTestRunnerService) applyPlanActions(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult) (bool, error) {
	if plan == nil || result == nil {
		return false, nil
	}

	if result.Status == "success" {
		if err := s.handleSuccessful401Recovery(ctx, plan, result); err != nil {
			return false, err
		}
		return false, s.handleSuccessful429GroupRestore(ctx, plan, result)
	}

	if result.HTTPStatusCode != nil {
		switch *result.HTTPStatusCode {
		case 401:
			return s.handleScheduled401(ctx, plan, result)
		case 429:
			return false, s.handle429GroupSwitch(ctx, plan, result)
		}
	}

	return false, nil
}

func (s *ScheduledTestRunnerService) handleScheduled401(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult) (bool, error) {
	if !plan.DeleteOnConfirmed401 || s.accountRepo == nil {
		return false, nil
	}

	account, err := s.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil {
		return false, fmt.Errorf("load account for scheduled 401 handling: %w", err)
	}

	if hasPendingScheduled401Delete(account, plan.ID) {
		result.ActionTaken = appendScheduledTestAction(result.ActionTaken, "deleted_after_repeated_401")
		return true, nil
	}

	if err := s.setPendingScheduled401Delete(ctx, plan.AccountID, plan.ID); err != nil {
		return false, fmt.Errorf("record pending scheduled 401 delete: %w", err)
	}
	result.ActionTaken = appendScheduledTestAction(result.ActionTaken, "marked_error_after_401")
	return false, nil
}

func (s *ScheduledTestRunnerService) handleSuccessful401Recovery(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult) error {
	if !plan.DeleteOnConfirmed401 || s.accountRepo == nil {
		return nil
	}

	account, err := s.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil {
		return fmt.Errorf("load account for scheduled 401 recovery: %w", err)
	}

	if !hasPendingScheduled401Delete(account, plan.ID) {
		return nil
	}

	if err := s.accountRepo.ClearError(ctx, plan.AccountID); err != nil {
		return fmt.Errorf("clear account error after successful scheduled test: %w", err)
	}
	if err := s.clearPendingScheduled401Delete(ctx, plan.AccountID, plan.ID); err != nil {
		return fmt.Errorf("clear pending scheduled 401 delete: %w", err)
	}
	result.ActionTaken = appendScheduledTestAction(result.ActionTaken, "released_after_401_recovery")
	return nil
}

func (s *ScheduledTestRunnerService) handle429GroupSwitch(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult) error {
	if plan.SwitchGroupFromID == nil || plan.SwitchGroupToID == nil || s.accountRepo == nil {
		return nil
	}
	if *plan.SwitchGroupFromID == *plan.SwitchGroupToID {
		return nil
	}

	account, err := s.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil {
		return fmt.Errorf("load account for 429 group switch: %w", err)
	}

	groupIDs := extractAccountGroupIDs(account)

	updatedGroupIDs, changed := replaceGroupID(groupIDs, *plan.SwitchGroupFromID, *plan.SwitchGroupToID)
	if !changed {
		result.ActionTaken = "switch_group_skipped"
		return nil
	}

	if s.groupRepo != nil {
		if _, err := s.groupRepo.GetByID(ctx, *plan.SwitchGroupToID); err != nil {
			return fmt.Errorf("validate target group: %w", err)
		}
	}

	if err := s.accountRepo.BindGroups(ctx, plan.AccountID, updatedGroupIDs); err != nil {
		return fmt.Errorf("switch groups after 429: %w", err)
	}
	if err := s.setPending429Restore(ctx, plan.AccountID, plan.ID, *plan.SwitchGroupFromID); err != nil {
		result.ActionTaken = "switch_group_marker_failed"
		return fmt.Errorf("record pending 429 restore: %w", err)
	}
	result.ActionTaken = fmt.Sprintf("switched_group_%d_to_%d", *plan.SwitchGroupFromID, *plan.SwitchGroupToID)
	return nil
}

func (s *ScheduledTestRunnerService) handleSuccessful429GroupRestore(ctx context.Context, plan *ScheduledTestPlan, result *ScheduledTestResult) error {
	if plan.SwitchGroupFromID == nil || plan.SwitchGroupToID == nil || s.accountRepo == nil {
		return nil
	}

	account, err := s.accountRepo.GetByID(ctx, plan.AccountID)
	if err != nil {
		return fmt.Errorf("load account for 429 restore: %w", err)
	}

	pendingFromID, hasPending := getPending429Restore(account, plan.ID)
	if !hasPending || pendingFromID != *plan.SwitchGroupFromID {
		return nil
	}

	groupIDs := extractAccountGroupIDs(account)
	restoredGroupIDs, changed := replaceGroupID(groupIDs, *plan.SwitchGroupToID, *plan.SwitchGroupFromID)
	if !changed {
		if err := s.clearPending429Restore(ctx, plan.AccountID, plan.ID); err != nil {
			return fmt.Errorf("clear pending 429 restore after skipped restore: %w", err)
		}
		result.ActionTaken = "restore_group_skipped"
		return nil
	}

	if s.groupRepo != nil {
		if _, err := s.groupRepo.GetByID(ctx, *plan.SwitchGroupFromID); err != nil {
			return fmt.Errorf("validate restore group: %w", err)
		}
	}

	if err := s.accountRepo.BindGroups(ctx, plan.AccountID, restoredGroupIDs); err != nil {
		return fmt.Errorf("restore groups after successful test: %w", err)
	}
	if err := s.clearPending429Restore(ctx, plan.AccountID, plan.ID); err != nil {
		result.ActionTaken = "restore_group_marker_failed"
		return fmt.Errorf("clear pending 429 restore: %w", err)
	}
	result.ActionTaken = fmt.Sprintf("restored_group_%d_to_%d", *plan.SwitchGroupToID, *plan.SwitchGroupFromID)
	return nil
}

func replaceGroupID(groupIDs []int64, fromID int64, toID int64) ([]int64, bool) {
	if len(groupIDs) == 0 {
		return groupIDs, false
	}

	updated := append([]int64(nil), groupIDs...)
	changed := false
	hasTarget := false
	for _, groupID := range updated {
		if groupID == toID {
			hasTarget = true
			break
		}
	}

	for i, groupID := range updated {
		if groupID != fromID {
			continue
		}
		if hasTarget {
			updated = append(updated[:i], updated[i+1:]...)
		} else {
			updated[i] = toID
			hasTarget = true
		}
		changed = true
		break
	}

	return updated, changed
}

func appendScheduledTestAction(existing string, action string) string {
	if action == "" {
		return existing
	}
	if existing == "" {
		return action
	}
	return existing + "," + action
}

func extractAccountGroupIDs(account *Account) []int64 {
	if account == nil {
		return nil
	}

	groupIDs := append([]int64(nil), account.GroupIDs...)
	if len(groupIDs) > 0 {
		return groupIDs
	}

	if len(account.AccountGroups) == 0 {
		return nil
	}

	groupIDs = make([]int64, 0, len(account.AccountGroups))
	for _, group := range account.AccountGroups {
		groupIDs = append(groupIDs, group.GroupID)
	}
	return groupIDs
}

func scheduledTest429RestoreKey(planID int64) string {
	return fmt.Sprintf("scheduled_test_429_restore_group_%d", planID)
}

func scheduledTest401DeleteKey(planID int64) string {
	return fmt.Sprintf("scheduled_test_401_delete_pending_%d", planID)
}

func hasPendingScheduled401Delete(account *Account, planID int64) bool {
	if account == nil || account.Extra == nil {
		return false
	}

	raw, ok := account.Extra[scheduledTest401DeleteKey(planID)]
	if !ok || raw == nil {
		return false
	}

	return ParseExtraInt(raw) > 0
}

func getPending429Restore(account *Account, planID int64) (int64, bool) {
	if account == nil || account.Extra == nil {
		return 0, false
	}

	raw, ok := account.Extra[scheduledTest429RestoreKey(planID)]
	if !ok || raw == nil {
		return 0, false
	}

	groupID := int64(ParseExtraInt(raw))
	if groupID <= 0 {
		return 0, false
	}
	return groupID, true
}

func (s *ScheduledTestRunnerService) setPending429Restore(ctx context.Context, accountID int64, planID int64, fromGroupID int64) error {
	if s.accountRepo == nil {
		return nil
	}
	return s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{
		scheduledTest429RestoreKey(planID): fromGroupID,
	})
}

func (s *ScheduledTestRunnerService) clearPending429Restore(ctx context.Context, accountID int64, planID int64) error {
	if s.accountRepo == nil {
		return nil
	}
	return s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{
		scheduledTest429RestoreKey(planID): 0,
	})
}

func (s *ScheduledTestRunnerService) setPendingScheduled401Delete(ctx context.Context, accountID int64, planID int64) error {
	if s.accountRepo == nil {
		return nil
	}
	return s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{
		scheduledTest401DeleteKey(planID): 1,
	})
}

func (s *ScheduledTestRunnerService) clearPendingScheduled401Delete(ctx context.Context, accountID int64, planID int64) error {
	if s.accountRepo == nil {
		return nil
	}
	return s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{
		scheduledTest401DeleteKey(planID): 0,
	})
}

// tryRecoverAccount attempts to recover an account from recoverable runtime state.
func (s *ScheduledTestRunnerService) tryRecoverAccount(ctx context.Context, accountID int64, planID int64) {
	if s.rateLimitSvc == nil {
		return
	}

	recovery, err := s.rateLimitSvc.RecoverAccountAfterSuccessfulTest(ctx, accountID)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover failed: %v", planID, err)
		return
	}
	if recovery == nil {
		return
	}

	if recovery.ClearedError {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d recovered from error status", planID, accountID)
	}
	if recovery.ClearedRateLimit {
		logger.LegacyPrintf("service.scheduled_test_runner", "[ScheduledTestRunner] plan=%d auto-recover: account=%d cleared rate-limit/runtime state", planID, accountID)
	}
}
