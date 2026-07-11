package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

const rewardDeliveryColumns = `id, source_type, source_id, user_id, prize_item_id,
	reward_snapshot, reward_type, reward_value, reward_detail, rule_version,
	idempotency_key, status, attempts, last_error, next_retry_at, locked_at,
	delivered_at, compensated_at, created_at, updated_at`

type rewardDeliveryRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewRewardDeliveryRepository(client *dbent.Client, db *sql.DB) service.RewardDeliveryStore {
	return &rewardDeliveryRepository{client: client, db: db}
}

func (r *rewardDeliveryRepository) CreatePending(ctx context.Context, input service.CreateRewardDelivery) (*service.RewardDelivery, error) {
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.RewardType = strings.TrimSpace(input.RewardType)
	input.RuleVersion = strings.TrimSpace(input.RuleVersion)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.RewardValue = roundRewardDeliveryAmount(input.RewardValue)
	if input.SourceType == "" || input.SourceID <= 0 || input.UserID <= 0 ||
		input.RewardType == "" || input.RuleVersion == "" || input.IdempotencyKey == "" ||
		math.IsNaN(input.RewardValue) || math.IsInf(input.RewardValue, 0) || input.RewardValue < 0 {
		return nil, fmt.Errorf("invalid reward delivery input")
	}
	snapshot := input.RewardSnapshot
	if len(snapshot) == 0 {
		snapshot = json.RawMessage(`{}`)
	}
	var snapshotObject map[string]any
	if err := json.Unmarshal(snapshot, &snapshotObject); err != nil || snapshotObject == nil {
		return nil, fmt.Errorf("reward snapshot must be a JSON object")
	}

	query := `INSERT INTO reward_deliveries (
		source_type, source_id, user_id, prize_item_id, reward_snapshot,
		reward_type, reward_value, reward_detail, rule_version, idempotency_key
	) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10)
	ON CONFLICT DO NOTHING
	RETURNING ` + rewardDeliveryColumns

	exec := txAwareSQLExecutor(ctx, r.db, r.client)
	if exec == nil {
		return nil, fmt.Errorf("create pending reward delivery: SQL executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, query,
		input.SourceType, input.SourceID, input.UserID, input.PrizeItemID,
		[]byte(snapshot), input.RewardType, input.RewardValue,
		input.RewardDetail, input.RuleVersion, input.IdempotencyKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create pending reward delivery: %w", err)
	}
	if rows.Next() {
		delivery, scanErr := scanRewardDelivery(rows)
		closeErr := rows.Close()
		if scanErr != nil {
			return nil, fmt.Errorf("create pending reward delivery: %w", scanErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("create pending reward delivery: %w", closeErr)
		}
		return delivery, nil
	}
	rowsErr := rows.Err()
	_ = rows.Close()
	if rowsErr != nil {
		return nil, fmt.Errorf("create pending reward delivery: %w", rowsErr)
	}

	// Use a new statement after a conflict. PostgreSQL's statement snapshot may
	// predate the concurrent winner that caused ON CONFLICT DO NOTHING.
	existing, err := getRewardDeliveryByIdempotencyKey(ctx, exec, input.IdempotencyKey)
	if err == nil {
		if rewardDeliveryMatchesCreate(existing, input, snapshotObject) {
			return existing, nil
		}
		return nil, service.ErrRewardDeliveryIdempotencyConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	existing, err = getRewardDeliveryBySource(ctx, exec, input.SourceType, input.SourceID)
	if err == nil {
		if rewardDeliveryMatchesCreate(existing, input, snapshotObject) {
			return existing, nil
		}
		return nil, service.ErrRewardDeliveryIdempotencyConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return nil, fmt.Errorf("create pending reward delivery: %w", sql.ErrNoRows)
}

func (r *rewardDeliveryRepository) ClaimDue(ctx context.Context, now time.Time, limit int) ([]service.RewardDelivery, error) {
	limit = normalizeRewardDeliveryLimit(limit)
	query := `WITH due AS (
	SELECT id
	FROM reward_deliveries
	WHERE status = $1
		AND (next_retry_at IS NULL OR next_retry_at <= $2)
	ORDER BY COALESCE(next_retry_at, created_at) ASC, id ASC
	LIMIT $3
	FOR UPDATE SKIP LOCKED
)
UPDATE reward_deliveries AS delivery
SET status = $4,
	attempts = delivery.attempts + 1,
	locked_at = $2,
	last_error = NULL,
	updated_at = $2
FROM due
WHERE delivery.id = due.id
RETURNING ` + prefixedRewardDeliveryColumns("delivery")

	rows, err := r.db.QueryContext(ctx, query,
		service.RewardDeliveryStatusPending, now, limit, service.RewardDeliveryStatusDelivering)
	if err != nil {
		return nil, fmt.Errorf("claim due reward deliveries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanRewardDeliveries(rows, "claim due reward deliveries")
}

func (r *rewardDeliveryRepository) ProcessClaimed(ctx context.Context, id int64, apply service.RewardDeliveryApply) error {
	if r.client == nil {
		return fmt.Errorf("process claimed reward delivery: ent client is not configured")
	}
	if apply == nil {
		return fmt.Errorf("process claimed reward delivery: apply callback is required")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("process claimed reward delivery: begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	txCtx := dbent.NewTxContext(ctx, tx)
	rows, err := tx.Client().QueryContext(txCtx, `SELECT `+rewardDeliveryColumns+`
FROM reward_deliveries WHERE id = $1 AND status = $2 FOR UPDATE`,
		id, service.RewardDeliveryStatusDelivering)
	if err != nil {
		return fmt.Errorf("process claimed reward delivery: lock delivery: %w", err)
	}
	if !rows.Next() {
		rowsErr := rows.Err()
		_ = rows.Close()
		if rowsErr != nil {
			return fmt.Errorf("process claimed reward delivery: lock delivery: %w", rowsErr)
		}
		return service.ErrRewardDeliveryStateConflict
	}
	delivery, err := scanRewardDelivery(rows)
	closeErr := rows.Close()
	if err != nil {
		return fmt.Errorf("process claimed reward delivery: scan delivery: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("process claimed reward delivery: close delivery rows: %w", closeErr)
	}

	detail, err := apply(txCtx, *delivery)
	if err != nil {
		return fmt.Errorf("process claimed reward delivery: apply reward: %w", err)
	}
	at := time.Now().UTC()
	result, err := tx.Client().ExecContext(txCtx, `UPDATE reward_deliveries
SET status = $1, reward_detail = $2, delivered_at = $3, locked_at = NULL,
	next_retry_at = NULL, last_error = NULL, updated_at = $3
WHERE id = $4 AND status = $5`,
		service.RewardDeliveryStatusDelivered, detail, at, id, service.RewardDeliveryStatusDelivering)
	if err := rewardDeliveryTransitionResult(result, err, "mark reward delivery delivered"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("process claimed reward delivery: commit transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *rewardDeliveryRepository) MarkFailed(ctx context.Context, id int64, lastError string, nextRetryAt *time.Time) error {
	status := service.RewardDeliveryStatusFailed
	if nextRetryAt != nil {
		status = service.RewardDeliveryStatusPending
	}
	result, err := r.db.ExecContext(ctx, `UPDATE reward_deliveries
SET status = $1, last_error = $2, next_retry_at = $3, locked_at = NULL, updated_at = NOW()
WHERE id = $4 AND status = $5`,
		status, lastError, nextRetryAt, id, service.RewardDeliveryStatusDelivering)
	return rewardDeliveryTransitionResult(result, err, "mark reward delivery failed")
}

func (r *rewardDeliveryRepository) RecoverStale(ctx context.Context, staleBefore, nextRetryAt time.Time) (int, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE reward_deliveries
SET status = $1,
	last_error = CASE
		WHEN last_error IS NULL OR last_error = '' THEN $2
		ELSE last_error || E'\n' || $2
	END,
	next_retry_at = $3, locked_at = NULL, updated_at = NOW()
WHERE status = $4 AND locked_at IS NOT NULL AND locked_at < $5`,
		service.RewardDeliveryStatusPending, "delivery lock expired and was recovered", nextRetryAt,
		service.RewardDeliveryStatusDelivering, staleBefore)
	if err != nil {
		return 0, fmt.Errorf("recover stale reward deliveries: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("recover stale reward deliveries: %w", err)
	}
	return int(count), nil
}

func (r *rewardDeliveryRepository) GetByID(ctx context.Context, id int64) (*service.RewardDelivery, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+rewardDeliveryColumns+`
FROM reward_deliveries WHERE id = $1 LIMIT 1`, id)
	if err != nil {
		return nil, fmt.Errorf("get reward delivery: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("get reward delivery: %w", err)
		}
		return nil, sql.ErrNoRows
	}
	delivery, err := scanRewardDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("get reward delivery: %w", err)
	}
	return delivery, nil
}

func getRewardDeliveryByIdempotencyKey(ctx context.Context, exec sqlQueryExecutor, key string) (*service.RewardDelivery, error) {
	rows, err := exec.QueryContext(ctx, `SELECT `+rewardDeliveryColumns+`
FROM reward_deliveries WHERE idempotency_key = $1 LIMIT 1`, key)
	if err != nil {
		return nil, fmt.Errorf("get idempotent reward delivery: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("get idempotent reward delivery: %w", err)
		}
		return nil, fmt.Errorf("get idempotent reward delivery: %w", sql.ErrNoRows)
	}
	delivery, err := scanRewardDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("get idempotent reward delivery: %w", err)
	}
	return delivery, nil
}

func getRewardDeliveryBySource(ctx context.Context, exec sqlQueryExecutor, sourceType string, sourceID int64) (*service.RewardDelivery, error) {
	rows, err := exec.QueryContext(ctx, `SELECT `+rewardDeliveryColumns+`
FROM reward_deliveries WHERE source_type = $1 AND source_id = $2 LIMIT 1`, sourceType, sourceID)
	if err != nil {
		return nil, fmt.Errorf("get source reward delivery: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("get source reward delivery: %w", err)
		}
		return nil, fmt.Errorf("get source reward delivery: %w", sql.ErrNoRows)
	}
	delivery, err := scanRewardDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("get source reward delivery: %w", err)
	}
	return delivery, nil
}

func rewardDeliveryMatchesCreate(existing *service.RewardDelivery, input service.CreateRewardDelivery, snapshot map[string]any) bool {
	if existing == nil || existing.SourceType != input.SourceType || existing.SourceID != input.SourceID ||
		existing.UserID != input.UserID || !equalOptionalInt64(existing.PrizeItemID, input.PrizeItemID) ||
		existing.RewardType != input.RewardType || roundRewardDeliveryAmount(existing.RewardValue) != input.RewardValue ||
		existing.RuleVersion != input.RuleVersion || existing.IdempotencyKey != input.IdempotencyKey {
		return false
	}
	var existingSnapshot map[string]any
	if err := json.Unmarshal(existing.RewardSnapshot, &existingSnapshot); err != nil || existingSnapshot == nil {
		return false
	}
	return reflect.DeepEqual(existingSnapshot, snapshot)
}

func equalOptionalInt64(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (r *rewardDeliveryRepository) List(ctx context.Context, filter service.RewardDeliveryFilter) ([]service.RewardDelivery, int64, error) {
	page, pageSize := normalizeRewardDeliveryPage(filter.Page, filter.PageSize)
	where, args := rewardDeliveryFilterWhere(filter)

	var total int64
	countRows, err := r.db.QueryContext(ctx, `SELECT COUNT(*) FROM reward_deliveries`+where, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count reward deliveries: %w", err)
	}
	if countRows.Next() {
		err = countRows.Scan(&total)
	} else if rowsErr := countRows.Err(); rowsErr != nil {
		err = rowsErr
	} else {
		err = sql.ErrNoRows
	}
	_ = countRows.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("count reward deliveries: %w", err)
	}
	if total == 0 {
		return []service.RewardDelivery{}, 0, nil
	}

	limitPlaceholder := len(args) + 1
	query := fmt.Sprintf(`SELECT %s FROM reward_deliveries%s
ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d`,
		rewardDeliveryColumns, where, limitPlaceholder, limitPlaceholder+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list reward deliveries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	deliveries, err := scanRewardDeliveries(rows, "list reward deliveries")
	return deliveries, total, err
}

func scanRewardDeliveries(rows *sql.Rows, operation string) ([]service.RewardDelivery, error) {
	deliveries := make([]service.RewardDelivery, 0)
	for rows.Next() {
		delivery, err := scanRewardDelivery(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", operation, err)
		}
		deliveries = append(deliveries, *delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", operation, err)
	}
	return deliveries, nil
}

type rewardDeliveryScanner interface {
	Scan(dest ...any) error
}

func scanRewardDelivery(scanner rewardDeliveryScanner) (*service.RewardDelivery, error) {
	var delivery service.RewardDelivery
	var snapshot []byte
	var prizeItemID sql.NullInt64
	var lastError sql.NullString
	var nextRetryAt, lockedAt, deliveredAt, compensatedAt sql.NullTime
	if err := scanner.Scan(
		&delivery.ID, &delivery.SourceType, &delivery.SourceID, &delivery.UserID, &prizeItemID,
		&snapshot, &delivery.RewardType, &delivery.RewardValue, &delivery.RewardDetail, &delivery.RuleVersion,
		&delivery.IdempotencyKey, &delivery.Status, &delivery.Attempts, &lastError, &nextRetryAt, &lockedAt,
		&deliveredAt, &compensatedAt, &delivery.CreatedAt, &delivery.UpdatedAt,
	); err != nil {
		return nil, err
	}
	delivery.RewardValue = roundRewardDeliveryAmount(delivery.RewardValue)
	delivery.RewardSnapshot = append(json.RawMessage(nil), snapshot...)
	if prizeItemID.Valid {
		delivery.PrizeItemID = &prizeItemID.Int64
	}
	if lastError.Valid {
		delivery.LastError = &lastError.String
	}
	if nextRetryAt.Valid {
		delivery.NextRetryAt = &nextRetryAt.Time
	}
	if lockedAt.Valid {
		delivery.LockedAt = &lockedAt.Time
	}
	if deliveredAt.Valid {
		delivery.DeliveredAt = &deliveredAt.Time
	}
	if compensatedAt.Valid {
		delivery.CompensatedAt = &compensatedAt.Time
	}
	return &delivery, nil
}

func rewardDeliveryTransitionResult(result sql.Result, err error, operation string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if rows != 1 {
		return service.ErrRewardDeliveryStateConflict
	}
	return nil
}

func rewardDeliveryFilterWhere(filter service.RewardDeliveryFilter) (string, []any) {
	conditions := make([]string, 0, 3)
	args := make([]any, 0, 3)
	add := func(column string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		add("status", status)
	}
	if sourceType := strings.TrimSpace(filter.SourceType); sourceType != "" {
		add("source_type", sourceType)
	}
	if filter.UserID != nil {
		add("user_id", *filter.UserID)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func normalizeRewardDeliveryLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func normalizeRewardDeliveryPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	pageSize = normalizeRewardDeliveryLimit(pageSize)
	return page, pageSize
}

func prefixedRewardDeliveryColumns(prefix string) string {
	parts := strings.Split(rewardDeliveryColumns, ",")
	for i := range parts {
		parts[i] = prefix + "." + strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, ", ")
}

func roundRewardDeliveryAmount(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

var _ service.RewardDeliveryStore = (*rewardDeliveryRepository)(nil)
