package leaderboard

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/stretchr/testify/require"
)

type leaderboardSettingsStub struct {
	settings contract.LeaderboardFeatureSettings
	err      error
}

func (s leaderboardSettingsStub) GetActivityLeaderboardSettings(context.Context) (contract.LeaderboardFeatureSettings, error) {
	return s.settings, s.err
}

type balanceReaderStub struct {
	queries []contract.LeaderboardQuery
}

func (r *balanceReaderStub) ListBalanceLeaderboard(_ context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	r.queries = append(r.queries, query)
	return contract.LeaderboardPage{Entries: []contract.LeaderboardEntry{{Rank: 1, Username: "alice", Value: 12.5}}, Total: 1}, nil
}

type consumptionReaderStub struct {
	queries []contract.LeaderboardQuery
}

func (r *consumptionReaderStub) ListConsumptionLeaderboard(_ context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	r.queries = append(r.queries, query)
	return contract.LeaderboardPage{}, nil
}

type checkinReaderStub struct {
	queries []contract.LeaderboardQuery
}

func (r *checkinReaderStub) ListCheckinLeaderboard(_ context.Context, query contract.LeaderboardQuery) (contract.LeaderboardPage, error) {
	r.queries = append(r.queries, query)
	return contract.LeaderboardPage{}, nil
}

func TestServiceListAppliesTrustedSettingsToBalanceReader(t *testing.T) {
	reader := &balanceReaderStub{}
	service := NewService(leaderboardSettingsStub{settings: contract.LeaderboardFeatureSettings{
		Enabled: true, BalanceEnabled: true, IncludeAdmin: true,
	}}, Readers{Balance: reader})

	result, err := service.List(context.Background(), contract.LeaderboardBalance, contract.LeaderboardQuery{
		Page: 1, PageSize: 20, IncludeAdmin: false,
	})

	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, reader.queries, 1)
	require.True(t, reader.queries[0].IncludeAdmin)
	require.Empty(t, reader.queries[0].Period)
}

func TestServiceListDefaultsTimeBasedBoardToDaily(t *testing.T) {
	reader := &consumptionReaderStub{}
	service := NewService(leaderboardSettingsStub{settings: contract.LeaderboardFeatureSettings{
		Enabled: true, ConsumptionEnabled: true,
	}}, Readers{Consumption: reader})

	_, err := service.List(context.Background(), contract.LeaderboardConsumption, contract.LeaderboardQuery{Page: 2, PageSize: 10})

	require.NoError(t, err)
	require.Equal(t, []contract.LeaderboardQuery{{Page: 2, PageSize: 10, Period: contract.LeaderboardPeriodDaily}}, reader.queries)
}

func TestServiceListRejectsDisabledBoardsWithoutReading(t *testing.T) {
	tests := []struct {
		name     string
		kind     contract.LeaderboardKind
		settings contract.LeaderboardFeatureSettings
		readers  Readers
	}{
		{
			name: "global", kind: contract.LeaderboardBalance,
			settings: contract.LeaderboardFeatureSettings{Enabled: false, BalanceEnabled: true},
			readers:  Readers{Balance: &balanceReaderStub{}},
		},
		{
			name: "balance", kind: contract.LeaderboardBalance,
			settings: contract.LeaderboardFeatureSettings{Enabled: true},
			readers:  Readers{Balance: &balanceReaderStub{}},
		},
		{
			name: "consumption", kind: contract.LeaderboardConsumption,
			settings: contract.LeaderboardFeatureSettings{Enabled: true},
			readers:  Readers{Consumption: &consumptionReaderStub{}},
		},
		{
			name: "checkin", kind: contract.LeaderboardCheckin,
			settings: contract.LeaderboardFeatureSettings{Enabled: true},
			readers:  Readers{Checkin: &checkinReaderStub{}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := NewService(leaderboardSettingsStub{settings: test.settings}, test.readers)

			_, err := service.List(context.Background(), test.kind, contract.LeaderboardQuery{Page: 1, PageSize: 20})

			require.ErrorIs(t, err, ErrDisabled)
			assertNoLeaderboardReaderCall(t, test.readers)
		})
	}
}

func TestServiceListRejectsInvalidQueriesBeforeReadingSettings(t *testing.T) {
	settingsErr := errors.New("settings should not be read")
	service := NewService(leaderboardSettingsStub{err: settingsErr}, Readers{})

	_, err := service.List(context.Background(), contract.LeaderboardCheckin, contract.LeaderboardQuery{
		Page: 1, PageSize: 20, Period: contract.LeaderboardPeriodDaily,
	})

	require.ErrorIs(t, err, ErrInvalidPeriod)
}

func assertNoLeaderboardReaderCall(t *testing.T, readers Readers) {
	t.Helper()
	if reader, ok := readers.Balance.(*balanceReaderStub); ok {
		require.Empty(t, reader.queries)
	}
	if reader, ok := readers.Consumption.(*consumptionReaderStub); ok {
		require.Empty(t, reader.queries)
	}
	if reader, ok := readers.Checkin.(*checkinReaderStub); ok {
		require.Empty(t, reader.queries)
	}
}
