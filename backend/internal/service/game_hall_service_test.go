//go:build unit

package service

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type gameHallSettingsReaderStub struct {
	values map[string]string
}

func (s *gameHallSettingsReaderStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

type gameHallStoreStub struct {
	snapshot           *GameWalletSnapshot
	exchangePlan       *GameExchangePlan
	slotPlan           *GameSlotRoundPlan
	dailyExchangeTotal float64
}

func (s *gameHallStoreStub) GetDailyExchangeTotal(_ context.Context, _ int64, _, _ time.Time) (float64, error) {
	return s.dailyExchangeTotal, nil
}

type gameHallBalanceCacheStub struct {
	userID int64
	err    error
}

func (s *gameHallBalanceCacheStub) InvalidateUserBalance(_ context.Context, userID int64) error {
	s.userID = userID
	return s.err
}

func (s *gameHallStoreStub) ListWalletTransactions(_ context.Context, _ *int64, _, _ int) ([]GameWalletTransaction, int64, error) {
	return []GameWalletTransaction{}, 0, nil
}

func (s *gameHallStoreStub) ListRounds(_ context.Context, _ *int64, _, _ int) ([]GameRound, int64, error) {
	return []GameRound{}, 0, nil
}

func (s *gameHallStoreStub) GetSnapshot(_ context.Context, _ int64) (*GameWalletSnapshot, error) {
	return s.snapshot, nil
}

func (s *gameHallStoreStub) CommitExchange(_ context.Context, plan GameExchangePlan) (*GameExchangeResult, error) {
	s.exchangePlan = &plan
	return &GameExchangeResult{
		Direction:         plan.Direction,
		Amount:            plan.Amount,
		MainBalanceBefore: plan.MainBalanceBefore,
		MainBalanceAfter:  plan.MainBalanceAfter,
		DGBalanceBefore:   plan.DGBalanceBefore,
		DGBalanceAfter:    plan.DGBalanceAfter,
	}, nil
}

func (s *gameHallStoreStub) CommitSlotRound(_ context.Context, plan GameSlotRoundPlan) (*GamePlayResult, error) {
	s.slotPlan = &plan
	return &GamePlayResult{
		GameType:        plan.GameType,
		BetAmount:       plan.BetAmount,
		PayoutAmount:    plan.PayoutAmount,
		NetAmount:       plan.NetAmount,
		Multiplier:      plan.Multiplier,
		DGBalanceBefore: plan.DGBalanceBefore,
		DGBalanceAfter:  plan.DGBalanceAfter,
		JackpotBalance:  plan.JackpotAfter,
		Outcome:         plan.Outcome,
		Symbols:         plan.Symbols,
		Message:         plan.Message,
	}, nil
}

func TestGameHallServiceExchangeBalanceToDGOneToOne(t *testing.T) {
	store := &gameHallStoreStub{
		snapshot: &GameWalletSnapshot{
			UserID:         1,
			MainBalance:    80,
			DGBalance:      5,
			JackpotBalance: 99,
		},
	}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{
		values: map[string]string{
			SettingKeyGameHallEnabled:  "true",
			SettingKeyGameSlotsEnabled: "true",
		},
	})

	result, err := svc.Exchange(context.Background(), GameExchangeInput{
		UserID:         1,
		Direction:      GameExchangeBalanceToDG,
		Amount:         20,
		IdempotencyKey: "exchange-1",
	})

	require.NoError(t, err)
	require.NotNil(t, store.exchangePlan)
	require.Equal(t, 60.0, result.MainBalanceAfter)
	require.Equal(t, 25.0, result.DGBalanceAfter)
	require.Equal(t, 60.0, store.exchangePlan.MainBalanceAfter)
	require.Equal(t, 25.0, store.exchangePlan.DGBalanceAfter)
}

func TestGameHallServiceExchangeEnforcesConfiguredRangeAndDailyLimit(t *testing.T) {
	settings := &gameHallSettingsReaderStub{values: map[string]string{
		SettingKeyGameHallEnabled: "true", SettingKeyGameExchangeMinAmount: "10",
		SettingKeyGameExchangeMaxAmount: "50", SettingKeyGameExchangeDailyLimit: "100",
	}}
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1, MainBalance: 200}}
	svc := NewGameHallService(store, settings)

	_, err := svc.Exchange(context.Background(), GameExchangeInput{UserID: 1, Direction: GameExchangeBalanceToDG, Amount: 9.99})
	require.ErrorIs(t, err, ErrGameExchangeOutOfRange)
	_, err = svc.Exchange(context.Background(), GameExchangeInput{UserID: 1, Direction: GameExchangeBalanceToDG, Amount: 50.01})
	require.ErrorIs(t, err, ErrGameExchangeOutOfRange)

	store.dailyExchangeTotal = 50
	_, err = svc.Exchange(context.Background(), GameExchangeInput{UserID: 1, Direction: GameExchangeBalanceToDG, Amount: 50})
	require.NoError(t, err)
	require.Equal(t, 100.0, store.exchangePlan.DailyLimit)
	require.False(t, store.exchangePlan.DayStart.IsZero())
	require.Equal(t, store.exchangePlan.DayStart.AddDate(0, 0, 1), store.exchangePlan.DayEnd)
}

func TestGameHallServiceExchangeRejectsDGReturnWhenDisabled(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1, DGBalance: 100}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{
		SettingKeyGameHallEnabled: "true", SettingKeyGameExchangeAllowDGToBalance: "false",
	}})

	_, err := svc.Exchange(context.Background(), GameExchangeInput{UserID: 1, Direction: GameExchangeDGToBalance, Amount: 10})
	require.ErrorIs(t, err, ErrGameExchangeReturnDisabled)
	require.Nil(t, store.exchangePlan)
}

func TestGameHallServiceGeneratesDistinctKeysWhenCallerOmitsIdempotencyKey(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1, MainBalance: 80, DGBalance: 5}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{SettingKeyGameHallEnabled: "true"}})

	_, err := svc.Exchange(context.Background(), GameExchangeInput{UserID: 1, Direction: GameExchangeBalanceToDG, Amount: 1})
	require.NoError(t, err)
	firstKey := store.exchangePlan.IdempotencyKey
	require.NotEmpty(t, firstKey)

	_, err = svc.Exchange(context.Background(), GameExchangeInput{UserID: 1, Direction: GameExchangeBalanceToDG, Amount: 1})
	require.NoError(t, err)
	require.NotEmpty(t, store.exchangePlan.IdempotencyKey)
	require.NotEqual(t, firstKey, store.exchangePlan.IdempotencyKey)
}

func TestGameHallServiceExchangeReturnsCommittedResultWhenCacheInvalidationFails(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 9, MainBalance: 80, DGBalance: 5}}
	cache := &gameHallBalanceCacheStub{err: context.DeadlineExceeded}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{SettingKeyGameHallEnabled: "true"}}, cache)

	result, err := svc.Exchange(context.Background(), GameExchangeInput{UserID: 9, Direction: GameExchangeBalanceToDG, Amount: 20, IdempotencyKey: "cache-failure"})

	require.NoError(t, err)
	require.Equal(t, int64(9), cache.userID)
	require.Equal(t, 60.0, result.MainBalanceAfter)
	require.Equal(t, 25.0, result.DGBalanceAfter)
}

func TestGameHallServiceGetHallStatusRejectsWhenMasterSwitchDisabled(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{SettingKeyGameHallEnabled: "false"}})

	_, err := svc.GetHallStatus(context.Background(), 1)

	require.ErrorIs(t, err, ErrGameHallDisabled)
}

func TestGameHallServiceListsUserAuditWithNormalizedPagination(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{SettingKeyGameHallEnabled: "true"}})

	transactions, transactionTotal, err := svc.ListUserTransactions(context.Background(), 1, 0, 1000)
	require.NoError(t, err)
	require.Empty(t, transactions)
	require.Zero(t, transactionTotal)

	rounds, roundTotal, err := svc.ListUserRounds(context.Background(), 1, -1, -1)
	require.NoError(t, err)
	require.Empty(t, rounds)
	require.Zero(t, roundTotal)
}

func TestGameHallServiceExchangeRejectsWhenDisabled(t *testing.T) {
	store := &gameHallStoreStub{
		snapshot: &GameWalletSnapshot{
			UserID:      1,
			MainBalance: 80,
			DGBalance:   5,
		},
	}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{
		values: map[string]string{
			SettingKeyGameHallEnabled: "false",
		},
	})

	_, err := svc.Exchange(context.Background(), GameExchangeInput{
		UserID:         1,
		Direction:      GameExchangeBalanceToDG,
		Amount:         20,
		IdempotencyKey: "exchange-2",
	})

	require.ErrorIs(t, err, ErrGameHallDisabled)
	require.Nil(t, store.exchangePlan)
}

func TestGameHallServiceGetHallStatusReturnsSlotsGame(t *testing.T) {
	store := &gameHallStoreStub{
		snapshot: &GameWalletSnapshot{
			UserID:         1,
			MainBalance:    88,
			DGBalance:      12,
			JackpotBalance: 1234,
		},
	}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{
		values: map[string]string{
			SettingKeyGameHallEnabled:  "true",
			SettingKeyGameSlotsEnabled: "true",
		},
	})

	status, err := svc.GetHallStatus(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, 88.0, status.MainBalance)
	require.Equal(t, 12.0, status.DGBalance)
	require.Equal(t, 1234.0, status.JackpotBalance)
	require.Equal(t, 0.01, status.ExchangeMinAmount)
	require.Equal(t, 1000.0, status.ExchangeMaxAmount)
	require.Equal(t, 1000.0, status.ExchangeDailyLimit)
	require.Equal(t, 1000.0, status.ExchangeDailyRemaining)
	require.True(t, status.ExchangeAllowDGToBalance)
	require.Len(t, status.Games, 1)
	require.Equal(t, GameTypeSlots, status.Games[0].Type)
	require.Equal(t, slotRuleVersion, status.Games[0].RuleVersion)
	require.InDelta(t, 0.953, status.Games[0].TheoreticalRTP, 0.0001)
	require.Len(t, status.Games[0].PayoutRules, len(slotSymbolTable))
	require.Equal(t, "cherry", status.Games[0].PayoutRules[0].Symbol)
	require.Equal(t, 3, status.Games[0].PayoutRules[0].MatchCount)
	require.Equal(t, 18.7, status.Games[0].PayoutRules[0].Multiplier)
	require.InDelta(t, math.Pow(25.0/98.0, 3), status.Games[0].PayoutRules[0].Probability, 0.00000001)
}

func TestGameHallServicePlaySlotsDeductsDGAndReturnsOutcome(t *testing.T) {
	store := &gameHallStoreStub{
		snapshot: &GameWalletSnapshot{
			UserID:         1,
			MainBalance:    88,
			DGBalance:      50,
			JackpotBalance: 100,
		},
	}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{
		values: map[string]string{
			SettingKeyGameHallEnabled:  "true",
			SettingKeyGameSlotsEnabled: "true",
		},
	})
	svc.SetSlotRoller(func() (float64, []string, string, error) {
		return 3, []string{"cherry", "cherry", "cherry"}, "中奖", nil
	})

	result, err := svc.Play(context.Background(), GamePlayInput{
		UserID:         1,
		GameType:       GameTypeSlots,
		BetAmount:      10,
		IdempotencyKey: "slot-1",
	})

	require.NoError(t, err)
	require.NotNil(t, store.slotPlan)
	require.Equal(t, 10.0, store.slotPlan.BetAmount)
	require.Equal(t, 30.0, store.slotPlan.PayoutAmount)
	require.Equal(t, 50.0, result.DGBalanceBefore)
	require.Equal(t, 70.0, result.DGBalanceAfter)
	require.Equal(t, 80.0, result.JackpotBalance)
	require.Equal(t, 20.0, result.NetAmount)
}

func TestGameHallServicePlayHonorsSlotsSwitchAndBetRange(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1, DGBalance: 100}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{
		SettingKeyGameHallEnabled: "true", SettingKeyGameSlotsEnabled: "false",
	}})
	_, err := svc.Play(context.Background(), GamePlayInput{UserID: 1, GameType: GameTypeSlots, BetAmount: 5, IdempotencyKey: "off"})
	require.ErrorIs(t, err, ErrGameSlotsDisabled)

	svc = NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{
		SettingKeyGameHallEnabled: "true", SettingKeyGameSlotsEnabled: "true",
		SettingKeyGameSlotsMinBet: "10", SettingKeyGameSlotsMaxBet: "20",
	}})
	_, err = svc.Play(context.Background(), GamePlayInput{UserID: 1, GameType: GameTypeSlots, BetAmount: 5, IdempotencyKey: "small"})
	require.ErrorIs(t, err, ErrGameBetOutOfRange)
	_, err = svc.Play(context.Background(), GamePlayInput{UserID: 1, GameType: GameTypeSlots, BetAmount: 25, IdempotencyKey: "large"})
	require.ErrorIs(t, err, ErrGameBetOutOfRange)
}

func TestGameHallServiceRejectsInvalidPersistedBetRange(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1, DGBalance: 100}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{
		SettingKeyGameHallEnabled: "true", SettingKeyGameSlotsEnabled: "true",
		SettingKeyGameSlotsMinBet: "20", SettingKeyGameSlotsMaxBet: "10",
	}})

	_, err := svc.GetHallStatus(context.Background(), 1)

	require.Error(t, err)
	require.Nil(t, store.slotPlan)
}

func TestDefaultSlotRollerUsesRandomizedRollsInsteadOfFixedOutcome(t *testing.T) {
	previous := slotRandomIntN
	t.Cleanup(func() {
		slotRandomIntN = previous
	})

	slotRandomIntN = sequenceIntN(96, 96, 96)
	firstMultiplier, firstSymbols, firstMessage, err := defaultSlotRoller()
	require.NoError(t, err)

	slotRandomIntN = sequenceIntN(0, 25, 43)
	secondMultiplier, secondSymbols, secondMessage, err := defaultSlotRoller()
	require.NoError(t, err)

	require.Equal(t, 320.0, firstMultiplier)
	require.Equal(t, []string{"seven", "seven", "seven"}, firstSymbols)
	require.Equal(t, "中奖", firstMessage)

	require.Equal(t, 0.0, secondMultiplier)
	require.Equal(t, []string{"cherry", "lemon", "orange"}, secondSymbols)
	require.Equal(t, "未中奖", secondMessage)
}

func TestRollSlotWithIntNReturnsThreeOfAKindPayout(t *testing.T) {
	multiplier, symbols, message, err := rollSlotWithIntN(sequenceIntN(96, 96, 96))
	require.NoError(t, err)

	require.Equal(t, 320.0, multiplier)
	require.Equal(t, []string{"seven", "seven", "seven"}, symbols)
	require.Equal(t, "中奖", message)
}

func TestSlotRuleTheoreticalRTPMatchesTarget(t *testing.T) {
	require.InDelta(t, 0.953, slotTheoreticalRTP(), 0.0001)
}

func TestSlotRuleDeterministicDistributionMatchesTheoreticalRTP(t *testing.T) {
	totalPayout := 0.0
	totalRounds := 0
	for first := 0; first < slotTotalWeight; first++ {
		for second := 0; second < slotTotalWeight; second++ {
			for third := 0; third < slotTotalWeight; third++ {
				multiplier, _, _, err := rollSlotWithIntN(sequenceIntN(first, second, third))
				require.NoError(t, err)
				totalPayout += multiplier
				totalRounds++
			}
		}
	}
	require.InDelta(t, 0.953, totalPayout/float64(totalRounds), 0.0001)
}

func TestRollSlotWithIntNReturnsLoseForMixedSymbols(t *testing.T) {
	multiplier, symbols, message, err := rollSlotWithIntN(sequenceIntN(0, 25, 43))
	require.NoError(t, err)

	require.Equal(t, 0.0, multiplier)
	require.Equal(t, []string{"cherry", "lemon", "orange"}, symbols)
	require.Equal(t, "未中奖", message)
}

func TestGameHallServicePlayRejectsWhenSecureRandomFails(t *testing.T) {
	store := &gameHallStoreStub{snapshot: &GameWalletSnapshot{UserID: 1, DGBalance: 100, JackpotBalance: 100}}
	svc := NewGameHallService(store, &gameHallSettingsReaderStub{values: map[string]string{
		SettingKeyGameHallEnabled: "true", SettingKeyGameSlotsEnabled: "true",
	}})
	svc.SetSlotRoller(func() (float64, []string, string, error) {
		return 0, nil, "", context.DeadlineExceeded
	})

	_, err := svc.Play(context.Background(), GamePlayInput{UserID: 1, GameType: GameTypeSlots, BetAmount: 1, IdempotencyKey: "rng-error"})

	require.ErrorIs(t, err, ErrGameRandomUnavailable)
	require.Nil(t, store.slotPlan)
}

func sequenceIntN(values ...int) func(int) (int, error) {
	index := 0

	return func(max int) (int, error) {
		if len(values) == 0 {
			return 0, nil
		}

		value := values[index%len(values)]
		index++
		if max <= 0 {
			return 0, nil
		}
		if value >= 0 && value < max {
			return value, nil
		}
		return value % max, nil
	}
}
