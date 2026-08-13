package rechargepromotions

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
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

func TestTieredPercentagePromotionBoundariesAndCap(t *testing.T) {
	module := NewModule(settingsReaderStub{config: settings.Config{
		Enabled: true, Mode: settings.ModeTieredPercentage, MaximumBonus: 12,
		TieredPercentageRules: []settings.TieredPercentageRule{
			{MinimumAmount: 10, MaximumAmount: 50, PercentageRate: 5},
			{MinimumAmount: 50, MaximumAmount: 100, PercentageRate: 8},
			{MinimumAmount: 100, MaximumAmount: 200, PercentageRate: 10},
			{MinimumAmount: 200, MaximumAmount: 0, PercentageRate: 15},
		},
	}}, nil)

	tests := []struct {
		payment float64
		base    float64
		credit  float64
		active  bool
		rate    string
	}{
		{payment: 9.99, base: 9.99, credit: 9.99, active: false, rate: "0.00"},
		{payment: 10, base: 10, credit: 10.5, active: true, rate: "5.00"},
		{payment: 49.99, base: 49.99, credit: 52.49, active: true, rate: "5.00"},
		{payment: 50, base: 50, credit: 54, active: true, rate: "8.00"},
		{payment: 99.99, base: 99.99, credit: 107.99, active: true, rate: "8.00"},
		{payment: 100, base: 100, credit: 110, active: true, rate: "10.00"},
		{payment: 199.99, base: 199.99, credit: 211.99, active: true, rate: "10.00"},
		{payment: 200, base: 200, credit: 212, active: true, rate: "15.00"},
	}
	for _, test := range tests {
		calculated, err := module.calculate(context.Background(), test.payment, test.base, time.Now())
		require.NoError(t, err)
		require.Equal(t, test.active, calculated.Eligible, "payment %.2f", test.payment)
		require.Equal(t, test.credit, calculated.FinalCredit.InexactFloat64(), "payment %.2f", test.payment)
		require.Equal(t, test.rate, calculated.AppliedPercentageRate.StringFixed(2), "payment %.2f", test.payment)
	}
}

func TestReadOrderRequiresCompletedPayment(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	mock.ExpectQuery(`(?s)FROM custom_recharge_promotion_orders promotion.*JOIN payment_orders payment.*payment\.completed_at IS NOT NULL`).
		WithArgs(int64(48)).
		WillReturnRows(sqlmock.NewRows([]string{"payment_order_id", "user_id", "mode", "payment_amount", "pay_amount", "base_credit", "bonus_credit", "final_credit", "refunded_credit", "refunded_base_credit", "refunded_bonus_credit", "created_at", "completed_at"}))

	_, found, err := NewModule(settingsReaderStub{}, db).readOrder(context.Background(), 48, nil)
	require.NoError(t, err)
	require.False(t, found)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSummaryOnlyCountsCompletedPayments(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(`(?s)FROM custom_recharge_promotion_orders promotion.*JOIN payment_orders payment.*payment\.completed_at IS NOT NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count", "bonus", "final", "refunded"}).AddRow(2, 10, 110, 0))

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/custom/admin/recharge-promotions/summary", nil)
	NewModule(settingsReaderStub{config: settings.Default()}, db).summary(context)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListOrdersOnlyReturnsCompletedPayments(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectQuery(`(?s)SELECT COUNT\(\*\).*JOIN payment_orders payment.*payment\.completed_at IS NOT NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`(?s)FROM custom_recharge_promotion_orders promotion.*JOIN payment_orders payment.*payment\.completed_at IS NOT NULL.*ORDER BY payment\.completed_at DESC`).
		WithArgs(20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"payment_order_id"}))

	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/custom/admin/recharge-promotions/orders", nil)
	NewModule(settingsReaderStub{}, db).listOrders(context)
	require.NoError(t, mock.ExpectationsWereMet())
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
