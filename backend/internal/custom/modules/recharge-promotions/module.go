// Package rechargepromotions implements the isolated recharge bonus Overlay.
package rechargepromotions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/recharge-promotions/settings"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

const moduleRoutePrefix = "/recharge-promotions"

type settingsReader interface {
	ReadRechargePromotions(context.Context) (settings.Config, error)
}

type Module struct {
	settings settingsReader
	db       *sql.DB
}

func NewModule(config settingsReader, db *sql.DB) *Module {
	return &Module{settings: config, db: db}
}

type calculation struct {
	Mode         string
	BaseCredit   decimal.Decimal
	BonusCredit  decimal.Decimal
	FinalCredit  decimal.Decimal
	Config       settings.Config
	Eligible     bool
	IneligibleBy string
}

func (m *Module) PrepareBalanceOrder(ctx context.Context, input service.BalanceOrderExtensionInput) (service.BalanceOrderExtensionResult, error) {
	calculated, err := m.calculate(ctx, input.PaymentAmount, input.BaseCredit, input.Now)
	if err != nil {
		return service.BalanceOrderExtensionResult{}, err
	}
	return service.BalanceOrderExtensionResult{
		CreditedAmount: calculated.FinalCredit.InexactFloat64(),
		Snapshot:       calculated,
	}, nil
}

func (m *Module) PersistBalanceOrder(ctx context.Context, client *dbent.Client, order *dbent.PaymentOrder, input service.BalanceOrderExtensionInput, result service.BalanceOrderExtensionResult) error {
	if m == nil || m.db == nil || client == nil || order == nil {
		return fmt.Errorf("recharge promotion transaction dependencies are required")
	}
	calculated, ok := result.Snapshot.(calculation)
	if !ok || !calculated.Eligible || calculated.BonusCredit.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	tierRules, err := json.Marshal(settingsTierRules(calculated.Config.TierRules))
	if err != nil {
		return fmt.Errorf("encode promotion tier rules: %w", err)
	}
	_, err = client.ExecContext(ctx, `
INSERT INTO custom_recharge_promotion_orders
    (payment_order_id, user_id, mode, payment_amount, pay_amount, base_credit, bonus_credit, final_credit,
     percentage_rate, minimum_amount, maximum_bonus, tier_rules, campaign_started_at, campaign_ended_at, created_at, updated_at)
VALUES
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, NULLIF($13, '')::timestamptz, NULLIF($14, '')::timestamptz, NOW(), NOW())`,
		order.ID,
		input.UserID,
		calculated.Mode,
		decimalAmount(input.PaymentAmount),
		decimalAmount(input.PayAmount),
		calculated.BaseCredit.StringFixed(2),
		calculated.BonusCredit.StringFixed(2),
		calculated.FinalCredit.StringFixed(2),
		decimalAmount(calculated.Config.PercentageRate),
		decimalAmount(calculated.Config.MinimumAmount),
		decimalAmount(calculated.Config.MaximumBonus),
		string(tierRules),
		settings.FormatTime(calculated.Config.StartAt),
		settings.FormatTime(calculated.Config.EndAt),
	)
	if err != nil {
		return fmt.Errorf("insert recharge promotion snapshot: %w", err)
	}
	return nil
}

func (m *Module) ResolveAffiliateRebateBase(ctx context.Context, order *dbent.PaymentOrder) (float64, bool, error) {
	if m == nil || m.db == nil || order == nil {
		return 0, false, nil
	}
	var baseCredit string
	err := m.db.QueryRowContext(ctx, `SELECT base_credit::text FROM custom_recharge_promotion_orders WHERE payment_order_id = $1`, order.ID).Scan(&baseCredit)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read recharge promotion rebate base: %w", err)
	}
	parsed, err := decimal.NewFromString(baseCredit)
	if err != nil {
		return 0, false, fmt.Errorf("parse recharge promotion rebate base: %w", err)
	}
	return parsed.InexactFloat64(), true, nil
}

func (m *Module) RecordBalanceRefund(ctx context.Context, client *dbent.Client, input service.BalanceRefundExtensionInput) error {
	if m == nil || client == nil || input.Order == nil {
		return nil
	}
	driver, ok := client.Driver().(interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	})
	if !ok {
		return fmt.Errorf("recharge promotion transaction driver does not support raw queries")
	}
	rows, err := driver.QueryContext(ctx, `
SELECT base_credit::text, bonus_credit::text, final_credit::text
FROM custom_recharge_promotion_orders WHERE payment_order_id = $1 FOR UPDATE`, input.Order.ID)
	if err != nil {
		return fmt.Errorf("lock recharge promotion refund snapshot: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return fmt.Errorf("lock recharge promotion refund snapshot: %w", err)
		}
		return nil
	}
	var baseText, bonusText, finalText string
	if err := rows.Scan(&baseText, &bonusText, &finalText); err != nil {
		_ = rows.Close()
		return fmt.Errorf("scan recharge promotion refund snapshot: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close recharge promotion refund snapshot: %w", err)
	}
	base, err := decimal.NewFromString(baseText)
	if err != nil {
		return err
	}
	bonus, err := decimal.NewFromString(bonusText)
	if err != nil {
		return err
	}
	final, err := decimal.NewFromString(finalText)
	if err != nil {
		return err
	}
	refund := decimal.NewFromFloat(input.RefundAmount).Round(2)
	if refund.GreaterThan(final) {
		refund = final
	}
	refundedBase := proportionalCredit(base, final, refund)
	refundedBonus := refund.Sub(refundedBase)
	if refundedBonus.GreaterThan(bonus) {
		refundedBonus = bonus
		refundedBase = refund.Sub(refundedBonus)
	}
	if refundedBase.GreaterThan(base) {
		refundedBase = base
		refundedBonus = refund.Sub(refundedBase)
	}
	_, err = client.ExecContext(ctx, `
UPDATE custom_recharge_promotion_orders
SET refunded_credit = $2, refunded_base_credit = $3, refunded_bonus_credit = $4, refunded_at = $5, updated_at = NOW()
WHERE payment_order_id = $1`,
		input.Order.ID,
		refund.StringFixed(2),
		refundedBase.StringFixed(2),
		refundedBonus.StringFixed(2),
		input.RefundedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("update recharge promotion refund snapshot: %w", err)
	}
	return nil
}

func proportionalCredit(part, total, amount decimal.Decimal) decimal.Decimal {
	if total.LessThanOrEqual(decimal.Zero) || amount.LessThanOrEqual(decimal.Zero) || part.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}
	return part.Mul(amount).Div(total).Round(2)
}

func (m *Module) calculate(ctx context.Context, paymentAmount, baseCredit float64, now time.Time) (calculation, error) {
	if !finite(paymentAmount) || !finite(baseCredit) {
		return calculation{}, fmt.Errorf("recharge promotion input contains invalid amount")
	}
	if m == nil || m.settings == nil {
		return calculation{}, fmt.Errorf("recharge promotion settings are required")
	}
	config, err := m.settings.ReadRechargePromotions(ctx)
	if err != nil {
		return calculation{}, err
	}
	base := decimal.NewFromFloat(baseCredit).Round(2)
	result := calculation{Mode: config.Mode, BaseCredit: base, FinalCredit: base, Config: config}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if !config.Enabled {
		result.IneligibleBy = "disabled"
		return result, nil
	}
	if !config.StartAt.IsZero() && now.Before(config.StartAt) {
		result.IneligibleBy = "not_started"
		return result, nil
	}
	if !config.EndAt.IsZero() && !now.Before(config.EndAt) {
		result.IneligibleBy = "ended"
		return result, nil
	}
	payment := decimal.NewFromFloat(paymentAmount).Round(2)
	bonus := decimal.Zero
	switch config.Mode {
	case settings.ModeTiered:
		for _, tier := range config.TierRules {
			if payment.GreaterThanOrEqual(decimal.NewFromFloat(tier.MinimumAmount).Round(2)) {
				candidate := decimal.NewFromFloat(tier.BonusCredit).Round(2)
				if candidate.GreaterThan(bonus) {
					bonus = candidate
				}
			}
		}
		if bonus.IsZero() {
			result.IneligibleBy = "minimum_amount"
			return result, nil
		}
	case settings.ModePercentage:
		minimum := decimal.NewFromFloat(config.MinimumAmount).Round(2)
		if payment.LessThan(minimum) {
			result.IneligibleBy = "minimum_amount"
			return result, nil
		}
		bonus = base.Mul(decimal.NewFromFloat(config.PercentageRate)).Div(decimal.NewFromInt(100)).Round(2)
		if config.MaximumBonus > 0 {
			cap := decimal.NewFromFloat(config.MaximumBonus).Round(2)
			if bonus.GreaterThan(cap) {
				bonus = cap
			}
		}
	default:
		return calculation{}, fmt.Errorf("unsupported recharge promotion mode %q", config.Mode)
	}
	if bonus.LessThanOrEqual(decimal.Zero) {
		return result, nil
	}
	result.Eligible = true
	result.BonusCredit = bonus
	result.FinalCredit = base.Add(bonus).Round(2)
	return result, nil
}

type previewRequest struct {
	Amount     float64 `json:"amount" binding:"required"`
	BaseCredit float64 `json:"base_credit" binding:"required"`
}

func (m *Module) RegisterRoutes(admin *gin.RouterGroup, preview *gin.RouterGroup, authenticated *gin.RouterGroup) {
	if m == nil {
		return
	}
	if preview != nil {
		preview.POST(moduleRoutePrefix+"/preview", m.preview)
	}
	if authenticated != nil {
		authenticated.GET(moduleRoutePrefix+"/orders/:id", m.getUserOrder)
	}
	if admin != nil {
		admin.GET(moduleRoutePrefix+"/summary", m.summary)
		admin.GET(moduleRoutePrefix+"/orders", m.listOrders)
		admin.GET(moduleRoutePrefix+"/orders/:id", m.getAdminOrder)
	}
}

func (m *Module) preview(c *gin.Context) {
	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 || math.IsNaN(req.Amount) || math.IsInf(req.Amount, 0) {
		response.BadRequest(c, "amount must be a positive number")
		return
	}
	// The checkout adapter computes the core multiplier locally for display.
	// The order transaction always recalculates this amount server-side.
	if req.BaseCredit <= 0 || math.IsNaN(req.BaseCredit) || math.IsInf(req.BaseCredit, 0) {
		response.BadRequest(c, "valid recharge base credit is required")
		return
	}
	calculated, err := m.calculate(c.Request.Context(), req.Amount, req.BaseCredit, time.Now().UTC())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, previewFromCalculation(calculated))
}

func previewFromCalculation(calculated calculation) gin.H {
	return gin.H{
		"active":              calculated.Eligible,
		"mode":                calculated.Mode,
		"base_credit":         calculated.BaseCredit.InexactFloat64(),
		"bonus_credit":        calculated.BonusCredit.InexactFloat64(),
		"final_credit":        calculated.FinalCredit.InexactFloat64(),
		"ineligible_reason":   calculated.IneligibleBy,
		"minimum_amount":      calculated.minimumAmount(),
		"percentage_rate":     calculated.Config.PercentageRate,
		"maximum_bonus":       calculated.Config.MaximumBonus,
		"tier_rules":          settingsTierRules(calculated.Config.TierRules),
		"campaign_started_at": settings.FormatTime(calculated.Config.StartAt),
		"campaign_ended_at":   settings.FormatTime(calculated.Config.EndAt),
	}
}

func (c calculation) minimumAmount() float64 {
	if c.Mode != settings.ModeTiered || len(c.Config.TierRules) == 0 {
		return c.Config.MinimumAmount
	}
	return c.Config.TierRules[0].MinimumAmount
}

type promotionOrder struct {
	PaymentOrderID      int64     `json:"payment_order_id"`
	UserID              int64     `json:"user_id"`
	Mode                string    `json:"mode"`
	PaymentAmount       float64   `json:"payment_amount"`
	PayAmount           float64   `json:"pay_amount"`
	BaseCredit          float64   `json:"base_credit"`
	BonusCredit         float64   `json:"bonus_credit"`
	FinalCredit         float64   `json:"final_credit"`
	RefundedCredit      float64   `json:"refunded_credit"`
	RefundedBaseCredit  float64   `json:"refunded_base_credit"`
	RefundedBonusCredit float64   `json:"refunded_bonus_credit"`
	CreatedAt           time.Time `json:"created_at"`
}

func (m *Module) getUserOrder(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	orderID, ok := parseID(c.Param("id"))
	if !ok {
		response.BadRequest(c, "invalid order ID")
		return
	}
	order, found, err := m.readOrder(c.Request.Context(), orderID, &userID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !found {
		response.NotFound(c, "recharge promotion order not found")
		return
	}
	response.Success(c, order)
}

func (m *Module) getAdminOrder(c *gin.Context) {
	orderID, ok := parseID(c.Param("id"))
	if !ok {
		response.BadRequest(c, "invalid order ID")
		return
	}
	order, found, err := m.readOrder(c.Request.Context(), orderID, nil)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !found {
		response.NotFound(c, "recharge promotion order not found")
		return
	}
	response.Success(c, order)
}

func (m *Module) summary(c *gin.Context) {
	var totalOrders int64
	var totalBonus, totalFinal, totalRefunded float64
	err := m.db.QueryRowContext(c.Request.Context(), `
SELECT COUNT(*), COALESCE(SUM(bonus_credit), 0), COALESCE(SUM(final_credit), 0), COALESCE(SUM(refunded_bonus_credit), 0)
FROM custom_recharge_promotion_orders`).Scan(&totalOrders, &totalBonus, &totalFinal, &totalRefunded)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	config, err := m.settings.ReadRechargePromotions(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{
		"enabled": config.Enabled, "mode": config.Mode, "orders": int64(totalOrders),
		"bonus_credit": totalBonus, "final_credit": totalFinal, "refunded_bonus_credit": totalRefunded,
	})
}

func (m *Module) listOrders(c *gin.Context) {
	page, pageSize := pagination(c)
	offset := (page - 1) * pageSize
	args := []any{}
	where := ""
	if mode := strings.TrimSpace(c.Query("mode")); mode == settings.ModePercentage || mode == settings.ModeTiered {
		args = append(args, mode)
		where = " WHERE mode = $1"
	}
	var total int64
	if err := m.db.QueryRowContext(c.Request.Context(), "SELECT COUNT(*) FROM custom_recharge_promotion_orders"+where, args...).Scan(&total); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	args = append(args, pageSize, offset)
	limitParam, offsetParam := len(args)-1, len(args)
	rows, err := m.db.QueryContext(c.Request.Context(), fmt.Sprintf(`
SELECT payment_order_id, user_id, mode, payment_amount::float8, pay_amount::float8, base_credit::float8, bonus_credit::float8, final_credit::float8,
       refunded_credit::float8, refunded_base_credit::float8, refunded_bonus_credit::float8, created_at
FROM custom_recharge_promotion_orders%s ORDER BY created_at DESC, payment_order_id DESC LIMIT $%d OFFSET $%d`, where, limitParam, offsetParam), args...)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	defer func() { _ = rows.Close() }()
	items := []promotionOrder{}
	for rows.Next() {
		item, err := scanPromotionOrder(rows)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, total, page, pageSize)
}

func (m *Module) readOrder(ctx context.Context, paymentOrderID int64, userID *int64) (promotionOrder, bool, error) {
	if m == nil || m.db == nil {
		return promotionOrder{}, false, fmt.Errorf("recharge promotion database is required")
	}
	args := []any{paymentOrderID}
	where := "payment_order_id = $1"
	if userID != nil {
		args = append(args, *userID)
		where += " AND user_id = $2"
	}
	row := m.db.QueryRowContext(ctx, `SELECT payment_order_id, user_id, mode, payment_amount::float8, pay_amount::float8, base_credit::float8, bonus_credit::float8, final_credit::float8,
refunded_credit::float8, refunded_base_credit::float8, refunded_bonus_credit::float8, created_at FROM custom_recharge_promotion_orders WHERE `+where, args...)
	item, err := scanPromotionOrder(row)
	if err == sql.ErrNoRows {
		return promotionOrder{}, false, nil
	}
	if err != nil {
		return promotionOrder{}, false, err
	}
	return item, true, nil
}

type rowScanner interface{ Scan(...any) error }

func scanPromotionOrder(scanner rowScanner) (promotionOrder, error) {
	var item promotionOrder
	err := scanner.Scan(&item.PaymentOrderID, &item.UserID, &item.Mode, &item.PaymentAmount, &item.PayAmount, &item.BaseCredit, &item.BonusCredit, &item.FinalCredit, &item.RefundedCredit, &item.RefundedBaseCredit, &item.RefundedBonusCredit, &item.CreatedAt)
	return item, err
}

func settingsTierRules(rules []settings.TierRule) []settings.TierRule {
	if rules == nil {
		return []settings.TierRule{}
	}
	return append([]settings.TierRule(nil), rules...)
}

func decimalAmount(value float64) string { return decimal.NewFromFloat(value).Round(2).StringFixed(2) }
func finite(value float64) bool          { return value >= 0 && !math.IsNaN(value) && !math.IsInf(value, 0) }

func currentUserID(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	return subject.UserID, ok && subject.UserID > 0
}

func parseID(value string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return id, err == nil && id > 0
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
