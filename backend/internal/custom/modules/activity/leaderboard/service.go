// Package leaderboard prepares activity-owned leaderboard composition.
package leaderboard

import (
	"context"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

var (
	ErrUnavailable     = errors.New("activity leaderboard is unavailable")
	ErrDisabled        = errors.New("activity leaderboard is disabled")
	ErrUnsupportedKind = errors.New("unsupported leaderboard kind")
	ErrInvalidPeriod   = errors.New("invalid leaderboard period")
	ErrInvalidPage     = errors.New("invalid leaderboard pagination")
)

// Readers contains only read-side public contracts. The future composition root
// may adapt legacy services or module public APIs, but this package must never
// import a wallet implementation.
type Readers struct {
	Balance     contract.BalanceLeaderboardReader
	Consumption contract.ConsumptionLeaderboardReader
	Checkin     contract.CheckinLeaderboardReader
}

// Service enforces activity feature policy before delegating to a read model.
type Service struct {
	settings contract.LeaderboardSettingsReader
	readers  Readers
}

func NewService(settings contract.LeaderboardSettingsReader, readers Readers) *Service {
	return &Service{settings: settings, readers: readers}
}

// List returns a public leaderboard result after applying the effective
// settings. IncludeAdmin is always supplied by the trusted settings reader.
func (s *Service) List(ctx context.Context, kind contract.LeaderboardKind, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	if s == nil || s.settings == nil {
		return contract.LeaderboardPage{}, ErrUnavailable
	}
	if err := validateQuery(kind, &query); err != nil {
		return contract.LeaderboardPage{}, err
	}

	settings, err := s.settings.GetActivityLeaderboardSettings(ctx)
	if err != nil {
		return contract.LeaderboardPage{}, fmt.Errorf("read activity leaderboard settings: %w", err)
	}
	if !enabled(settings, kind) {
		return contract.LeaderboardPage{}, ErrDisabled
	}
	query.IncludeAdmin = settings.IncludeAdmin

	switch kind {
	case contract.LeaderboardBalance:
		if s.readers.Balance == nil {
			return contract.LeaderboardPage{}, ErrUnavailable
		}
		return s.readers.Balance.ListBalanceLeaderboard(ctx, query)
	case contract.LeaderboardConsumption:
		if s.readers.Consumption == nil {
			return contract.LeaderboardPage{}, ErrUnavailable
		}
		return s.readers.Consumption.ListConsumptionLeaderboard(ctx, query)
	case contract.LeaderboardCheckin:
		if s.readers.Checkin == nil {
			return contract.LeaderboardPage{}, ErrUnavailable
		}
		return s.readers.Checkin.ListCheckinLeaderboard(ctx, query)
	default:
		return contract.LeaderboardPage{}, ErrUnsupportedKind
	}
}

func enabled(settings contract.LeaderboardFeatureSettings, kind contract.LeaderboardKind) bool {
	if !settings.Enabled {
		return false
	}
	switch kind {
	case contract.LeaderboardBalance:
		return settings.BalanceEnabled
	case contract.LeaderboardConsumption:
		return settings.ConsumptionEnabled
	case contract.LeaderboardCheckin:
		return settings.CheckinEnabled
	default:
		return false
	}
}

func validateQuery(kind contract.LeaderboardKind, query *contract.LeaderboardQuery) error {
	if query.Page < 1 || query.PageSize < 1 {
		return ErrInvalidPage
	}

	switch kind {
	case contract.LeaderboardBalance, contract.LeaderboardCheckin:
		if query.Period != "" {
			return ErrInvalidPeriod
		}
	case contract.LeaderboardConsumption:
		if query.Period == "" {
			query.Period = contract.LeaderboardPeriodDaily
		}
		if query.Period != contract.LeaderboardPeriodDaily &&
			query.Period != contract.LeaderboardPeriodWeekly &&
			query.Period != contract.LeaderboardPeriodMonthly {
			return ErrInvalidPeriod
		}
	default:
		return ErrUnsupportedKind
	}

	return nil
}
