package rechargepromotions

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	rechargepromotionsettings "github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

type Calculator struct {
	settings settingsReader
	db       *sql.DB
}

type settingsReader interface {
	ReadRechargePromotions(context.Context) (rechargepromotionsettings.Config, error)
}

func NewCalculator(settings settingsReader, db *sql.DB) *Calculator {
	return &Calculator{settings: settings, db: db}
}

func (c *Calculator) CalculateBalanceCredit(ctx context.Context, input service.BalanceCreditAdjustmentInput) (service.BalanceCreditAdjustmentResult, error) {
	config, err := c.settings.ReadRechargePromotions(ctx)
	if err != nil {
		return service.BalanceCreditAdjustmentResult{}, err
	}
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if !config.Enabled || config.BonusRate <= 0 || input.PaymentAmount < config.MinAmount ||
		(!config.StartAt.IsZero() && now.Before(config.StartAt)) ||
		(!config.EndAt.IsZero() && !now.Before(config.EndAt)) {
		return service.BalanceCreditAdjustmentResult{}, nil
	}
	if !finiteNonnegative(input.PaymentAmount) || !finiteNonnegative(input.BaseCredit) {
		return service.BalanceCreditAdjustmentResult{}, fmt.Errorf("recharge promotion input contains invalid amount")
	}
	base := decimal.NewFromFloat(input.BaseCredit).Round(2)
	bonus := base.Mul(decimal.NewFromFloat(config.BonusRate)).Div(decimal.NewFromInt(100)).Round(2)
	if config.MaxBonus > 0 {
		maxBonus := decimal.NewFromFloat(config.MaxBonus).Round(2)
		if bonus.GreaterThan(maxBonus) {
			bonus = maxBonus
		}
	}
	amount := bonus.InexactFloat64()
	if !finiteNonnegative(amount) {
		return service.BalanceCreditAdjustmentResult{}, fmt.Errorf("recharge promotion produced invalid bonus")
	}
	return service.BalanceCreditAdjustmentResult{ExtraCredit: amount, Snapshot: map[string]string{
		"bonus_rate":          decimal.NewFromFloat(config.BonusRate).Round(2).StringFixed(2),
		"min_amount":          decimal.NewFromFloat(config.MinAmount).Round(2).StringFixed(2),
		"max_bonus":           decimal.NewFromFloat(config.MaxBonus).Round(2).StringFixed(2),
		"campaign_started_at": rechargepromotionsettings.FormatTime(config.StartAt),
		"campaign_ended_at":   rechargepromotionsettings.FormatTime(config.EndAt),
	}}, nil
}

func (c *Calculator) RecordBalanceCreditAdjustment(ctx context.Context, orderID int64, input service.BalanceCreditAdjustmentInput, result service.BalanceCreditAdjustmentResult) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("recharge promotion database is required")
	}
	if result.ExtraCredit <= 0 {
		return nil
	}
	baseCredit := decimal.NewFromFloat(input.BaseCredit).Round(2)
	bonusCredit := decimal.NewFromFloat(result.ExtraCredit).Round(2)
	finalCredit := baseCredit.Add(bonusCredit)
	_, err := c.db.ExecContext(ctx, `
INSERT INTO custom_recharge_promotion_orders
    (payment_order_id, user_id, payment_amount, pay_amount, base_credit, bonus_credit, final_credit,
     bonus_rate, min_amount, max_bonus, campaign_started_at, campaign_ended_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, '')::timestamptz, NULLIF($12, '')::timestamptz, NOW(), NOW())
ON CONFLICT (payment_order_id) DO NOTHING`,
		orderID, input.UserID, decimal.NewFromFloat(input.PaymentAmount).Round(2).StringFixed(2), decimal.NewFromFloat(input.PayAmount).Round(2).StringFixed(2),
		baseCredit.StringFixed(2), bonusCredit.StringFixed(2), finalCredit.StringFixed(2),
		result.Snapshot["bonus_rate"], result.Snapshot["min_amount"], result.Snapshot["max_bonus"],
		result.Snapshot["campaign_started_at"], result.Snapshot["campaign_ended_at"])
	return err
}

func (c *Calculator) ResolveAffiliateRebateBase(ctx context.Context, order *dbent.PaymentOrder) (float64, error) {
	if order == nil || c == nil || c.db == nil {
		if order == nil {
			return 0, fmt.Errorf("payment order is required")
		}
		return order.Amount, nil
	}
	var base float64
	err := c.db.QueryRowContext(ctx, `SELECT base_credit FROM custom_recharge_promotion_orders WHERE payment_order_id = $1`, order.ID).Scan(&base)
	if err == sql.ErrNoRows {
		return order.Amount, nil
	}
	if err != nil {
		return 0, err
	}
	return base, nil
}

func finiteNonnegative(value float64) bool {
	return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}
