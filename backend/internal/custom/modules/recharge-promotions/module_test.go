package rechargepromotions

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestPercentagePromotionHonorsCapAndWindow(t *testing.T) {
	now := time.Date(2026, 8, 12, 1, 0, 0, 0, time.UTC)
	module := NewModule(settingsReaderStub{config: settings.Config{Enabled: true, Mode: settings.ModePercentage, PercentageRate: 20, MinimumAmount: 10, MaximumBonus: 5, StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}}, nil)
	result, err := module.PrepareBalanceOrder(context.Background(), service.BalanceOrderExtensionInput{PaymentAmount: 100, BaseCredit: 120, Now: now})
	require.NoError(t, err)
	require.Equal(t, 125.0, result.CreditedAmount)
}

func TestTieredPromotionUsesHighestEligibleTier(t *testing.T) {
	module := NewModule(settingsReaderStub{config: settings.Config{Enabled: true, Mode: settings.ModeTiered, TierRules: []settings.TierRule{{MinimumAmount: 10, BonusCredit: 1}, {MinimumAmount: 50, BonusCredit: 8}}}}, nil)
	result, err := module.PrepareBalanceOrder(context.Background(), service.BalanceOrderExtensionInput{PaymentAmount: 80, BaseCredit: 80, Now: time.Now()})
	require.NoError(t, err)
	require.Equal(t, 88.0, result.CreditedAmount)
}

func TestTieredPromotionDoesNotStackRules(t *testing.T) {
	module := NewModule(settingsReaderStub{config: settings.Config{Enabled: true, Mode: settings.ModeTiered, TierRules: []settings.TierRule{{MinimumAmount: 10, BonusCredit: 1}, {MinimumAmount: 50, BonusCredit: 8}}}}, nil)
	calculated, err := module.calculate(context.Background(), 80, 80, time.Now())
	require.NoError(t, err)
	require.Equal(t, "8.00", calculated.BonusCredit.StringFixed(2))
}

func TestTieredPreviewReportsFirstQualifyingAmount(t *testing.T) {
	module := NewModule(settingsReaderStub{config: settings.Config{Enabled: true, Mode: settings.ModeTiered, TierRules: []settings.TierRule{{MinimumAmount: 10, BonusCredit: 1}, {MinimumAmount: 50, BonusCredit: 8}}}}, nil)
	calculated, err := module.calculate(context.Background(), 5, 5, time.Now())
	require.NoError(t, err)
	require.False(t, calculated.Eligible)
	require.Equal(t, 10.0, calculated.minimumAmount())
}

func TestInactivePromotionPreservesBaseCredit(t *testing.T) {
	module := NewModule(settingsReaderStub{config: settings.Config{Enabled: false, Mode: settings.ModePercentage, PercentageRate: 20}}, nil)
	result, err := module.PrepareBalanceOrder(context.Background(), service.BalanceOrderExtensionInput{PaymentAmount: 100, BaseCredit: 120, Now: time.Now()})
	require.NoError(t, err)
	require.Equal(t, 120.0, result.CreditedAmount)
}

func TestProportionalCreditPreservesRefundTotal(t *testing.T) {
	base := proportionalCredit(decimalFrom(120), decimalFrom(144), decimalFrom(72))
	require.Equal(t, "60.00", base.StringFixed(2))
}

func decimalFrom(value float64) decimal.Decimal { return decimal.NewFromFloat(value) }

type settingsReaderStub struct{ config settings.Config }

func (s settingsReaderStub) ReadRechargePromotions(context.Context) (settings.Config, error) {
	return s.config, nil
}
