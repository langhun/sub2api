package checkin

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/stretchr/testify/require"
)

type checkinTestContextKey struct{}

type checkinTransactionStub struct{ calls int }

func (s *checkinTransactionStub) RunInTransaction(ctx context.Context, operation func(context.Context) error) error {
	s.calls++
	return operation(context.WithValue(ctx, checkinTestContextKey{}, true))
}

type checkinRepositoryStub struct {
	todayRecords  []*Record
	previous      *Record
	lockedAccount contract.Account
	created       []*Record
	locks         int
}

func (s *checkinRepositoryStub) FindToday(context.Context, int64, time.Time) (*Record, error) {
	if len(s.todayRecords) == 0 {
		return nil, nil
	}
	record := s.todayRecords[0]
	s.todayRecords = s.todayRecords[1:]
	return record, nil
}

func (s *checkinRepositoryStub) FindPrevious(context.Context, int64, time.Time) (*Record, error) {
	return s.previous, nil
}

func (s *checkinRepositoryStub) Create(ctx context.Context, record *Record) error {
	if ctx.Value(checkinTestContextKey{}) != true {
		return ErrCheckinNotAllowed
	}
	record.ID = int64(100 + len(s.created))
	s.created = append(s.created, record)
	return nil
}

func (s *checkinRepositoryStub) ListCalendar(context.Context, int64, time.Time, time.Time) ([]Record, error) {
	return nil, nil
}

func (s *checkinRepositoryStub) LockAccount(ctx context.Context, _ int64) error {
	if ctx.Value(checkinTestContextKey{}) != true {
		return ErrCheckinNotAllowed
	}
	s.locks++
	return nil
}

func (s *checkinRepositoryStub) GetLockedAccount(ctx context.Context, _ int64) (contract.Account, error) {
	if ctx.Value(checkinTestContextKey{}) != true {
		return contract.Account{}, ErrCheckinNotAllowed
	}
	return s.lockedAccount, nil
}

type checkinSettingsStub struct{ value contract.Settings }

func (s checkinSettingsStub) GetActivitySettings(context.Context) (contract.Settings, error) {
	return s.value, nil
}

type checkinAccountStub struct{ account contract.Account }

func (s checkinAccountStub) GetAccount(context.Context, int64) (contract.Account, error) {
	return s.account, nil
}

type checkinBalanceStub struct {
	credits []contract.BalanceOperation
	debits  []contract.BalanceOperation
}

func (s *checkinBalanceStub) Credit(ctx context.Context, operation contract.BalanceOperation) error {
	if ctx.Value(checkinTestContextKey{}) != true {
		return ErrCheckinNotAllowed
	}
	s.credits = append(s.credits, operation)
	return nil
}

func (s *checkinBalanceStub) DebitIfSufficient(ctx context.Context, operation contract.BalanceOperation) (bool, error) {
	if ctx.Value(checkinTestContextKey{}) != true {
		return false, ErrCheckinNotAllowed
	}
	s.debits = append(s.debits, operation)
	return true, nil
}

type checkinLedgerStub struct{ entries []CheckinAuditEntry }

func (s *checkinLedgerStub) RecordCheckinAdjustment(ctx context.Context, entry CheckinAuditEntry) error {
	if ctx.Value(checkinTestContextKey{}) != true {
		return ErrCheckinNotAllowed
	}
	s.entries = append(s.entries, entry)
	return nil
}

type checkinCacheStub struct{ invalidated []int64 }

func (s *checkinCacheStub) InvalidateBalance(_ context.Context, userID int64) error {
	s.invalidated = append(s.invalidated, userID)
	return nil
}

type checkinBlindboxStub struct {
	prepared     *PreparedBlindbox
	prepareCalls int
	deliverCalls int
}

func (s *checkinBlindboxStub) PrepareForCheckin(ctx context.Context, _ int64, _ int64, _ int) (*PreparedBlindbox, error) {
	if ctx.Value(checkinTestContextKey{}) != true {
		return nil, ErrCheckinNotAllowed
	}
	s.prepareCalls++
	return s.prepared, nil
}

func (s *checkinBlindboxStub) Deliver(context.Context, int64) (*DeliveredBlindbox, error) {
	s.deliverCalls++
	return &DeliveredBlindbox{RewardDetail: "credited"}, nil
}

type checkinClockStub struct{ now time.Time }

func (s checkinClockStub) Today() time.Time { return s.now }
func (s checkinClockStub) Now() time.Time   { return s.now }

type checkinRandomStub struct{ value float64 }

func (s checkinRandomStub) Float64() (float64, error) { return s.value, nil }

func TestServiceCheckinKeepsActivityMutationsInOneTransaction(t *testing.T) {
	today := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.Local)
	repository := &checkinRepositoryStub{}
	transactions := &checkinTransactionStub{}
	balance := &checkinBalanceStub{}
	ledger := &checkinLedgerStub{}
	cache := &checkinCacheStub{}
	blindbox := &checkinBlindboxStub{prepared: &PreparedBlindbox{
		DeliveryID: 77,
		Result:     contract.BlindboxResult{PrizeName: "Bonus", RewardValue: 3},
	}}
	service, err := NewService(Dependencies{
		Repository: repository, Transactions: transactions,
		Accounts: checkinAccountStub{account: contract.Account{ID: 9, Status: accountStatusActive, Balance: 20}},
		Balance:  balance, Ledger: ledger,
		Settings: checkinSettingsStub{value: contract.Settings{Checkin: contract.CheckinSettings{
			Enabled: true, MinimumReward: 1, MaximumReward: 5,
		}}},
		Cache: cache, Blindbox: blindbox, Clock: checkinClockStub{now: today}, Random: checkinRandomStub{value: 0.5},
	})
	require.NoError(t, err)

	result, err := service.Checkin(context.Background(), 9)

	require.NoError(t, err)
	require.Equal(t, 1, transactions.calls)
	require.Equal(t, 1, repository.locks)
	require.Len(t, repository.created, 1)
	require.Equal(t, 3.0, repository.created[0].RewardAmount)
	require.Equal(t, checkinTypeNormal, repository.created[0].CheckinType)
	require.Equal(t, []contract.BalanceOperation{{UserID: 9, Amount: 3, Reason: adjustmentTypeCheckin, IdempotencyKey: "checkin:9:2026-07-25"}}, balance.credits)
	require.Len(t, ledger.entries, 1)
	require.Equal(t, adjustmentTypeCheckin, ledger.entries[0].Type)
	require.Equal(t, []int64{9}, cache.invalidated)
	require.Equal(t, 1, blindbox.prepareCalls)
	require.Equal(t, 1, blindbox.deliverCalls)
	require.NotNil(t, result.Blindbox)
	require.Equal(t, "Bonus", result.Blindbox.PrizeName)
	require.Equal(t, "credited", result.Blindbox.RewardDetail)
}

func TestServiceLuckCheckinUsesLockedBalanceForMaximumBet(t *testing.T) {
	today := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.Local)
	repository := &checkinRepositoryStub{lockedAccount: contract.Account{ID: 9, Status: accountStatusActive, Balance: 4.5}}
	balance := &checkinBalanceStub{}
	ledger := &checkinLedgerStub{}
	service, err := NewService(Dependencies{
		Repository: repository, Transactions: &checkinTransactionStub{},
		Accounts: checkinAccountStub{account: contract.Account{ID: 9, Status: accountStatusActive, Balance: 10}},
		Balance:  balance, Ledger: ledger,
		Settings: checkinSettingsStub{value: contract.Settings{Checkin: contract.CheckinSettings{
			LuckEnabled: true, MinimumMultiplier: 1, MaximumMultiplier: 3,
		}}},
		Clock: checkinClockStub{now: today}, Random: checkinRandomStub{value: 0.5},
	})
	require.NoError(t, err)

	result, err := service.LuckCheckin(context.Background(), 9, 10, true)

	require.NoError(t, err)
	require.Equal(t, 4.5, result.BetAmount)
	require.Equal(t, 2.0, result.Multiplier)
	require.Equal(t, 4.5, result.RewardAmount)
	require.Equal(t, []contract.BalanceOperation{{UserID: 9, Amount: 4.5, Reason: adjustmentTypeCheckinLuck, IdempotencyKey: "checkin:9:2026-07-25"}}, balance.credits)
	require.Empty(t, balance.debits)
	require.Len(t, ledger.entries, 1)
	require.Equal(t, adjustmentTypeCheckinLuck, ledger.entries[0].Type)
	require.Equal(t, 4.5, ledger.entries[0].BetAmount)
}

func TestServiceLuckCheckinDebitsNegativeRewardWithoutRechargeAccounting(t *testing.T) {
	today := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.Local)
	repository := &checkinRepositoryStub{lockedAccount: contract.Account{ID: 9, Status: accountStatusActive, Balance: 10}}
	balance := &checkinBalanceStub{}
	service, err := NewService(Dependencies{
		Repository: repository, Transactions: &checkinTransactionStub{},
		Accounts: checkinAccountStub{account: contract.Account{ID: 9, Status: accountStatusActive, Balance: 10}},
		Balance:  balance, Ledger: &checkinLedgerStub{},
		Settings: checkinSettingsStub{value: contract.Settings{Checkin: contract.CheckinSettings{
			LuckEnabled: true, MinimumMultiplier: 0.5, MaximumMultiplier: 0.5,
		}}},
		Clock: checkinClockStub{now: today}, Random: checkinRandomStub{},
	})
	require.NoError(t, err)

	result, err := service.LuckCheckin(context.Background(), 9, 5, false)

	require.NoError(t, err)
	require.Equal(t, -2.5, result.RewardAmount)
	require.Empty(t, balance.credits)
	require.Equal(t, []contract.BalanceOperation{{UserID: 9, Amount: 2.5, Reason: adjustmentTypeCheckinLuck, IdempotencyKey: "checkin:9:2026-07-25"}}, balance.debits)
}

func TestServiceCheckinReturnsConcurrentRecordWithoutRepeatingEffects(t *testing.T) {
	today := time.Date(2026, time.July, 25, 0, 0, 0, 0, time.Local)
	existing := &Record{ID: 12, UserID: 9, CheckinDate: today, RewardAmount: 8, StreakDays: 4, CheckinType: checkinTypeLuck, BetAmount: 5, Multiplier: 1.6}
	repository := &checkinRepositoryStub{todayRecords: []*Record{nil, existing}}
	balance := &checkinBalanceStub{}
	ledger := &checkinLedgerStub{}
	cache := &checkinCacheStub{}
	service, err := NewService(Dependencies{
		Repository: repository, Transactions: &checkinTransactionStub{},
		Accounts: checkinAccountStub{account: contract.Account{ID: 9, Status: accountStatusActive, Balance: 10}},
		Balance:  balance, Ledger: ledger,
		Settings: checkinSettingsStub{value: contract.Settings{Checkin: contract.CheckinSettings{Enabled: true, MinimumReward: 1, MaximumReward: 1}}},
		Cache:    cache, Clock: checkinClockStub{now: today}, Random: checkinRandomStub{},
	})
	require.NoError(t, err)

	result, err := service.Checkin(context.Background(), 9)

	require.NoError(t, err)
	require.Equal(t, 8.0, result.RewardAmount)
	require.Equal(t, checkinTypeLuck, result.CheckinType)
	require.Empty(t, repository.created)
	require.Empty(t, balance.credits)
	require.Empty(t, ledger.entries)
	require.Empty(t, cache.invalidated)
}

func TestResolveLuckCheckinBetAmount(t *testing.T) {
	largeBet := 81058106151016.73
	oneUnitBelow := math.Nextafter(largeBet, math.Inf(-1))
	cases := []struct {
		name, want   string
		bet, balance float64
		useMax, ok   bool
		amount       float64
	}{
		{name: "normal", bet: 2.5, balance: 10, amount: 2.5, ok: true},
		{name: "stale maximum", bet: 10, balance: 8.5, useMax: true, amount: 8.5, ok: true},
		{name: "large strict amount", bet: largeBet, balance: oneUnitBelow, ok: false},
		{name: "non-finite amount", bet: math.Inf(1), balance: 10, ok: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			amount, ok := resolveLuckCheckinBetAmount(testCase.bet, testCase.balance, testCase.useMax)
			require.Equal(t, testCase.ok, ok)
			require.Equal(t, testCase.amount, amount)
		})
	}
}
