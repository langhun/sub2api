package accountdrain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) List(ctx context.Context) ([]Plan, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("account drain repository is unavailable")
	}
	const query = `
SELECT p.id, p.name, p.status, p.expires_at, p.created_at, p.updated_at,
       COALESCE(json_agg(pa.account_id ORDER BY pa.account_id) FILTER (WHERE pa.account_id IS NOT NULL), '[]'::json)
FROM custom_account_drain_plans p
LEFT JOIN custom_account_drain_plan_accounts pa ON pa.plan_id = p.id
GROUP BY p.id
ORDER BY p.created_at DESC, p.id DESC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list account drain plans: %w", err)
	}
	defer rows.Close()

	plans := make([]Plan, 0)
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate account drain plans: %w", err)
	}
	return plans, nil
}

func (r *Repository) IsAccountTargeted(ctx context.Context, accountID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("account drain repository is unavailable")
	}
	var targeted bool
	err := r.db.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM custom_account_drain_plan_accounts pa
    JOIN custom_account_drain_plans p ON p.id = pa.plan_id
    WHERE pa.account_id = $1
      AND p.status = $2
      AND (p.expires_at IS NULL OR p.expires_at > NOW())
)`, accountID, StatusActive).Scan(&targeted)
	if err != nil {
		return false, fmt.Errorf("check account drain target: %w", err)
	}
	return targeted, nil
}

// EnableAccount creates a private, single-account plan when the account is not
// already targeted. The advisory lock makes repeated menu clicks idempotent.
func (r *Repository) EnableAccount(ctx context.Context, accountID int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, errors.New("account drain repository is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin account drain target: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, accountID); err != nil {
		return false, fmt.Errorf("lock account drain target: %w", err)
	}
	var platform string
	if err := tx.QueryRowContext(ctx, `SELECT platform FROM accounts WHERE id = $1`, accountID).Scan(&platform); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, err
		}
		return false, fmt.Errorf("load account drain target: %w", err)
	}
	if platform != "openai" {
		return false, fmt.Errorf("only OpenAI accounts support directed consumption")
	}
	var alreadyActive bool
	if err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM custom_account_drain_plan_accounts pa
    JOIN custom_account_drain_plans p ON p.id = pa.plan_id
    WHERE pa.account_id = $1
      AND p.status = $2
      AND (p.expires_at IS NULL OR p.expires_at > NOW())
)`, accountID, StatusActive).Scan(&alreadyActive); err != nil {
		return false, fmt.Errorf("check account drain target: %w", err)
	}
	if alreadyActive {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit account drain target: %w", err)
		}
		return false, nil
	}

	var planID int64
	if err := tx.QueryRowContext(ctx, `
INSERT INTO custom_account_drain_plans (name, status)
VALUES ($1, $2)
RETURNING id`, fmt.Sprintf("account-target-%d", accountID), StatusActive).Scan(&planID); err != nil {
		return false, fmt.Errorf("create account drain target: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO custom_account_drain_plan_accounts (plan_id, account_id)
VALUES ($1, $2)`, planID, accountID); err != nil {
		return false, fmt.Errorf("add account drain target: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit account drain target: %w", err)
	}
	return true, nil
}

// DisableAccount removes the account from every active plan. Plans made empty
// by that removal are stopped, while other accounts in a multi-account plan
// keep their directed-consumption setting.
func (r *Repository) DisableAccount(ctx context.Context, accountID int64) error {
	if r == nil || r.db == nil {
		return errors.New("account drain repository is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin account drain removal: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, accountID); err != nil {
		return fmt.Errorf("lock account drain target: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
WITH removed AS (
    DELETE FROM custom_account_drain_plan_accounts pa
    USING custom_account_drain_plans p
    WHERE pa.plan_id = p.id AND pa.account_id = $1 AND p.status = $2
    RETURNING pa.plan_id
)
UPDATE custom_account_drain_plans p
SET status = $3, updated_at = NOW()
WHERE p.id IN (SELECT plan_id FROM removed)
  AND p.status = $2
  AND NOT EXISTS (
      SELECT 1 FROM custom_account_drain_plan_accounts pa WHERE pa.plan_id = p.id
  )`, accountID, StatusActive, StatusStopped); err != nil {
		return fmt.Errorf("remove account drain target: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit account drain removal: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlan(row rowScanner) (Plan, error) {
	var plan Plan
	var accountIDsJSON []byte
	if err := row.Scan(&plan.ID, &plan.Name, &plan.Status, &plan.ExpiresAt, &plan.CreatedAt, &plan.UpdatedAt, &accountIDsJSON); err != nil {
		return Plan{}, fmt.Errorf("scan account drain plan: %w", err)
	}
	if err := json.Unmarshal(accountIDsJSON, &plan.AccountIDs); err != nil {
		return Plan{}, fmt.Errorf("decode account drain plan accounts: %w", err)
	}
	if plan.Status == StatusActive && plan.ExpiresAt != nil && !plan.ExpiresAt.After(time.Now().UTC()) {
		plan.Status = StatusExpired
	}
	return plan, nil
}
