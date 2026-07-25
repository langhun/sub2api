// Package checkin owns the activity module's check-in HTTP and service boundary.
package checkin

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	legacy "github.com/Wei-Shaw/sub2api/internal/service"
)

var ErrUnavailable = infraerrors.InternalServer("ACTIVITY_CHECKIN_UNAVAILABLE", "activity check-in is unavailable")

// legacyCheckinService confines the legacy service dependency to this adapter.
// The module handler and callers only depend on contract.CheckinService.
type legacyCheckinService interface {
	Checkin(ctx context.Context, userID int64) (*legacy.CheckinResult, error)
	LuckCheckin(ctx context.Context, userID int64, betAmount float64, useMaxBalance bool) (*legacy.CheckinResult, error)
	GetStatus(ctx context.Context, userID int64) (*legacy.CheckinStatus, error)
	GetCalendar(ctx context.Context, userID int64) (*legacy.CheckinCalendar, error)
}

// LegacyAdapter preserves the existing atomic check-in implementation while
// exposing only activity-owned contract types. It is a temporary composition
// adapter: the core transaction, balance, audit, and blind-box delivery logic
// cannot be safely duplicated in the Overlay module.
type LegacyAdapter struct {
	service legacyCheckinService
}

var _ contract.CheckinService = (*LegacyAdapter)(nil)

// NewLegacyAdapter adapts the current core CheckinService without changing its
// route, Wire, repository, or transaction ownership.
func NewLegacyAdapter(service legacyCheckinService) *LegacyAdapter {
	return &LegacyAdapter{service: service}
}

func (a *LegacyAdapter) Checkin(ctx context.Context, userID int64) (*contract.CheckinResult, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	result, err := a.service.Checkin(ctx, userID)
	return checkinResultFromLegacy(result), err
}

func (a *LegacyAdapter) LuckCheckin(ctx context.Context, userID int64, betAmount float64, useMaxBalance bool) (*contract.CheckinResult, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	result, err := a.service.LuckCheckin(ctx, userID, betAmount, useMaxBalance)
	return checkinResultFromLegacy(result), err
}

func (a *LegacyAdapter) GetStatus(ctx context.Context, userID int64) (*contract.CheckinStatus, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	status, err := a.service.GetStatus(ctx, userID)
	return checkinStatusFromLegacy(status), err
}

func (a *LegacyAdapter) GetCalendar(ctx context.Context, userID int64) (*contract.CheckinCalendar, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	calendar, err := a.service.GetCalendar(ctx, userID)
	return checkinCalendarFromLegacy(calendar), err
}

type legacyBlindboxRecordsService interface {
	GetUserRecords(ctx context.Context, userID int64, page, pageSize int) (*legacy.BlindboxRecordList, error)
}

// LegacyBlindboxRecordsAdapter is the check-in route's read-only bridge to the
// existing blind-box service until that activity submodule has its own store.
type LegacyBlindboxRecordsAdapter struct {
	service legacyBlindboxRecordsService
}

var _ contract.BlindboxRecordsReader = (*LegacyBlindboxRecordsAdapter)(nil)

func NewLegacyBlindboxRecordsAdapter(service legacyBlindboxRecordsService) *LegacyBlindboxRecordsAdapter {
	return &LegacyBlindboxRecordsAdapter{service: service}
}

func (a *LegacyBlindboxRecordsAdapter) GetUserRecords(ctx context.Context, userID int64, page, pageSize int) (*contract.BlindboxRecordList, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	result, err := a.service.GetUserRecords(ctx, userID, page, pageSize)
	return blindboxRecordListFromLegacy(result), err
}

func checkinResultFromLegacy(result *legacy.CheckinResult) *contract.CheckinResult {
	if result == nil {
		return nil
	}
	mapped := &contract.CheckinResult{
		RewardAmount: result.RewardAmount,
		StreakDays:   result.StreakDays,
		CheckedAt:    result.CheckedAt,
		CheckinType:  result.CheckinType,
		BetAmount:    result.BetAmount,
		Multiplier:   result.Multiplier,
	}
	if result.Blindbox != nil {
		mapped.Blindbox = &contract.BlindboxResult{
			PrizeName:        result.Blindbox.PrizeName,
			Rarity:           result.Blindbox.Rarity,
			RewardType:       result.Blindbox.RewardType,
			RewardValue:      result.Blindbox.RewardValue,
			SubscriptionDays: result.Blindbox.SubscriptionDays,
			RewardDetail:     result.Blindbox.RewardDetail,
		}
	}
	return mapped
}

func checkinStatusFromLegacy(status *legacy.CheckinStatus) *contract.CheckinStatus {
	if status == nil {
		return nil
	}
	return &contract.CheckinStatus{
		Enabled:             status.Enabled,
		LuckEnabled:         status.LuckEnabled,
		BlindboxEnabled:     status.BlindboxEnabled,
		BlindboxTriggerType: status.BlindboxTriggerType,
		BlindboxInterval:    status.BlindboxInterval,
		CanCheckin:          status.CanCheckin,
		StreakDays:          status.StreakDays,
		TodayReward:         status.TodayReward,
		TodayCheckinType:    status.TodayCheckinType,
		TodayMultiplier:     status.TodayMultiplier,
		MinReward:           status.MinReward,
		MaxReward:           status.MaxReward,
		MinMultiplier:       status.MinMultiplier,
		MaxMultiplier:       status.MaxMultiplier,
		Balance:             status.Balance,
	}
}

func checkinCalendarFromLegacy(calendar *legacy.CheckinCalendar) *contract.CheckinCalendar {
	if calendar == nil {
		return nil
	}
	days := make([]contract.CheckinCalendarDay, len(calendar.Days))
	for i, day := range calendar.Days {
		days[i] = contract.CheckinCalendarDay{
			Date:        day.Date,
			CheckedIn:   day.CheckedIn,
			RewardType:  day.RewardType,
			RewardValue: day.RewardValue,
			StreakDays:  day.StreakDays,
		}
	}
	return &contract.CheckinCalendar{Days: days}
}

func blindboxRecordListFromLegacy(records *legacy.BlindboxRecordList) *contract.BlindboxRecordList {
	if records == nil {
		return nil
	}
	items := make([]contract.BlindboxRecord, len(records.Items))
	for i, record := range records.Items {
		items[i] = contract.BlindboxRecord{
			ID:               record.ID,
			PrizeName:        record.PrizeName,
			Rarity:           record.Rarity,
			RewardType:       record.RewardType,
			RewardValue:      record.RewardValue,
			RewardDetail:     record.RewardDetail,
			SubscriptionDays: record.SubscriptionDays,
			StreakDays:       record.StreakDays,
			CreatedAt:        record.CreatedAt,
		}
	}
	return &contract.BlindboxRecordList{Items: items, Total: records.Total}
}
