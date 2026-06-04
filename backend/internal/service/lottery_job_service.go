package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

const (
	lotteryJobDefaultSchedule = "*/30 * * * *"
	lotteryJobRunTimeout      = 5 * time.Minute
)

var lotteryJobCronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

type lotteryJobRunner interface {
	SyncLatestResult(ctx context.Context, lotteryType string) (*LotterySyncResult, error)
	SettleOpenedIssues(ctx context.Context, lotteryType string, limit int) (*LotteryOpenedSettlementResult, error)
}

// LotteryJobService synchronizes latest draw results and settles opened issues.
type LotteryJobService struct {
	runner   lotteryJobRunner
	cfg      *config.Config
	schedule string

	mu        sync.Mutex
	cron      *cron.Cron
	running   bool
	startOnce sync.Once
	stopOnce  sync.Once
}

func NewLotteryJobService(lotteryService *LotteryService, cfg *config.Config) *LotteryJobService {
	return newLotteryJobService(lotteryService, cfg, lotteryJobDefaultSchedule)
}

func newLotteryJobService(runner lotteryJobRunner, cfg *config.Config, schedule string) *LotteryJobService {
	return &LotteryJobService{
		runner:   runner,
		cfg:      cfg,
		schedule: schedule,
	}
}

func (s *LotteryJobService) Start() {
	if s == nil || s.runner == nil {
		return
	}
	s.startOnce.Do(func() {
		loc := time.Local
		if s.cfg != nil {
			if parsed, err := time.LoadLocation(s.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}
		schedule := s.schedule
		if schedule == "" {
			schedule = lotteryJobDefaultSchedule
		}
		c := cron.New(cron.WithParser(lotteryJobCronParser), cron.WithLocation(loc))
		if _, err := c.AddFunc(schedule, func() { s.RunOnce(context.Background()) }); err != nil {
			logger.LegacyPrintf("service.lottery_job", "[LotteryJob] not started (invalid schedule=%q): %v", schedule, err)
			return
		}
		s.cron = c
		s.cron.Start()
		logger.LegacyPrintf("service.lottery_job", "[LotteryJob] started (schedule=%q tz=%s)", schedule, loc.String())
		go s.RunOnce(context.Background())
	})
}

func (s *LotteryJobService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		if s.cron == nil {
			return
		}
		ctx := s.cron.Stop()
		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
			logger.LegacyPrintf("service.lottery_job", "[LotteryJob] cron stop timed out")
		}
	})
}

func (s *LotteryJobService) RunOnce(ctx context.Context) (*LotteryJobRunResult, error) {
	if s == nil || s.runner == nil {
		return nil, ErrLotteryProviderNotFound
	}
	if !s.tryMarkRunning() {
		return &LotteryJobRunResult{LotteryType: LotteryTypeSSQ}, nil
	}
	defer s.clearRunning()

	runCtx, cancel := context.WithTimeout(ctx, lotteryJobRunTimeout)
	defer cancel()

	result := &LotteryJobRunResult{LotteryType: LotteryTypeSSQ}
	var joined error
	synced, err := s.runner.SyncLatestResult(runCtx, LotteryTypeSSQ)
	if err != nil {
		if errors.Is(err, ErrLotteryProviderUnavailable) {
			logger.LegacyPrintf("service.lottery_job", "[LotteryJob] warning: sync latest result skipped because provider is temporarily unavailable")
		} else {
			joined = errors.Join(joined, fmt.Errorf("sync latest lottery result: %w", err))
			logger.LegacyPrintf("service.lottery_job", "[LotteryJob] sync latest result failed: %v", err)
		}
	} else if synced != nil {
		result.SyncedIssueNo = synced.IssueNo
		result.SyncReplayed = synced.Replayed
		logger.LegacyPrintf("service.lottery_job", "[LotteryJob] synced issue=%s replayed=%t", synced.IssueNo, synced.Replayed)
	}

	settlement, settleErr := s.runner.SettleOpenedIssues(runCtx, LotteryTypeSSQ, lotteryOpenedSettlementDefaultLimit)
	result.SettlementSummary = settlement
	if settleErr != nil {
		joined = errors.Join(joined, fmt.Errorf("settle opened lottery issues: %w", settleErr))
		logger.LegacyPrintf("service.lottery_job", "[LotteryJob] settle opened issues failed: %v", settleErr)
	} else if settlement != nil && len(settlement.SettledIssues) > 0 {
		logger.LegacyPrintf("service.lottery_job", "[LotteryJob] settled issues=%d", len(settlement.SettledIssues))
	}
	return result, joined
}

func (s *LotteryJobService) tryMarkRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *LotteryJobService) clearRunning() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}
