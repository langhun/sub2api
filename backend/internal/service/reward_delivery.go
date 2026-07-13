package service

import (
	"context"
	"encoding/json"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	RewardDeliveryStatusPending     = "pending"
	RewardDeliveryStatusDelivering  = "delivering"
	RewardDeliveryStatusDelivered   = "delivered"
	RewardDeliveryStatusFailed      = "failed"
	RewardDeliveryStatusCompensated = "compensated"
)

var (
	ErrRewardDeliveryStateConflict = infraerrors.Conflict(
		"REWARD_DELIVERY_STATE_CONFLICT",
		"reward delivery state conflict",
	)
	ErrRewardDeliveryIdempotencyConflict = infraerrors.Conflict(
		"REWARD_DELIVERY_IDEMPOTENCY_CONFLICT",
		"reward delivery idempotency payload conflict",
	)
)

// RewardDelivery is an immutable reward eligibility plus its delivery state.
// RewardSnapshot must contain the prize fields needed to deliver without
// consulting mutable prize-pool configuration.
type RewardDelivery struct {
	ID             int64           `json:"id"`
	SourceType     string          `json:"source_type"`
	SourceID       int64           `json:"source_id"`
	UserID         int64           `json:"user_id"`
	PrizeItemID    *int64          `json:"prize_item_id,omitempty"`
	RewardSnapshot json.RawMessage `json:"reward_snapshot"`
	RewardType     string          `json:"reward_type"`
	RewardValue    float64         `json:"reward_value"`
	RewardDetail   string          `json:"reward_detail"`
	RuleVersion    string          `json:"rule_version"`
	IdempotencyKey string          `json:"idempotency_key"`
	Status         string          `json:"status"`
	Attempts       int             `json:"attempts"`
	LastError      *string         `json:"last_error,omitempty"`
	NextRetryAt    *time.Time      `json:"next_retry_at,omitempty"`
	LockedAt       *time.Time      `json:"locked_at,omitempty"`
	DeliveredAt    *time.Time      `json:"delivered_at,omitempty"`
	CompensatedAt  *time.Time      `json:"compensated_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type CreateRewardDelivery struct {
	SourceType     string
	SourceID       int64
	UserID         int64
	PrizeItemID    *int64
	RewardSnapshot json.RawMessage
	RewardType     string
	RewardValue    float64
	RewardDetail   string
	RuleVersion    string
	IdempotencyKey string
}

type RewardDeliveryFilter struct {
	Status     string
	SourceType string
	UserID     *int64
	Page       int
	PageSize   int
}

type RewardDeliveryApply func(ctx context.Context, delivery RewardDelivery) (detail string, err error)

type RewardDeliveryStore interface {
	CreatePending(ctx context.Context, input CreateRewardDelivery) (*RewardDelivery, error)
	ClaimByID(ctx context.Context, id int64, now time.Time) (*RewardDelivery, error)
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]RewardDelivery, error)
	ProcessClaimed(ctx context.Context, id int64, apply RewardDeliveryApply) error
	MarkFailed(ctx context.Context, id int64, lastError string, nextRetryAt *time.Time) error
	RecoverStale(ctx context.Context, staleBefore, nextRetryAt time.Time) (int, error)
	GetByID(ctx context.Context, id int64) (*RewardDelivery, error)
	List(ctx context.Context, filter RewardDeliveryFilter) ([]RewardDelivery, int64, error)
}

type RewardDeliveryAdminStore interface {
	RewardDeliveryStore
	Retry(ctx context.Context, id int64) error
	Compensate(ctx context.Context, id int64, reason string) error
}
