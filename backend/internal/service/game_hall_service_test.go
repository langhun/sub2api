package service

import (
	"context"
	"testing"

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
	snapshot     *GameWalletSnapshot
	exchangePlan *GameExchangePlan
	slotPlan     *GameSlotRoundPlan
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

func TestGameHallService_ExchangeBalanceToDG_OneToOne(t *testing.T) {
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
			SettingKeyGameHallEnabled: "true",
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

func TestGameHallService_ExchangeRejectsWhenDisabled(t *testing.T) {
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

func TestGameHallService_GetHallStatusReturnsSlotsGame(t *testing.T) {
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
			SettingKeyGameHallEnabled: "true",
		},
	})

	status, err := svc.GetHallStatus(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, 88.0, status.MainBalance)
	require.Equal(t, 12.0, status.DGBalance)
	require.Equal(t, 1234.0, status.JackpotBalance)
	require.Len(t, status.Games, 1)
	require.Equal(t, GameTypeSlots, status.Games[0].Type)
}

func TestGameHallService_PlaySlotsDeductsDGAndReturnsOutcome(t *testing.T) {
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
			SettingKeyGameHallEnabled: "true",
		},
	})
	svc.SetSlotRoller(func() (float64, []string, string) {
		return 3, []string{"cherry", "cherry", "cherry"}, "中奖"
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

func TestDefaultSlotRoller_UsesRandomizedRollsInsteadOfFixedOutcome(t *testing.T) {
	previous := slotRandomIntN
	t.Cleanup(func() {
		slotRandomIntN = previous
	})

	slotRandomIntN = sequenceIntN(96, 96, 96)
	firstMultiplier, firstSymbols, firstMessage := defaultSlotRoller()

	slotRandomIntN = sequenceIntN(0, 25, 43)
	secondMultiplier, secondSymbols, secondMessage := defaultSlotRoller()

	require.Equal(t, 50.0, firstMultiplier)
	require.Equal(t, []string{"seven", "seven", "seven"}, firstSymbols)
	require.Equal(t, "中奖", firstMessage)

	require.Equal(t, 0.0, secondMultiplier)
	require.Equal(t, []string{"cherry", "lemon", "orange"}, secondSymbols)
	require.Equal(t, "未中奖", secondMessage)
}

func TestRollSlotWithIntN_ReturnsThreeOfAKindPayout(t *testing.T) {
	multiplier, symbols, message := rollSlotWithIntN(sequenceIntN(96, 96, 96))

	require.Equal(t, 50.0, multiplier)
	require.Equal(t, []string{"seven", "seven", "seven"}, symbols)
	require.Equal(t, "中奖", message)
}

func TestRollSlotWithIntN_ReturnsLoseForMixedSymbols(t *testing.T) {
	multiplier, symbols, message := rollSlotWithIntN(sequenceIntN(0, 25, 43))

	require.Equal(t, 0.0, multiplier)
	require.Equal(t, []string{"cherry", "lemon", "orange"}, symbols)
	require.Equal(t, "未中奖", message)
}

func sequenceIntN(values ...int) func(int) int {
	index := 0

	return func(max int) int {
		if len(values) == 0 {
			return 0
		}

		value := values[index%len(values)]
		index++
		if max <= 0 {
			return 0
		}
		if value >= 0 && value < max {
			return value
		}
		return value % max
	}
}
