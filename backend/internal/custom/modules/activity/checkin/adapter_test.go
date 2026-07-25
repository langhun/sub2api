package checkin

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	legacy "github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type legacyCheckinServiceStub struct {
	checkin  *legacy.CheckinResult
	status   *legacy.CheckinStatus
	calendar *legacy.CheckinCalendar
	luckBet  float64
	luckMax  bool
}

func (s *legacyCheckinServiceStub) Checkin(context.Context, int64) (*legacy.CheckinResult, error) {
	return s.checkin, nil
}

func (s *legacyCheckinServiceStub) LuckCheckin(_ context.Context, _ int64, betAmount float64, useMaxBalance bool) (*legacy.CheckinResult, error) {
	s.luckBet = betAmount
	s.luckMax = useMaxBalance
	return s.checkin, nil
}

func (s *legacyCheckinServiceStub) GetStatus(context.Context, int64) (*legacy.CheckinStatus, error) {
	return s.status, nil
}

func (s *legacyCheckinServiceStub) GetCalendar(context.Context, int64) (*legacy.CheckinCalendar, error) {
	return s.calendar, nil
}

type legacyBlindboxRecordsServiceStub struct {
	records *legacy.BlindboxRecordList
}

func (s *legacyBlindboxRecordsServiceStub) GetUserRecords(context.Context, int64, int, int) (*legacy.BlindboxRecordList, error) {
	return s.records, nil
}

func TestLegacyAdapterMapsActivityContract(t *testing.T) {
	todayReward := 3.2
	todayMultiplier := 1.4
	stub := &legacyCheckinServiceStub{
		checkin: &legacy.CheckinResult{
			RewardAmount: 2.5,
			StreakDays:   7,
			CheckedAt:    "2026-07-25",
			CheckinType:  legacy.CheckinTypeLuck,
			BetAmount:    5,
			Multiplier:   1.5,
			Blindbox: &legacy.BlindboxResult{
				PrizeName: "Premium",
				Rarity:    legacy.RarityEpic,
			},
		},
		status: &legacy.CheckinStatus{
			Enabled:         true,
			LuckEnabled:     true,
			CanCheckin:      false,
			TodayReward:     &todayReward,
			TodayMultiplier: &todayMultiplier,
		},
		calendar: &legacy.CheckinCalendar{Days: []legacy.CheckinCalendarDay{{Date: "2026-07-25", CheckedIn: true, StreakDays: 7}}},
	}

	adapter := NewLegacyAdapter(stub)
	result, err := adapter.LuckCheckin(context.Background(), 42, 5, true)
	require.NoError(t, err)
	require.Equal(t, 5.0, stub.luckBet)
	require.True(t, stub.luckMax)
	require.Equal(t, 2.5, result.RewardAmount)
	require.Equal(t, "Premium", result.Blindbox.PrizeName)

	status, err := adapter.GetStatus(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, todayReward, *status.TodayReward)
	require.Equal(t, todayMultiplier, *status.TodayMultiplier)

	calendar, err := adapter.GetCalendar(context.Background(), 42)
	require.NoError(t, err)
	require.Equal(t, []contract.CheckinCalendarDay{{Date: "2026-07-25", CheckedIn: true, StreakDays: 7}}, calendar.Days)
}

func TestLegacyBlindboxRecordsAdapterMapsActivityContract(t *testing.T) {
	adapter := NewLegacyBlindboxRecordsAdapter(&legacyBlindboxRecordsServiceStub{
		records: &legacy.BlindboxRecordList{
			Items: []legacy.BlindboxRecord{{ID: 9, PrizeName: "Bonus", RewardValue: 2.5}},
			Total: 1,
		},
	})

	result, err := adapter.GetUserRecords(context.Background(), 42, 1, 20)
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Equal(t, int64(9), result.Items[0].ID)
	require.Equal(t, "Bonus", result.Items[0].PrizeName)
}

func TestLegacyAdapterRejectsMissingService(t *testing.T) {
	result, err := NewLegacyAdapter(nil).Checkin(context.Background(), 42)
	require.Nil(t, result)
	require.ErrorIs(t, err, ErrUnavailable)
}
