package accountdrain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

func (r *Repository) Create(ctx context.Context, input CreatePlanInput) (*Plan, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("account drain repository is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin account drain plan: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	plan := &Plan{Name: input.Name, Status: StatusActive, ExpiresAt: input.ExpiresAt}
	err = tx.QueryRowContext(ctx, `
INSERT INTO custom_account_drain_plans (name, status, expires_at)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at`, input.Name, StatusActive, input.ExpiresAt).Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create account drain plan: %w", err)
	}
	for _, accountID := range input.AccountIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO custom_account_drain_plan_accounts (plan_id, account_id) VALUES ($1, $2)`, plan.ID, accountID); err != nil {
			return nil, fmt.Errorf("add account %d to account drain plan: %w", accountID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit account drain plan: %w", err)
	}
	plan.AccountIDs = append([]int64(nil), input.AccountIDs...)
	return plan, nil
}

func (r *Repository) Stop(ctx context.Context, id int64) error {
	if r == nil || r.db == nil {
		return errors.New("account drain repository is unavailable")
	}
	result, err := r.db.ExecContext(ctx, `UPDATE custom_account_drain_plans SET status = $2, updated_at = NOW() WHERE id = $1 AND status = $3`, id, StatusStopped, StatusActive)
	if err != nil {
		return fmt.Errorf("stop account drain plan: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read account drain stop result: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
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

func normalizeCreateInput(input CreatePlanInput) (CreatePlanInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return input, errors.New("plan name is required")
	}
	if len(input.AccountIDs) == 0 {
		return input, errors.New("at least one target account is required")
	}
	seen := make(map[int64]struct{}, len(input.AccountIDs))
	unique := make([]int64, 0, len(input.AccountIDs))
	for _, accountID := range input.AccountIDs {
		if accountID <= 0 {
			return input, errors.New("account IDs must be positive")
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		unique = append(unique, accountID)
	}
	input.AccountIDs = unique
	if input.ExpiresAt != nil && !input.ExpiresAt.After(time.Now().UTC()) {
		return input, errors.New("expiry must be in the future")
	}
	return input, nil
}
