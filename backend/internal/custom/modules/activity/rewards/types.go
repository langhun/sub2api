// Package rewards owns the activity blind-box and durable reward-delivery
// boundary. Adapters for legacy persistence and core services belong outside
// this package so this package never owns a balance transaction.
package rewards

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	SourceCheckinBlindbox = "checkin_blindbox"
	CheckinBlindboxRuleV1 = "v1"

	DeliveryStatusPending     = "pending"
	DeliveryStatusDelivering  = "delivering"
	DeliveryStatusDelivered   = "delivered"
	DeliveryStatusFailed      = "failed"
	DeliveryStatusCompensated = "compensated"
)

var (
	ErrInvalidPrize = infraerrors.BadRequest(
		"BLINDBOX_PRIZE_INVALID",
		"blind box prize configuration is invalid",
	)
	ErrInvalidDelivery = infraerrors.BadRequest(
		"REWARD_DELIVERY_INVALID",
		"reward delivery is invalid",
	)
	ErrUnavailable = infraerrors.InternalServer(
		"ACTIVITY_REWARD_UNAVAILABLE",
		"activity reward dependency is unavailable",
	)
	ErrStateConflict = infraerrors.Conflict(
		"REWARD_DELIVERY_STATE_CONFLICT",
		"reward delivery state conflict",
	)
	ErrIdempotencyConflict = infraerrors.Conflict(
		"REWARD_DELIVERY_IDEMPOTENCY_CONFLICT",
		"reward delivery idempotency payload conflict",
	)
)

type Rarity string

const (
	RarityCommon    Rarity = "common"
	RarityRare      Rarity = "rare"
	RarityEpic      Rarity = "epic"
	RarityLegendary Rarity = "legendary"
)

type RewardType string

const (
	RewardTypeBalance        RewardType = "balance"
	RewardTypeConcurrency    RewardType = "concurrency"
	RewardTypeSubscription   RewardType = "subscription"
	RewardTypeInvitationCode RewardType = "invitation_code"
)

// Prize is the activity-owned representation of a blind-box configuration.
// The selected fields are frozen into Snapshot before a delivery is queued.
type Prize struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Rarity           Rarity     `json:"rarity"`
	RewardType       RewardType `json:"reward_type"`
	RewardValue      float64    `json:"reward_value"`
	RewardValueMax   float64    `json:"reward_value_max"`
	SubscriptionID   *int64     `json:"subscription_id,omitempty"`
	SubscriptionDays int        `json:"subscription_days"`
	Weight           int        `json:"weight"`
	Enabled          bool       `json:"is_enabled"`
	CreatedAt        string     `json:"created_at"`
	UpdatedAt        string     `json:"updated_at"`
}

func (p Prize) Validate() error {
	if p.ID < 0 || strings.TrimSpace(p.Name) == "" || !p.Rarity.Valid() || !p.RewardType.Valid() || p.Weight <= 0 ||
		invalidRewardValue(p.RewardValue) || invalidRewardValue(p.RewardValueMax) || p.RewardValueMax > 0 && p.RewardValueMax < p.RewardValue {
		return ErrInvalidPrize
	}
	if p.RewardType == RewardTypeSubscription && (p.SubscriptionID == nil || *p.SubscriptionID <= 0 || p.SubscriptionDays <= 0) {
		return ErrInvalidPrize
	}
	if p.RewardType == RewardTypeConcurrency && p.RewardValue != math.Trunc(p.RewardValue) {
		return ErrInvalidPrize
	}
	return nil
}

func (r Rarity) Valid() bool {
	return r == RarityCommon || r == RarityRare || r == RarityEpic || r == RarityLegendary
}

func (r RewardType) Valid() bool {
	return r == RewardTypeBalance || r == RewardTypeConcurrency || r == RewardTypeSubscription || r == RewardTypeInvitationCode
}

func invalidRewardValue(value float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0) || value < 0
}

// PrizeCatalog is the activity persistence port. ListEnabled must observe the
// caller's transaction context so a selected prize and pending outbox record
// can commit atomically with check-in eligibility.
type PrizeCatalog interface {
	ListEnabled(ctx context.Context) ([]Prize, error)
	List(ctx context.Context) ([]Prize, error)
	Get(ctx context.Context, prizeID int64) (*Prize, error)
	Save(ctx context.Context, prize Prize) (Prize, error)
	Archive(ctx context.Context, prizeID int64) error
	Stats(ctx context.Context) (PrizeStats, error)
}

// PrizeStats is the established admin blind-box summary payload.
type PrizeStats struct {
	TotalItems   int `json:"total_items"`
	EnabledItems int `json:"enabled_items"`
	TotalDraws   int `json:"total_draws"`
}

// CheckinCounter supplies the total-count trigger from the same transaction as
// the check-in write when the trigger type is "total".
type CheckinCounter interface {
	CountCheckins(ctx context.Context, userID int64) (int, error)
}

// RandomSource lets integration use cryptographic randomness while tests stay
// deterministic. IntN must return a value in [0, n); Float64 in [0, 1).
type RandomSource interface {
	IntN(n int) (int, error)
	Float64() (float64, error)
}

// InvitationCodeGenerator freezes a code into the immutable delivery
// snapshot. Persisting that code is performed later by contract.InvitationCodeIssuer.
type InvitationCodeGenerator interface {
	GenerateInvitationCode(ctx context.Context) (string, error)
}

// Snapshot captures every mutable prize attribute needed by a worker. Delivery
// processors must not query the current prize pool while applying a snapshot.
type Snapshot struct {
	PrizeID          int64      `json:"prize_item_id"`
	PrizeName        string     `json:"prize_name"`
	Rarity           Rarity     `json:"rarity"`
	RewardType       RewardType `json:"reward_type"`
	RewardValue      float64    `json:"reward_value"`
	SubscriptionID   *int64     `json:"subscription_id,omitempty"`
	SubscriptionDays int        `json:"subscription_days,omitempty"`
	StreakDays       int        `json:"streak_days"`
	InvitationCode   string     `json:"invitation_code,omitempty"`
}

func (s Snapshot) Validate() error {
	prize := Prize{
		ID:               s.PrizeID,
		Name:             s.PrizeName,
		Rarity:           s.Rarity,
		RewardType:       s.RewardType,
		RewardValue:      s.RewardValue,
		SubscriptionID:   s.SubscriptionID,
		SubscriptionDays: s.SubscriptionDays,
		Weight:           1,
	}
	if s.PrizeID <= 0 || s.StreakDays < 0 || prize.Validate() != nil {
		return ErrInvalidPrize
	}
	if s.RewardType == RewardTypeInvitationCode && strings.TrimSpace(s.InvitationCode) == "" {
		return ErrInvalidPrize
	}
	return nil
}

type Delivery struct {
	ID             int64           `json:"id"`
	SourceType     string          `json:"source_type"`
	SourceID       int64           `json:"source_id"`
	UserID         int64           `json:"user_id"`
	PrizeID        *int64          `json:"prize_item_id,omitempty"`
	RewardSnapshot json.RawMessage `json:"reward_snapshot"`
	RewardType     RewardType      `json:"reward_type"`
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

type CreateDelivery struct {
	SourceType     string
	SourceID       int64
	UserID         int64
	PrizeID        *int64
	RewardSnapshot json.RawMessage
	RewardType     RewardType
	RewardValue    float64
	RuleVersion    string
	IdempotencyKey string
}

func (d CreateDelivery) Validate() error {
	if strings.TrimSpace(d.SourceType) == "" || d.SourceID <= 0 || d.UserID <= 0 || d.PrizeID == nil || *d.PrizeID <= 0 ||
		!d.RewardType.Valid() || invalidRewardValue(d.RewardValue) || strings.TrimSpace(d.RuleVersion) == "" || strings.TrimSpace(d.IdempotencyKey) == "" {
		return ErrInvalidDelivery
	}
	var snapshot Snapshot
	if len(d.RewardSnapshot) == 0 || json.Unmarshal(d.RewardSnapshot, &snapshot) != nil || snapshot.Validate() != nil ||
		snapshot.PrizeID != *d.PrizeID || snapshot.RewardType != d.RewardType || !sameRewardValue(snapshot.RewardValue, d.RewardValue) {
		return ErrInvalidDelivery
	}
	return nil
}

type DeliveryFilter struct {
	Status     string
	SourceType string
	UserID     *int64
	Page       int
	PageSize   int
}

type DeliveryApply func(ctx context.Context, delivery Delivery) (detail string, err error)

// Outbox coordinates durable state transitions. ExecuteClaimed must call apply
// in the delivery transaction and mark the row delivered before committing.
// Core balance, subscription and audit adapters receive that same context.
type Outbox interface {
	Enqueue(ctx context.Context, input CreateDelivery) (*Delivery, error)
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]Delivery, error)
	ClaimByID(ctx context.Context, id int64, now time.Time) (*Delivery, error)
	ExecuteClaimed(ctx context.Context, id int64, apply DeliveryApply) error
	MarkFailed(ctx context.Context, id int64, lastError string, nextRetryAt *time.Time) error
	RecoverStale(ctx context.Context, staleBefore, nextRetryAt time.Time) (int, error)
	Get(ctx context.Context, id int64) (*Delivery, error)
	List(ctx context.Context, filter DeliveryFilter) ([]Delivery, int64, error)
}

// AdminOutbox exposes only recovery operations needed by future activity admin
// screens. It intentionally does not expose direct balance mutations.
type AdminOutbox interface {
	Outbox
	Retry(ctx context.Context, id int64) error
	Compensate(ctx context.Context, id int64, reason string) error
}

// BlindboxRecord is the immutable activity history written in the same
// transaction as delivery effects.
type BlindboxRecord struct {
	DeliveryID       int64
	UserID           int64
	PrizeID          int64
	PrizeName        string
	Rarity           Rarity
	RewardType       RewardType
	RewardValue      float64
	RewardDetail     string
	SubscriptionDays int
	StreakDays       int
}

type BlindboxRecordWriter interface {
	RecordBlindboxDelivery(ctx context.Context, record BlindboxRecord) error
}

func sameRewardValue(left, right float64) bool {
	return math.Round(left*1e8) == math.Round(right*1e8)
}

func deliveryKey(sourceType string, sourceID int64) string {
	return fmt.Sprintf("%s:%d", sourceType, sourceID)
}
