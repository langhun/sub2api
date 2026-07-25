package checkin

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

var (
	ErrCheckinDisabled     = infraerrors.Forbidden("CHECKIN_DISABLED", "check-in feature is not enabled")
	ErrCheckinLuckDisabled = infraerrors.Forbidden("CHECKIN_LUCK_DISABLED", "luck check-in feature is not enabled")
	ErrAlreadyCheckedIn    = infraerrors.Conflict("ALREADY_CHECKED_IN", "you have already checked in today")
	ErrCheckinNotAllowed   = infraerrors.Forbidden("CHECKIN_NOT_ALLOWED", "check-in is not allowed for your account")
	ErrInvalidBetAmount    = infraerrors.BadRequest("INVALID_BET_AMOUNT", "bet amount must be greater than 0 and not exceed your balance")
)

// Service owns the activity check-in transaction. It deliberately depends only
// on narrow Activity and platform ports, never on the shared CheckinService or
// BlindBoxService implementations.
type Service struct {
	repository   Repository
	transactions contract.TransactionRunner
	accounts     contract.AccountReader
	balance      contract.BalanceWriter
	ledger       CheckinLedger
	settings     contract.SettingsReader
	cache        contract.BalanceCacheInvalidator
	blindbox     BlindboxDelivery
	clock        Clock
	random       RandomSource
}

var _ contract.CheckinService = (*Service)(nil)

func NewService(deps Dependencies) (*Service, error) {
	if err := deps.validate(); err != nil {
		return nil, err
	}
	clock := deps.Clock
	if clock == nil {
		clock = systemClock{}
	}
	random := deps.Random
	if random == nil {
		random = cryptoRandomSource{}
	}
	return &Service{
		repository:   deps.Repository,
		transactions: deps.Transactions,
		accounts:     deps.Accounts,
		balance:      deps.Balance,
		ledger:       deps.Ledger,
		settings:     deps.Settings,
		cache:        deps.Cache,
		blindbox:     deps.Blindbox,
		clock:        clock,
		random:       random,
	}, nil
}

func (s *Service) Checkin(ctx context.Context, userID int64) (*contract.CheckinResult, error) {
	settings, err := s.activitySettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Checkin.Enabled {
		return nil, ErrCheckinDisabled
	}
	if _, err := s.activeAccount(ctx, userID); err != nil {
		return nil, err
	}

	today := s.clock.Today()
	todayText := today.Format("2006-01-02")
	if existing, err := s.repository.FindToday(ctx, userID, today); err != nil {
		return nil, fmt.Errorf("query checkin: %w", err)
	} else if existing != nil {
		return resultFromRecord(existing, todayText), nil
	}

	fraction, err := s.random.Float64()
	if err != nil || !validRandomFraction(fraction) {
		if err != nil {
			return nil, fmt.Errorf("generate checkin reward: %w", err)
		}
		return nil, fmt.Errorf("generate checkin reward: invalid random value")
	}
	reward := roundToCents(settings.Checkin.MinimumReward + fraction*(settings.Checkin.MaximumReward-settings.Checkin.MinimumReward))
	streak := s.calculateStreak(ctx, userID, today)

	var prepared *PreparedBlindbox
	var record *Record
	created := false
	if err := s.transactions.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repository.LockAccount(txCtx, userID); err != nil {
			return fmt.Errorf("lock checkin user: %w", err)
		}
		if existing, err := s.repository.FindToday(txCtx, userID, today); err != nil {
			return err
		} else if existing != nil {
			record = existing
			return nil
		}

		record = &Record{
			UserID: userID, CheckinDate: today, RewardAmount: reward, StreakDays: streak, CheckinType: checkinTypeNormal,
		}
		if err := s.repository.Create(txCtx, record); err != nil {
			return fmt.Errorf("create checkin record: %w", err)
		}
		created = true
		key := checkinIdempotencyKey(userID, today)
		if err := s.balance.Credit(txCtx, contract.BalanceOperation{
			UserID: userID, Amount: reward, Reason: adjustmentTypeCheckin, IdempotencyKey: key,
		}); err != nil {
			return fmt.Errorf("update user balance: %w", err)
		}
		if err := s.ledger.RecordCheckinAdjustment(txCtx, CheckinAuditEntry{
			UserID: userID, Type: adjustmentTypeCheckin, Amount: reward, IdempotencyKey: key, OccurredAt: s.clock.Now(),
		}); err != nil {
			return fmt.Errorf("create checkin audit record: %w", err)
		}
		if s.blindbox != nil {
			preparedDelivery, prepareErr := s.blindbox.PrepareForCheckin(txCtx, userID, record.ID, streak)
			if prepareErr != nil {
				return fmt.Errorf("prepare checkin blindbox: %w", prepareErr)
			}
			prepared = preparedDelivery
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if record == nil {
		return nil, fmt.Errorf("checkin transaction completed without a record")
	}
	// A concurrent caller can commit first after the optimistic read. Preserve
	// the established idempotent response and do not invalidate caches again.
	if !created {
		return resultFromRecord(record, todayText), nil
	}
	s.invalidateCache(ctx, userID)
	result := &contract.CheckinResult{RewardAmount: reward, StreakDays: streak, CheckedAt: todayText, CheckinType: checkinTypeNormal}
	s.completeBlindbox(ctx, prepared, result, "immediate check-in blindbox delivery failed")
	return result, nil
}

func (s *Service) LuckCheckin(ctx context.Context, userID int64, betAmount float64, useMaxBalance bool) (*contract.CheckinResult, error) {
	settings, err := s.activitySettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Checkin.LuckEnabled {
		return nil, ErrCheckinLuckDisabled
	}
	if _, err := s.activeAccount(ctx, userID); err != nil {
		return nil, err
	}

	today := s.clock.Today()
	todayText := today.Format("2006-01-02")
	if existing, err := s.repository.FindToday(ctx, userID, today); err != nil {
		return nil, fmt.Errorf("query checkin: %w", err)
	} else if existing != nil {
		return resultFromRecord(existing, todayText), nil
	}

	fraction, err := s.random.Float64()
	if err != nil || !validRandomFraction(fraction) {
		if err != nil {
			return nil, fmt.Errorf("generate luck checkin multiplier: %w", err)
		}
		return nil, fmt.Errorf("generate luck checkin multiplier: invalid random value")
	}
	multiplier := roundToCents(settings.Checkin.MinimumMultiplier + fraction*(settings.Checkin.MaximumMultiplier-settings.Checkin.MinimumMultiplier))
	streak := s.calculateStreak(ctx, userID, today)

	var prepared *PreparedBlindbox
	var record *Record
	var resolvedBet float64
	created := false
	if err := s.transactions.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := s.repository.LockAccount(txCtx, userID); err != nil {
			return fmt.Errorf("lock checkin user: %w", err)
		}
		if existing, err := s.repository.FindToday(txCtx, userID, today); err != nil {
			return err
		} else if existing != nil {
			record = existing
			return nil
		}
		lockedAccount, err := s.repository.GetLockedAccount(txCtx, userID)
		if err != nil {
			return fmt.Errorf("get locked user: %w", err)
		}
		if lockedAccount.Status != accountStatusActive {
			return ErrCheckinNotAllowed
		}
		var ok bool
		resolvedBet, ok = resolveLuckCheckinBetAmount(betAmount, lockedAccount.Balance, useMaxBalance)
		if !ok {
			return ErrInvalidBetAmount
		}
		reward := roundToCents(resolvedBet * (multiplier - 1))
		record = &Record{
			UserID: userID, CheckinDate: today, RewardAmount: reward, StreakDays: streak, CheckinType: checkinTypeLuck,
			BetAmount: resolvedBet, Multiplier: multiplier,
		}
		if err := s.repository.Create(txCtx, record); err != nil {
			return fmt.Errorf("create checkin record: %w", err)
		}
		created = true
		key := checkinIdempotencyKey(userID, today)
		if reward > 0 {
			if err := s.balance.Credit(txCtx, contract.BalanceOperation{
				UserID: userID, Amount: reward, Reason: adjustmentTypeCheckinLuck, IdempotencyKey: key,
			}); err != nil {
				return fmt.Errorf("update user balance: %w", err)
			}
		} else if reward < 0 {
			updated, err := s.balance.DebitIfSufficient(txCtx, contract.BalanceOperation{
				UserID: userID, Amount: -reward, Reason: adjustmentTypeCheckinLuck, IdempotencyKey: key,
			})
			if err != nil {
				return fmt.Errorf("update user balance: %w", err)
			}
			if !updated {
				return ErrInvalidBetAmount
			}
		}
		if err := s.ledger.RecordCheckinAdjustment(txCtx, CheckinAuditEntry{
			UserID: userID, Type: adjustmentTypeCheckinLuck, Amount: reward, Multiplier: multiplier, BetAmount: resolvedBet,
			IdempotencyKey: key, OccurredAt: s.clock.Now(),
		}); err != nil {
			return fmt.Errorf("create luck checkin audit record: %w", err)
		}
		if s.blindbox != nil {
			preparedDelivery, prepareErr := s.blindbox.PrepareForCheckin(txCtx, userID, record.ID, streak)
			if prepareErr != nil {
				return fmt.Errorf("prepare luck checkin blindbox: %w", prepareErr)
			}
			prepared = preparedDelivery
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if record == nil {
		return nil, fmt.Errorf("luck checkin transaction completed without a record")
	}
	if !created {
		return resultFromRecord(record, todayText), nil
	}
	s.invalidateCache(ctx, userID)
	result := &contract.CheckinResult{
		RewardAmount: record.RewardAmount, StreakDays: record.StreakDays, CheckedAt: todayText, CheckinType: checkinTypeLuck,
		BetAmount: resolvedBet, Multiplier: multiplier,
	}
	s.completeBlindbox(ctx, prepared, result, "immediate luck check-in blindbox delivery failed")
	return result, nil
}

func (s *Service) GetStatus(ctx context.Context, userID int64) (*contract.CheckinStatus, error) {
	settings, err := s.activitySettings(ctx)
	if err != nil {
		return nil, err
	}
	account, err := s.accounts.GetAccount(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	anyEnabled := settings.Checkin.Enabled || settings.Checkin.LuckEnabled
	status := &contract.CheckinStatus{
		Enabled: settings.Checkin.Enabled, LuckEnabled: settings.Checkin.LuckEnabled,
		BlindboxEnabled: settings.Blindbox.Enabled, CanCheckin: anyEnabled,
		MinReward: settings.Checkin.MinimumReward, MaxReward: settings.Checkin.MaximumReward,
		MinMultiplier: settings.Checkin.MinimumMultiplier, MaxMultiplier: settings.Checkin.MaximumMultiplier,
		Balance: account.Balance,
	}
	if !anyEnabled {
		return status, nil
	}
	status.BlindboxTriggerType = settings.Blindbox.TriggerType
	status.BlindboxInterval = settings.Blindbox.Interval
	today := s.clock.Today()
	record, err := s.repository.FindToday(ctx, userID, today)
	if err != nil {
		return nil, fmt.Errorf("query today checkin: %w", err)
	}
	if record == nil {
		status.StreakDays = s.calculateStreak(ctx, userID, today)
		return status, nil
	}
	status.CanCheckin = false
	status.StreakDays = record.StreakDays
	reward := record.RewardAmount
	status.TodayReward = &reward
	status.TodayCheckinType = record.CheckinType
	if record.CheckinType == checkinTypeLuck {
		multiplier := record.Multiplier
		status.TodayMultiplier = &multiplier
	}
	return status, nil
}

func (s *Service) GetCalendar(ctx context.Context, userID int64) (*contract.CheckinCalendar, error) {
	today := s.clock.Today()
	start := today.AddDate(0, 0, -29)
	records, err := s.repository.ListCalendar(ctx, userID, start, today)
	if err != nil {
		return nil, fmt.Errorf("query calendar: %w", err)
	}
	byDate := make(map[string]Record, len(records))
	for _, record := range records {
		byDate[record.CheckinDate.Format("2006-01-02")] = record
	}
	days := make([]contract.CheckinCalendarDay, 30)
	for index := range days {
		date := start.AddDate(0, 0, index)
		key := date.Format("2006-01-02")
		days[index] = contract.CheckinCalendarDay{Date: key}
		if record, ok := byDate[key]; ok {
			days[index].CheckedIn = true
			days[index].RewardType = record.CheckinType
			days[index].RewardValue = record.RewardAmount
			days[index].StreakDays = record.StreakDays
		}
	}
	return &contract.CheckinCalendar{Days: days}, nil
}

func (s *Service) activitySettings(ctx context.Context) (contract.Settings, error) {
	settings, err := s.settings.GetActivitySettings(ctx)
	if err != nil {
		return contract.Settings{}, fmt.Errorf("read activity settings: %w", err)
	}
	return settings, nil
}

func (s *Service) activeAccount(ctx context.Context, userID int64) (contract.Account, error) {
	account, err := s.accounts.GetAccount(ctx, userID)
	if err != nil {
		return contract.Account{}, fmt.Errorf("get user: %w", err)
	}
	if account.Status != accountStatusActive {
		return contract.Account{}, ErrCheckinNotAllowed
	}
	return account, nil
}

func (s *Service) calculateStreak(ctx context.Context, userID int64, today time.Time) int {
	previous, err := s.repository.FindPrevious(ctx, userID, today)
	if err != nil || previous == nil {
		return 1
	}
	yesterday := today.AddDate(0, 0, -1)
	if sameDate(previous.CheckinDate, yesterday) {
		return previous.StreakDays + 1
	}
	return 1
}

func (s *Service) invalidateCache(ctx context.Context, userID int64) {
	if s.cache == nil {
		return
	}
	if err := s.cache.InvalidateBalance(ctx, userID); err != nil {
		slog.Warn("invalidate check-in balance cache failed", "user_id", userID, "error", err)
	}
}

func (s *Service) completeBlindbox(ctx context.Context, prepared *PreparedBlindbox, result *contract.CheckinResult, message string) {
	if prepared == nil || result == nil {
		return
	}
	copyResult := prepared.Result
	result.Blindbox = &copyResult
	if s.blindbox == nil || prepared.DeliveryID <= 0 {
		return
	}
	delivered, err := s.blindbox.Deliver(ctx, prepared.DeliveryID)
	if err != nil {
		slog.Warn(message, "delivery_id", prepared.DeliveryID, "error", err)
		return
	}
	if delivered != nil {
		result.Blindbox.RewardDetail = delivered.RewardDetail
	}
}

func resultFromRecord(record *Record, checkedAt string) *contract.CheckinResult {
	if record == nil {
		return nil
	}
	return &contract.CheckinResult{
		RewardAmount: record.RewardAmount, StreakDays: record.StreakDays, CheckedAt: checkedAt,
		CheckinType: record.CheckinType, BetAmount: record.BetAmount, Multiplier: record.Multiplier,
	}
}

func resolveLuckCheckinBetAmount(betAmount, balance float64, useMaxBalance bool) (float64, bool) {
	if math.IsNaN(betAmount) || math.IsInf(betAmount, 0) || math.IsNaN(balance) || math.IsInf(balance, 0) || betAmount <= 0 || balance <= 0 {
		return 0, false
	}
	if useMaxBalance {
		return math.Min(betAmount, balance), true
	}
	if betAmount <= balance {
		return betAmount, true
	}
	return 0, false
}

func sameDate(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func validRandomFraction(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value < 1
}

func roundToCents(value float64) float64 {
	return math.Round(value*100) / 100
}

func checkinIdempotencyKey(userID int64, today time.Time) string {
	return fmt.Sprintf("checkin:%d:%s", userID, today.Format("2006-01-02"))
}

type systemClock struct{}

func (systemClock) Today() time.Time { return timezone.Today() }
func (systemClock) Now() time.Time   { return time.Now() }

type cryptoRandomSource struct{}

func (cryptoRandomSource) Float64() (float64, error) {
	const precision = int64(1) << 53
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(precision))
	if err != nil {
		return 0, err
	}
	return float64(value.Int64()) / float64(precision), nil
}
