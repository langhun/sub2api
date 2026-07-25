package rewards

import (
	"context"
	"fmt"
)

// AdminOperations is the module-owned application boundary for the established
// blind-box prize and reward-delivery administration endpoints.
type AdminOperations interface {
	ListPrizeItems(ctx context.Context) ([]Prize, error)
	CreatePrizeItem(ctx context.Context, request CreatePrizeItemRequest) (*Prize, error)
	UpdatePrizeItem(ctx context.Context, id int64, request UpdatePrizeItemRequest) (*Prize, error)
	DeletePrizeItem(ctx context.Context, id int64) error
	GetStats(ctx context.Context) (PrizeStats, error)
	ListRewardDeliveries(ctx context.Context, filter DeliveryFilter) ([]Delivery, int64, error)
	RetryRewardDelivery(ctx context.Context, id int64) (*Delivery, error)
	CompensateRewardDelivery(ctx context.Context, id int64, reason string) (*Delivery, error)
}

type CreatePrizeItemRequest struct {
	Name             string     `json:"name" binding:"required"`
	Rarity           Rarity     `json:"rarity" binding:"required"`
	RewardType       RewardType `json:"reward_type" binding:"required"`
	RewardValue      float64    `json:"reward_value"`
	RewardValueMax   float64    `json:"reward_value_max"`
	SubscriptionID   *int64     `json:"subscription_id"`
	SubscriptionDays int        `json:"subscription_days"`
	Weight           int        `json:"weight"`
	IsEnabled        *bool      `json:"is_enabled"`
}

type UpdatePrizeItemRequest struct {
	Name             *string     `json:"name"`
	Rarity           *Rarity     `json:"rarity"`
	RewardType       *RewardType `json:"reward_type"`
	RewardValue      *float64    `json:"reward_value"`
	RewardValueMax   *float64    `json:"reward_value_max"`
	SubscriptionID   **int64     `json:"subscription_id"`
	SubscriptionDays *int        `json:"subscription_days"`
	Weight           *int        `json:"weight"`
	IsEnabled        *bool       `json:"is_enabled"`
}

// AdminService owns the activity prize and delivery administration behavior.
// Its worker and persistence ports remain module-local.
type AdminService struct {
	prizes PrizeCatalog
	outbox AdminOutbox
	worker *Worker
}

func NewAdminService(prizes PrizeCatalog, outbox AdminOutbox, worker *Worker) *AdminService {
	return &AdminService{prizes: prizes, outbox: outbox, worker: worker}
}

func (s *AdminService) ListPrizeItems(ctx context.Context) ([]Prize, error) {
	if s == nil || s.prizes == nil {
		return nil, ErrUnavailable
	}
	return s.prizes.List(ctx)
}

func (s *AdminService) CreatePrizeItem(ctx context.Context, request CreatePrizeItemRequest) (*Prize, error) {
	if s == nil || s.prizes == nil {
		return nil, ErrUnavailable
	}
	weight := request.Weight
	if weight <= 0 {
		weight = 100
	}
	enabled := true
	if request.IsEnabled != nil {
		enabled = *request.IsEnabled
	}
	prize, err := s.prizes.Save(ctx, Prize{
		Name: request.Name, Rarity: request.Rarity, RewardType: request.RewardType,
		RewardValue: request.RewardValue, RewardValueMax: request.RewardValueMax,
		SubscriptionID: request.SubscriptionID, SubscriptionDays: request.SubscriptionDays,
		Weight: weight, Enabled: enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("create prize item: %w", err)
	}
	return &prize, nil
}

func (s *AdminService) UpdatePrizeItem(ctx context.Context, id int64, request UpdatePrizeItemRequest) (*Prize, error) {
	if s == nil || s.prizes == nil {
		return nil, ErrUnavailable
	}
	if id <= 0 {
		return nil, ErrInvalidPrize
	}
	prize, err := s.prizes.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get prize item: %w", err)
	}
	if prize == nil {
		return nil, ErrInvalidPrize
	}
	if request.Name != nil {
		prize.Name = *request.Name
	}
	if request.Rarity != nil {
		prize.Rarity = *request.Rarity
	}
	if request.RewardType != nil {
		prize.RewardType = *request.RewardType
	}
	if request.RewardValue != nil {
		prize.RewardValue = *request.RewardValue
	}
	if request.RewardValueMax != nil {
		prize.RewardValueMax = *request.RewardValueMax
	}
	if request.SubscriptionID != nil {
		prize.SubscriptionID = *request.SubscriptionID
	}
	if request.SubscriptionDays != nil {
		prize.SubscriptionDays = *request.SubscriptionDays
	}
	if request.Weight != nil {
		prize.Weight = *request.Weight
	}
	if request.IsEnabled != nil {
		prize.Enabled = *request.IsEnabled
	}
	saved, err := s.prizes.Save(ctx, *prize)
	if err != nil {
		return nil, fmt.Errorf("update prize item: %w", err)
	}
	return &saved, nil
}

func (s *AdminService) DeletePrizeItem(ctx context.Context, id int64) error {
	if s == nil || s.prizes == nil {
		return ErrUnavailable
	}
	if id <= 0 {
		return ErrInvalidPrize
	}
	return s.prizes.Archive(ctx, id)
}

func (s *AdminService) GetStats(ctx context.Context) (PrizeStats, error) {
	if s == nil || s.prizes == nil {
		return PrizeStats{}, ErrUnavailable
	}
	return s.prizes.Stats(ctx)
}

func (s *AdminService) ListRewardDeliveries(ctx context.Context, filter DeliveryFilter) ([]Delivery, int64, error) {
	if s == nil || s.outbox == nil {
		return nil, 0, ErrUnavailable
	}
	return s.outbox.List(ctx, filter)
}

func (s *AdminService) RetryRewardDelivery(ctx context.Context, id int64) (*Delivery, error) {
	if s == nil || s.outbox == nil || s.worker == nil {
		return nil, ErrUnavailable
	}
	if err := s.outbox.Retry(ctx, id); err != nil {
		return nil, err
	}
	if err := s.worker.RunByID(ctx, id); err != nil {
		return nil, err
	}
	return s.outbox.Get(ctx, id)
}

func (s *AdminService) CompensateRewardDelivery(ctx context.Context, id int64, reason string) (*Delivery, error) {
	if s == nil || s.outbox == nil {
		return nil, ErrUnavailable
	}
	if err := s.outbox.Compensate(ctx, id, reason); err != nil {
		return nil, err
	}
	return s.outbox.Get(ctx, id)
}

var _ AdminOperations = (*AdminService)(nil)
