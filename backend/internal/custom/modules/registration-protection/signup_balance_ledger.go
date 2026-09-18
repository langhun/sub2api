package registrationprotection

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

const signupBalanceLedgerIDOffset int64 = -1_000_000_000_000_000

// SignupBalanceLedger owns the immutable registration-grant audit table.
// It has one row per user so retries cannot create duplicate grant records.
type SignupBalanceLedger struct {
	db *sql.DB
}

func NewSignupBalanceLedger(db *sql.DB) *SignupBalanceLedger {
	return &SignupBalanceLedger{db: db}
}

func (l *SignupBalanceLedger) RecordSignupBalance(ctx context.Context, userID int64, amount float64, signupSource string) error {
	if l == nil || l.db == nil || userID <= 0 || amount <= 0 || math.IsNaN(amount) || math.IsInf(amount, 0) {
		return fmt.Errorf("invalid signup balance ledger entry")
	}
	source := normalizeSignupSource(signupSource)
	var id int64
	err := l.db.QueryRowContext(ctx, `
		INSERT INTO custom_signup_balance_ledger (user_id, amount, signup_source)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE
		SET user_id = EXCLUDED.user_id
		WHERE custom_signup_balance_ledger.amount = EXCLUDED.amount
		  AND custom_signup_balance_ledger.signup_source = EXCLUDED.signup_source
		RETURNING id`, userID, amount, source).Scan(&id)
	if err == sql.ErrNoRows {
		return fmt.Errorf("conflicting signup balance ledger entry for user %d", userID)
	}
	if err != nil {
		return fmt.Errorf("record signup balance ledger: %w", err)
	}
	return nil
}

func (l *SignupBalanceLedger) DeleteSignupBalance(ctx context.Context, userID int64, amount float64, signupSource string) error {
	if l == nil || l.db == nil || userID <= 0 || amount <= 0 {
		return fmt.Errorf("invalid signup balance ledger cleanup")
	}
	result, err := l.db.ExecContext(ctx, `
		DELETE FROM custom_signup_balance_ledger
		WHERE user_id = $1 AND amount = $2 AND signup_source = $3`, userID, amount, normalizeSignupSource(signupSource))
	if err != nil {
		return fmt.Errorf("delete signup balance ledger: %w", err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("read signup balance ledger cleanup result: %w", err)
	} else if affected != 1 {
		return fmt.Errorf("signup balance ledger cleanup affected %d rows", affected)
	}
	return nil
}

func (l *SignupBalanceLedger) ListSignupBalanceHistory(ctx context.Context, userID int64, page, pageSize int) ([]service.RedeemCode, int64, error) {
	if l == nil || l.db == nil || userID <= 0 {
		return nil, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 15
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := l.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM custom_signup_balance_ledger WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count signup balance ledger: %w", err)
	}
	rows, err := l.db.QueryContext(ctx, `
		SELECT id, amount::double precision, signup_source, created_at
		FROM custom_signup_balance_ledger
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC
		OFFSET $2 LIMIT $3`, userID, offset, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list signup balance ledger: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]service.RedeemCode, 0)
	for rows.Next() {
		var id int64
		var amount float64
		var source string
		var createdAt time.Time
		if err := rows.Scan(&id, &amount, &source, &createdAt); err != nil {
			return nil, 0, fmt.Errorf("scan signup balance ledger: %w", err)
		}
		usedBy := userID
		usedAt := createdAt
		items = append(items, service.RedeemCode{
			ID:        signupBalanceLedgerIDOffset - id,
			Code:      fmt.Sprintf("SIGNUP-%d", id),
			Type:      service.SignupBalanceHistoryType,
			Value:     amount,
			Status:    service.StatusUsed,
			UsedBy:    &usedBy,
			UsedAt:    &usedAt,
			CreatedAt: createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate signup balance ledger: %w", err)
	}
	return items, total, nil
}

func normalizeSignupSource(source string) string {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		return "email"
	}
	if len(source) > 32 {
		return source[:32]
	}
	return source
}

var _ service.SignupBalanceLedger = (*SignupBalanceLedger)(nil)
var _ service.SignupBalanceHistoryProvider = (*SignupBalanceLedger)(nil)
