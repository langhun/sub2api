package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	ungroupedAccountAutoTestTickInterval    = 2 * time.Minute
	ungroupedAccountAutoTestConcurrency     = 8
	ungroupedAccountAutoTestMaxPerRun       = 1000
	ungroupedAccountAutoTestPageSize        = 200
	ungroupedAccountAutoTestRecencyInterval = 6 * time.Hour
)

// UngroupedAccountAutoTestService periodically tests ungrouped accounts in the
// background so newly imported accounts can be activated and auto-bound without
// manual intervention.
type UngroupedAccountAutoTestService struct {
	accountRepo    AccountRepository
	accountTestSvc *AccountTestService
	cfg            *config.Config

	stopCh    chan struct{}
	stopOnce  sync.Once
	startOnce sync.Once
	wg        sync.WaitGroup
}

func NewUngroupedAccountAutoTestService(
	accountRepo AccountRepository,
	accountTestSvc *AccountTestService,
	cfg *config.Config,
) *UngroupedAccountAutoTestService {
	return &UngroupedAccountAutoTestService{
		accountRepo:    accountRepo,
		accountTestSvc: accountTestSvc,
		cfg:            cfg,
		stopCh:         make(chan struct{}),
	}
}

func (s *UngroupedAccountAutoTestService) Start() {
	if s == nil || s.accountRepo == nil || s.accountTestSvc == nil {
		return
	}
	s.startOnce.Do(func() {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.run()
		}()
	})
}

func (s *UngroupedAccountAutoTestService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *UngroupedAccountAutoTestService) run() {
	ticker := time.NewTicker(ungroupedAccountAutoTestTickInterval)
	defer ticker.Stop()

	s.runOnce()
	for {
		select {
		case <-ticker.C:
			s.runOnce()
		case <-s.stopCh:
			return
		}
	}
}

func (s *UngroupedAccountAutoTestService) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	accounts, err := s.listCandidates(ctx)
	if err != nil {
		logger.LegacyPrintf("service.ungrouped_account_auto_test", "[UngroupedAutoTest] list candidates failed: %v", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	workers := ungroupedAccountAutoTestConcurrency
	if len(accounts) < workers {
		workers = len(accounts)
	}
	if workers <= 0 {
		return
	}

	logger.LegacyPrintf("service.ungrouped_account_auto_test", "[UngroupedAutoTest] testing %d ungrouped accounts", len(accounts))

	jobs := make(chan Account)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for account := range jobs {
				s.runOne(ctx, account)
			}
		}()
	}

	for _, account := range accounts {
		select {
		case jobs <- account:
		case <-s.stopCh:
			close(jobs)
			wg.Wait()
			return
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		}
	}
	close(jobs)
	wg.Wait()
}

func (s *UngroupedAccountAutoTestService) runOne(ctx context.Context, account Account) {
	if account.ID <= 0 {
		return
	}
	startedAt := time.Now().UTC()
	result, err := s.accountTestSvc.RunTestBackground(ctx, account.ID, "")
	if err != nil {
		logger.LegacyPrintf("service.ungrouped_account_auto_test", "[UngroupedAutoTest] account=%d run failed: %v", account.ID, err)
		result = &ScheduledTestResult{
			Status:       "failed",
			ErrorMessage: err.Error(),
			StartedAt:    startedAt,
			FinishedAt:   time.Now().UTC(),
		}
	}
	updates := buildUngroupedAutoTestExtraUpdates(startedAt, result)
	if len(updates) == 0 {
		return
	}
	if updateErr := s.accountRepo.UpdateExtra(ctx, account.ID, updates); updateErr != nil {
		logger.LegacyPrintf("service.ungrouped_account_auto_test", "[UngroupedAutoTest] account=%d persist failed: %v", account.ID, updateErr)
	}
}

func (s *UngroupedAccountAutoTestService) listCandidates(ctx context.Context) ([]Account, error) {
	now := time.Now().UTC()
	candidates := make([]Account, 0, ungroupedAccountAutoTestMaxPerRun)
	for page := 1; len(candidates) < ungroupedAccountAutoTestMaxPerRun; page++ {
		params := pagination.PaginationParams{
			Page:      page,
			PageSize:  ungroupedAccountAutoTestPageSize,
			SortBy:    "updated_at",
			SortOrder: pagination.SortOrderAsc,
		}
		accounts, _, err := s.accountRepo.ListWithFilters(
			ctx,
			params,
			"",
			"",
			StatusActive,
			"",
			AccountListGroupUngrouped,
			"",
			"",
		)
		if err != nil {
			return nil, err
		}
		if len(accounts) == 0 {
			break
		}
		for _, account := range accounts {
			if !account.IsSchedulable() || account.ID <= 0 {
				continue
			}
			if !shouldRunUngroupedAutoTest(now, account.Extra) {
				continue
			}
			candidates = append(candidates, account)
			if len(candidates) >= ungroupedAccountAutoTestMaxPerRun {
				break
			}
		}
		if len(accounts) < ungroupedAccountAutoTestPageSize {
			break
		}
	}
	return candidates, nil
}

func shouldRunUngroupedAutoTest(now time.Time, extra map[string]any) bool {
	if extra == nil {
		return true
	}
	lastAt := parseExtraTimeValue(extra["auto_test_last_at"])
	if lastAt.IsZero() {
		return true
	}
	return now.Sub(lastAt) >= ungroupedAccountAutoTestRecencyInterval
}

func buildUngroupedAutoTestExtraUpdates(startedAt time.Time, result *ScheduledTestResult) map[string]any {
	updates := map[string]any{
		"auto_test_last_at": startedAt.Format(time.RFC3339),
	}
	if result == nil {
		updates["auto_test_last_status"] = "failed"
		updates["auto_test_last_error"] = "missing test result"
		return updates
	}

	finishedAt := result.FinishedAt
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	updates["auto_test_last_finished_at"] = finishedAt.Format(time.RFC3339)
	if result.Status != "" {
		updates["auto_test_last_status"] = result.Status
	}
	if result.ErrorMessage != "" {
		updates["auto_test_last_error"] = truncateAutoTestError(result.ErrorMessage)
	} else {
		updates["auto_test_last_error"] = ""
	}
	if result.HTTPStatusCode != nil {
		updates["auto_test_http_status"] = *result.HTTPStatusCode
	} else {
		updates["auto_test_http_status"] = 0
	}
	if result.LatencyMs > 0 {
		updates["auto_test_latency_ms"] = result.LatencyMs
	}
	return updates
}

func truncateAutoTestError(msg string) string {
	msg = strings.TrimSpace(msg)
	if len(msg) <= 500 {
		return msg
	}
	return strings.TrimSpace(msg[:500])
}

func parseExtraTimeValue(raw any) time.Time {
	switch v := raw.(type) {
	case string:
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
		if err == nil {
			return parsed.UTC()
		}
	case time.Time:
		return v.UTC()
	}
	return time.Time{}
}
