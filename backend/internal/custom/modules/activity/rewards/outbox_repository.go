package rewards

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
)

const outboxColumns = `id, source_type, source_id, user_id, prize_item_id,
	reward_snapshot, reward_type, reward_value, reward_detail, rule_version,
	idempotency_key, status, attempts, last_error, next_retry_at, locked_at,
	delivered_at, compensated_at, created_at, updated_at`

// OutboxRepository is the module-owned implementation over the durable
// reward_deliveries outbox. It deliberately preserves the established SQL
// state machine, locking, idempotency, retry, and transaction semantics.
type OutboxRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewOutboxRepository(client *dbent.Client, db *sql.DB) *OutboxRepository {
	return &OutboxRepository{client: client, db: db}
}

func (r *OutboxRepository) Enqueue(ctx context.Context, input CreateDelivery) (*Delivery, error) {
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.RuleVersion = strings.TrimSpace(input.RuleVersion)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	input.RewardValue = roundDeliveryAmount(input.RewardValue)
	if err := input.Validate(); err != nil {
		return nil, err
	}
	snapshot := input.RewardSnapshot
	var snapshotObject map[string]any
	if err := json.Unmarshal(snapshot, &snapshotObject); err != nil || snapshotObject == nil {
		return nil, fmt.Errorf("reward snapshot must be a JSON object")
	}
	exec := r.executor(ctx)
	if exec == nil {
		return nil, fmt.Errorf("create pending reward delivery: SQL executor is not configured")
	}
	rows, err := exec.QueryContext(ctx, `INSERT INTO reward_deliveries (
		source_type, source_id, user_id, prize_item_id, reward_snapshot,
		reward_type, reward_value, reward_detail, rule_version, idempotency_key
	) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9, $10)
	ON CONFLICT DO NOTHING
	RETURNING `+outboxColumns,
		input.SourceType, input.SourceID, input.UserID, input.PrizeID, []byte(snapshot),
		string(input.RewardType), input.RewardValue, "", input.RuleVersion, input.IdempotencyKey,
	)
	if err != nil {
		return nil, fmt.Errorf("create pending reward delivery: %w", err)
	}
	if rows.Next() {
		delivery, scanErr := scanDelivery(rows)
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

	existing, err := r.getByIdempotencyKey(ctx, exec, input.IdempotencyKey)
	if err == nil {
		if deliveryMatchesCreate(existing, input, snapshotObject) {
			return existing, nil
		}
		return nil, ErrIdempotencyConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	existing, err = r.getBySource(ctx, exec, input.SourceType, input.SourceID)
	if err == nil {
		if deliveryMatchesCreate(existing, input, snapshotObject) {
			return existing, nil
		}
		return nil, ErrIdempotencyConflict
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return nil, fmt.Errorf("create pending reward delivery: %w", sql.ErrNoRows)
}

func (r *OutboxRepository) ClaimDue(ctx context.Context, now time.Time, limit int) ([]Delivery, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, ErrUnavailable
	}
	limit = normalizeDeliveryLimit(limit)
	rows, err := exec.QueryContext(ctx, `WITH due AS (
	SELECT id FROM reward_deliveries
	WHERE status = $1 AND (next_retry_at IS NULL OR next_retry_at <= $2)
	ORDER BY COALESCE(next_retry_at, created_at) ASC, id ASC
	LIMIT $3 FOR UPDATE SKIP LOCKED
)
UPDATE reward_deliveries AS delivery
SET status = $4, attempts = delivery.attempts + 1, locked_at = $2,
	last_error = NULL, updated_at = $2
FROM due
WHERE delivery.id = due.id
RETURNING `+prefixedOutboxColumns("delivery"),
		DeliveryStatusPending, now, limit, DeliveryStatusDelivering,
	)
	if err != nil {
		return nil, fmt.Errorf("claim due reward deliveries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanDeliveries(rows, "claim due reward deliveries")
}

func (r *OutboxRepository) ClaimByID(ctx context.Context, id int64, now time.Time) (*Delivery, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, ErrUnavailable
	}
	rows, err := exec.QueryContext(ctx, `UPDATE reward_deliveries
SET status = $1, attempts = attempts + 1, locked_at = $2,
	last_error = NULL, updated_at = $2
WHERE id = $3 AND status = $4 AND (next_retry_at IS NULL OR next_retry_at <= $2)
RETURNING `+outboxColumns, DeliveryStatusDelivering, now, id, DeliveryStatusPending)
	if err != nil {
		return nil, fmt.Errorf("claim reward delivery by id: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("claim reward delivery by id: %w", err)
		}
		return nil, nil
	}
	delivery, err := scanDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("claim reward delivery by id: %w", err)
	}
	return delivery, nil
}

func (r *OutboxRepository) ExecuteClaimed(ctx context.Context, id int64, apply DeliveryApply) error {
	if r == nil || r.client == nil {
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
	rows, err := tx.Client().QueryContext(txCtx, `SELECT `+outboxColumns+`
FROM reward_deliveries WHERE id = $1 AND status = $2 FOR UPDATE`, id, DeliveryStatusDelivering)
	if err != nil {
		return fmt.Errorf("process claimed reward delivery: lock delivery: %w", err)
	}
	if !rows.Next() {
		rowsErr := rows.Err()
		_ = rows.Close()
		if rowsErr != nil {
			return fmt.Errorf("process claimed reward delivery: lock delivery: %w", rowsErr)
		}
		return ErrStateConflict
	}
	delivery, err := scanDelivery(rows)
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
WHERE id = $4 AND status = $5`, DeliveryStatusDelivered, detail, at, id, DeliveryStatusDelivering)
	if err := transitionResult(result, err, "mark reward delivery delivered"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("process claimed reward delivery: commit transaction: %w", err)
	}
	committed = true
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, id int64, lastError string, nextRetryAt *time.Time) error {
	exec := r.executor(ctx)
	if exec == nil {
		return ErrUnavailable
	}
	status := DeliveryStatusFailed
	if nextRetryAt != nil {
		status = DeliveryStatusPending
	}
	result, err := exec.ExecContext(ctx, `UPDATE reward_deliveries
SET status = $1, last_error = $2, next_retry_at = $3, locked_at = NULL, updated_at = NOW()
WHERE id = $4 AND status = $5`, status, lastError, nextRetryAt, id, DeliveryStatusDelivering)
	return transitionResult(result, err, "mark reward delivery failed")
}

func (r *OutboxRepository) Retry(ctx context.Context, id int64) error {
	exec := r.executor(ctx)
	if exec == nil {
		return ErrUnavailable
	}
	result, err := exec.ExecContext(ctx, `UPDATE reward_deliveries
SET status = $1, last_error = NULL, next_retry_at = NOW(), locked_at = NULL, updated_at = NOW()
WHERE id = $2 AND status = $3`, DeliveryStatusPending, id, DeliveryStatusFailed)
	return transitionResult(result, err, "retry reward delivery")
}

func (r *OutboxRepository) Compensate(ctx context.Context, id int64, reason string) error {
	exec := r.executor(ctx)
	if exec == nil {
		return ErrUnavailable
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return fmt.Errorf("compensation reason is required")
	}
	result, err := exec.ExecContext(ctx, `UPDATE reward_deliveries
SET status = $1, reward_detail = $2, next_retry_at = NULL, locked_at = NULL,
	compensated_at = NOW(), updated_at = NOW()
WHERE id = $3 AND status = $4`, DeliveryStatusCompensated, reason, id, DeliveryStatusFailed)
	return transitionResult(result, err, "compensate reward delivery")
}

func (r *OutboxRepository) RecoverStale(ctx context.Context, staleBefore, nextRetryAt time.Time) (int, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return 0, ErrUnavailable
	}
	result, err := exec.ExecContext(ctx, `UPDATE reward_deliveries
SET status = $1,
	last_error = CASE WHEN last_error IS NULL OR last_error = '' THEN $2 ELSE last_error || E'\n' || $2 END,
	next_retry_at = $3, locked_at = NULL, updated_at = NOW()
WHERE status = $4 AND locked_at IS NOT NULL AND locked_at < $5`,
		DeliveryStatusPending, "delivery lock expired and was recovered", nextRetryAt, DeliveryStatusDelivering, staleBefore)
	if err != nil {
		return 0, fmt.Errorf("recover stale reward deliveries: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("recover stale reward deliveries: %w", err)
	}
	return int(count), nil
}

func (r *OutboxRepository) Get(ctx context.Context, id int64) (*Delivery, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, ErrUnavailable
	}
	rows, err := exec.QueryContext(ctx, `SELECT `+outboxColumns+` FROM reward_deliveries WHERE id = $1 LIMIT 1`, id)
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
	delivery, err := scanDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("get reward delivery: %w", err)
	}
	return delivery, nil
}

func (r *OutboxRepository) List(ctx context.Context, filter DeliveryFilter) ([]Delivery, int64, error) {
	exec := r.executor(ctx)
	if exec == nil {
		return nil, 0, ErrUnavailable
	}
	page, pageSize := normalizeDeliveryPage(filter.Page, filter.PageSize)
	where, args := deliveryFilterWhere(filter)
	countRows, err := exec.QueryContext(ctx, `SELECT COUNT(*) FROM reward_deliveries`+where, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("count reward deliveries: %w", err)
	}
	var total int64
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
		return []Delivery{}, 0, nil
	}
	limitPlaceholder := len(args) + 1
	query := fmt.Sprintf(`SELECT %s FROM reward_deliveries%s
ORDER BY created_at DESC, id DESC LIMIT $%d OFFSET $%d`, outboxColumns, where, limitPlaceholder, limitPlaceholder+1)
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list reward deliveries: %w", err)
	}
	defer func() { _ = rows.Close() }()
	deliveries, err := scanDeliveries(rows, "list reward deliveries")
	return deliveries, total, err
}

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func (r *OutboxRepository) executor(ctx context.Context) sqlExecutor {
	if r == nil {
		return nil
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	if r.db != nil {
		return r.db
	}
	if r.client != nil {
		return r.client
	}
	return nil
}

func (r *OutboxRepository) getByIdempotencyKey(ctx context.Context, exec sqlExecutor, key string) (*Delivery, error) {
	rows, err := exec.QueryContext(ctx, `SELECT `+outboxColumns+` FROM reward_deliveries WHERE idempotency_key = $1 LIMIT 1`, key)
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
	delivery, err := scanDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("get idempotent reward delivery: %w", err)
	}
	return delivery, nil
}

func (r *OutboxRepository) getBySource(ctx context.Context, exec sqlExecutor, sourceType string, sourceID int64) (*Delivery, error) {
	rows, err := exec.QueryContext(ctx, `SELECT `+outboxColumns+` FROM reward_deliveries WHERE source_type = $1 AND source_id = $2 LIMIT 1`, sourceType, sourceID)
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
	delivery, err := scanDelivery(rows)
	if err != nil {
		return nil, fmt.Errorf("get source reward delivery: %w", err)
	}
	return delivery, nil
}

func deliveryMatchesCreate(existing *Delivery, input CreateDelivery, snapshot map[string]any) bool {
	if existing == nil || existing.SourceType != input.SourceType || existing.SourceID != input.SourceID ||
		existing.UserID != input.UserID || !equalOptionalID(existing.PrizeID, input.PrizeID) ||
		existing.RewardType != input.RewardType || roundDeliveryAmount(existing.RewardValue) != input.RewardValue ||
		existing.RuleVersion != input.RuleVersion || existing.IdempotencyKey != input.IdempotencyKey {
		return false
	}
	var existingSnapshot map[string]any
	if err := json.Unmarshal(existing.RewardSnapshot, &existingSnapshot); err != nil || existingSnapshot == nil {
		return false
	}
	return reflect.DeepEqual(existingSnapshot, snapshot)
}

func equalOptionalID(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func scanDeliveries(rows *sql.Rows, operation string) ([]Delivery, error) {
	deliveries := make([]Delivery, 0)
	for rows.Next() {
		delivery, err := scanDelivery(rows)
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

func scanDelivery(scanner interface{ Scan(...any) error }) (*Delivery, error) {
	var delivery Delivery
	var snapshot []byte
	var prizeID sql.NullInt64
	var lastError sql.NullString
	var nextRetryAt, lockedAt, deliveredAt, compensatedAt sql.NullTime
	if err := scanner.Scan(
		&delivery.ID, &delivery.SourceType, &delivery.SourceID, &delivery.UserID, &prizeID,
		&snapshot, &delivery.RewardType, &delivery.RewardValue, &delivery.RewardDetail, &delivery.RuleVersion,
		&delivery.IdempotencyKey, &delivery.Status, &delivery.Attempts, &lastError, &nextRetryAt, &lockedAt,
		&deliveredAt, &compensatedAt, &delivery.CreatedAt, &delivery.UpdatedAt,
	); err != nil {
		return nil, err
	}
	delivery.RewardValue = roundDeliveryAmount(delivery.RewardValue)
	delivery.RewardSnapshot = append(json.RawMessage(nil), snapshot...)
	if prizeID.Valid {
		delivery.PrizeID = &prizeID.Int64
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

func transitionResult(result sql.Result, err error, operation string) error {
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", operation, err)
	}
	if rows != 1 {
		return ErrStateConflict
	}
	return nil
}

func deliveryFilterWhere(filter DeliveryFilter) (string, []any) {
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

func normalizeDeliveryLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func normalizeDeliveryPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	return page, normalizeDeliveryLimit(pageSize)
}

func prefixedOutboxColumns(prefix string) string {
	parts := strings.Split(outboxColumns, ",")
	for i := range parts {
		parts[i] = prefix + "." + strings.TrimSpace(parts[i])
	}
	return strings.Join(parts, ", ")
}

func roundDeliveryAmount(value float64) float64 { return math.Round(value*1e8) / 1e8 }

var _ AdminOutbox = (*OutboxRepository)(nil)
