package rewards

import (
	"context"
	"fmt"
	"time"

	legacy "github.com/Wei-Shaw/sub2api/internal/service"
)

// LegacyBlindboxAdapter is the executable migration bridge for the existing
// blind-box implementation. It preserves the legacy service's transaction
// semantics while callers move to activity-owned types and worker lifecycle.
// It is temporary: new core-port adapters can replace it without changing the
// activity package API.
type LegacyBlindboxAdapter struct {
	service *legacy.BlindBoxService
}

func NewLegacyBlindboxAdapter(service *legacy.BlindBoxService) *LegacyBlindboxAdapter {
	return &LegacyBlindboxAdapter{service: service}
}

func (a *LegacyBlindboxAdapter) ShouldTrigger(ctx context.Context, userID int64, streakDays int) (bool, error) {
	if a == nil || a.service == nil {
		return false, ErrUnavailable
	}
	return a.service.ShouldTriggerBlindbox(ctx, userID, streakDays), nil
}

// PrepareCheckinBlindbox delegates to the legacy service, which selects the
// prize and inserts the pending delivery in the caller's transaction context.
func (a *LegacyBlindboxAdapter) PrepareCheckinBlindbox(ctx context.Context, userID, checkinID int64, streakDays int) (*PreparedDelivery, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	result, delivery, err := a.service.PrepareDelivery(ctx, userID, checkinID, streakDays)
	if err != nil {
		return nil, fmt.Errorf("legacy prepare blind-box delivery: %w", err)
	}
	if result == nil || delivery == nil {
		return nil, nil
	}
	return &PreparedDelivery{
		Delivery: deliveryFromLegacy(*delivery),
		Result: DrawResult{
			PrizeName: result.PrizeName, Rarity: Rarity(result.Rarity), RewardType: RewardType(result.RewardType),
			RewardValue: result.RewardValue, SubscriptionDays: result.SubscriptionDays,
		},
	}, nil
}

// ProcessDelivery implements Processor using the established blind-box
// transaction effects. It is the safe runtime bridge until BalanceWriter,
// SubscriptionGranter, AuditWriter, and record adapters are wired by root.
func (a *LegacyBlindboxAdapter) ProcessDelivery(ctx context.Context, delivery Delivery) (string, error) {
	if a == nil || a.service == nil {
		return "", ErrUnavailable
	}
	return a.service.ProcessRewardDelivery(ctx, deliveryToLegacy(delivery))
}

// LegacyPrizeCatalog maps the existing admin prize-pool service to the module
// catalog boundary. It never mutates the balance or subscription domains.
type LegacyPrizeCatalog struct {
	service *legacy.BlindBoxService
}

func NewLegacyPrizeCatalog(service *legacy.BlindBoxService) *LegacyPrizeCatalog {
	return &LegacyPrizeCatalog{service: service}
}

func (a *LegacyPrizeCatalog) ListEnabled(ctx context.Context) ([]Prize, error) {
	prizes, err := a.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Prize, 0, len(prizes))
	for _, prize := range prizes {
		if prize.Enabled {
			result = append(result, prize)
		}
	}
	return result, nil
}

func (a *LegacyPrizeCatalog) List(ctx context.Context) ([]Prize, error) {
	if a == nil || a.service == nil {
		return nil, ErrUnavailable
	}
	items, err := a.service.ListPrizeItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("legacy list blind-box prizes: %w", err)
	}
	result := make([]Prize, 0, len(items))
	for _, item := range items {
		result = append(result, prizeFromLegacy(item))
	}
	return result, nil
}

func (a *LegacyPrizeCatalog) Save(ctx context.Context, prize Prize) (Prize, error) {
	if a == nil || a.service == nil {
		return Prize{}, ErrUnavailable
	}
	if err := prize.Validate(); err != nil {
		return Prize{}, err
	}
	enabled := prize.Enabled
	if prize.ID == 0 {
		item, err := a.service.CreatePrizeItem(ctx, legacy.CreatePrizeItemRequest{
			Name: prize.Name, Rarity: string(prize.Rarity), RewardType: string(prize.RewardType),
			RewardValue: prize.RewardValue, RewardValueMax: prize.RewardValueMax,
			SubscriptionID: prize.SubscriptionID, SubscriptionDays: prize.SubscriptionDays,
			Weight: prize.Weight, IsEnabled: &enabled,
		})
		if err != nil {
			return Prize{}, fmt.Errorf("legacy create blind-box prize: %w", err)
		}
		return prizeFromLegacy(*item), nil
	}
	name, rarity, rewardType := prize.Name, string(prize.Rarity), string(prize.RewardType)
	value, valueMax, days, weight := prize.RewardValue, prize.RewardValueMax, prize.SubscriptionDays, prize.Weight
	subscriptionID := prize.SubscriptionID
	item, err := a.service.UpdatePrizeItem(ctx, prize.ID, legacy.UpdatePrizeItemRequest{
		Name: &name, Rarity: &rarity, RewardType: &rewardType,
		RewardValue: &value, RewardValueMax: &valueMax, SubscriptionID: &subscriptionID,
		SubscriptionDays: &days, Weight: &weight, IsEnabled: &enabled,
	})
	if err != nil {
		return Prize{}, fmt.Errorf("legacy update blind-box prize: %w", err)
	}
	return prizeFromLegacy(*item), nil
}

func (a *LegacyPrizeCatalog) Archive(ctx context.Context, prizeID int64) error {
	if a == nil || a.service == nil {
		return ErrUnavailable
	}
	if prizeID <= 0 {
		return ErrInvalidPrize
	}
	return a.service.DeletePrizeItem(ctx, prizeID)
}

// LegacyOutboxAdapter allows the activity Worker to use the currently proven
// reward_deliveries persistence implementation. It maps only data and does not
// create transactions, alter SQL, or change idempotency behavior.
type LegacyOutboxAdapter struct {
	store legacy.RewardDeliveryStore
}

func NewLegacyOutboxAdapter(store legacy.RewardDeliveryStore) *LegacyOutboxAdapter {
	return &LegacyOutboxAdapter{store: store}
}

// NewLegacyRuntime creates the one activity-owned worker that uses existing
// reward-delivery persistence and blind-box transaction effects. It does not
// start automatically, so application composition can register Stop before
// beginning background work.
//
// Do not use it together with service.ProvideRewardDeliveryWorkerRuntime: both
// workers claim the same pending rows and duplicate lifecycle ownership.
func NewLegacyRuntime(store legacy.RewardDeliveryStore, blindbox *legacy.BlindBoxService, options WorkerOptions) *Runtime {
	return NewRuntime(NewWorker(NewLegacyOutboxAdapter(store), NewLegacyBlindboxAdapter(blindbox), options))
}

// ProvideLegacyRuntime is the Wire-friendly migration provider. Root must
// replace, rather than add beside, the legacy reward-delivery runtime provider
// and invoke Runtime.Stop from cmd/server's cleanup path.
func ProvideLegacyRuntime(store legacy.RewardDeliveryStore, blindbox *legacy.BlindBoxService) *Runtime {
	runtime := NewLegacyRuntime(store, blindbox, WorkerOptions{})
	runtime.Start(context.Background())
	return runtime
}

func (a *LegacyOutboxAdapter) Enqueue(ctx context.Context, input CreateDelivery) (*Delivery, error) {
	if a == nil || a.store == nil {
		return nil, ErrUnavailable
	}
	if err := input.Validate(); err != nil {
		return nil, err
	}
	delivery, err := a.store.CreatePending(ctx, legacy.CreateRewardDelivery{
		SourceType: input.SourceType, SourceID: input.SourceID, UserID: input.UserID, PrizeItemID: input.PrizeID,
		RewardSnapshot: input.RewardSnapshot, RewardType: string(input.RewardType), RewardValue: input.RewardValue,
		RuleVersion: input.RuleVersion, IdempotencyKey: input.IdempotencyKey,
	})
	if err != nil {
		return nil, err
	}
	return deliveryFromLegacy(*delivery), nil
}

func (a *LegacyOutboxAdapter) ClaimDue(ctx context.Context, now time.Time, limit int) ([]Delivery, error) {
	if a == nil || a.store == nil {
		return nil, ErrUnavailable
	}
	deliveries, err := a.store.ClaimDue(ctx, now, limit)
	if err != nil {
		return nil, err
	}
	return deliveriesFromLegacy(deliveries), nil
}

func (a *LegacyOutboxAdapter) ClaimByID(ctx context.Context, id int64, now time.Time) (*Delivery, error) {
	if a == nil || a.store == nil {
		return nil, ErrUnavailable
	}
	delivery, err := a.store.ClaimByID(ctx, id, now)
	if err != nil || delivery == nil {
		return nil, err
	}
	return deliveryFromLegacy(*delivery), nil
}

func (a *LegacyOutboxAdapter) ExecuteClaimed(ctx context.Context, id int64, apply DeliveryApply) error {
	if a == nil || a.store == nil {
		return ErrUnavailable
	}
	if apply == nil {
		return ErrInvalidDelivery
	}
	return a.store.ProcessClaimed(ctx, id, func(txCtx context.Context, delivery legacy.RewardDelivery) (string, error) {
		return apply(txCtx, *deliveryFromLegacy(delivery))
	})
}

func (a *LegacyOutboxAdapter) MarkFailed(ctx context.Context, id int64, lastError string, nextRetryAt *time.Time) error {
	if a == nil || a.store == nil {
		return ErrUnavailable
	}
	return a.store.MarkFailed(ctx, id, lastError, nextRetryAt)
}

func (a *LegacyOutboxAdapter) RecoverStale(ctx context.Context, staleBefore, nextRetryAt time.Time) (int, error) {
	if a == nil || a.store == nil {
		return 0, ErrUnavailable
	}
	return a.store.RecoverStale(ctx, staleBefore, nextRetryAt)
}

func (a *LegacyOutboxAdapter) Get(ctx context.Context, id int64) (*Delivery, error) {
	if a == nil || a.store == nil {
		return nil, ErrUnavailable
	}
	delivery, err := a.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return deliveryFromLegacy(*delivery), nil
}

func (a *LegacyOutboxAdapter) List(ctx context.Context, filter DeliveryFilter) ([]Delivery, int64, error) {
	if a == nil || a.store == nil {
		return nil, 0, ErrUnavailable
	}
	deliveries, total, err := a.store.List(ctx, legacy.RewardDeliveryFilter{
		Status: filter.Status, SourceType: filter.SourceType, UserID: filter.UserID, Page: filter.Page, PageSize: filter.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	return deliveriesFromLegacy(deliveries), total, nil
}

func (a *LegacyOutboxAdapter) Retry(ctx context.Context, id int64) error {
	adminStore, ok := a.adminStore()
	if !ok {
		return ErrUnavailable
	}
	return adminStore.Retry(ctx, id)
}

func (a *LegacyOutboxAdapter) Compensate(ctx context.Context, id int64, reason string) error {
	adminStore, ok := a.adminStore()
	if !ok {
		return ErrUnavailable
	}
	return adminStore.Compensate(ctx, id, reason)
}

func (a *LegacyOutboxAdapter) adminStore() (legacy.RewardDeliveryAdminStore, bool) {
	if a == nil || a.store == nil {
		return nil, false
	}
	store, ok := a.store.(legacy.RewardDeliveryAdminStore)
	return store, ok
}

func prizeFromLegacy(item legacy.PrizeItem) Prize {
	return Prize{
		ID: item.ID, Name: item.Name, Rarity: Rarity(item.Rarity), RewardType: RewardType(item.RewardType),
		RewardValue: item.RewardValue, RewardValueMax: item.RewardValueMax, SubscriptionID: item.SubscriptionID,
		SubscriptionDays: item.SubscriptionDays, Weight: item.Weight, Enabled: item.IsEnabled,
	}
}

func deliveryFromLegacy(delivery legacy.RewardDelivery) *Delivery {
	return &Delivery{
		ID: delivery.ID, SourceType: delivery.SourceType, SourceID: delivery.SourceID, UserID: delivery.UserID,
		PrizeID: delivery.PrizeItemID, RewardSnapshot: delivery.RewardSnapshot, RewardType: RewardType(delivery.RewardType),
		RewardValue: delivery.RewardValue, RewardDetail: delivery.RewardDetail, RuleVersion: delivery.RuleVersion,
		IdempotencyKey: delivery.IdempotencyKey, Status: delivery.Status, Attempts: delivery.Attempts,
		LastError: delivery.LastError, NextRetryAt: delivery.NextRetryAt, LockedAt: delivery.LockedAt,
		DeliveredAt: delivery.DeliveredAt, CompensatedAt: delivery.CompensatedAt,
		CreatedAt: delivery.CreatedAt, UpdatedAt: delivery.UpdatedAt,
	}
}

func deliveriesFromLegacy(deliveries []legacy.RewardDelivery) []Delivery {
	result := make([]Delivery, 0, len(deliveries))
	for _, delivery := range deliveries {
		result = append(result, *deliveryFromLegacy(delivery))
	}
	return result
}

func deliveryToLegacy(delivery Delivery) legacy.RewardDelivery {
	return legacy.RewardDelivery{
		ID: delivery.ID, SourceType: delivery.SourceType, SourceID: delivery.SourceID, UserID: delivery.UserID,
		PrizeItemID: delivery.PrizeID, RewardSnapshot: delivery.RewardSnapshot, RewardType: string(delivery.RewardType),
		RewardValue: delivery.RewardValue, RewardDetail: delivery.RewardDetail, RuleVersion: delivery.RuleVersion,
		IdempotencyKey: delivery.IdempotencyKey, Status: delivery.Status, Attempts: delivery.Attempts,
		LastError: delivery.LastError, NextRetryAt: delivery.NextRetryAt, LockedAt: delivery.LockedAt,
		DeliveredAt: delivery.DeliveredAt, CompensatedAt: delivery.CompensatedAt,
		CreatedAt: delivery.CreatedAt, UpdatedAt: delivery.UpdatedAt,
	}
}
