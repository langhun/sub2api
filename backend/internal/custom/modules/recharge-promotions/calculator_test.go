package rechargepromotions

import (
	"context"
	"testing"
	"time"

	rechargepromotionsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestCalculatorAddsPercentageToBaseCredit(t *testing.T) {
	reader := &settingsReaderStub{config: rechargepromotionsettings.Config{
		Enabled: true, BonusRate: 20,
	}}
	calculator := NewCalculator(reader, nil)
	result, err := calculator.CalculateBalanceCredit(context.Background(), serviceInput(100, 120, time.Now()))
	require.NoError(t, err)
	require.Equal(t, 24.0, result.ExtraCredit)
}

func TestCalculatorRespectsWindowAndCap(t *testing.T) {
	start := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	reader := &settingsReaderStub{config: rechargepromotionsettings.Config{
		Enabled: true, BonusRate: 20, MaxBonus: 10, StartAt: start, EndAt: start.Add(time.Hour),
	}}
	calculator := NewCalculator(reader, nil)
	before, err := calculator.CalculateBalanceCredit(context.Background(), serviceInput(100, 120, start.Add(-time.Second)))
	require.NoError(t, err)
	require.Zero(t, before.ExtraCredit)
	active, err := calculator.CalculateBalanceCredit(context.Background(), serviceInput(100, 120, start))
	require.NoError(t, err)
	require.Equal(t, 10.0, active.ExtraCredit)
	expired, err := calculator.CalculateBalanceCredit(context.Background(), serviceInput(100, 120, start.Add(time.Hour)))
	require.NoError(t, err)
	require.Zero(t, expired.ExtraCredit)
}

// Keep the fixture local to avoid coupling calculator tests to service mocks.
func serviceInput(paymentAmount, baseCredit float64, now time.Time) service.BalanceCreditAdjustmentInput {
	return service.BalanceCreditAdjustmentInput{PaymentAmount: paymentAmount, BaseCredit: baseCredit, Now: now}
}

type settingsReaderStub struct {
	config rechargepromotionsettings.Config
}

func (s *settingsReaderStub) ReadRechargePromotions(context.Context) (rechargepromotionsettings.Config, error) {
	return s.config, nil
}
