package checkin

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	activityrewards "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/rewards"
)

// RewardDeliveryRunner is the immediate-delivery slice exposed by the activity
// rewards worker. It is intentionally smaller than the worker's lifecycle API.
type RewardDeliveryRunner interface {
	RunByID(ctx context.Context, id int64) error
}

// NewRewardsBlindboxDelivery connects check-in to the activity rewards module
// without importing or delegating to the legacy BlindBoxService.
func NewRewardsBlindboxDelivery(service *activityrewards.Service, runner RewardDeliveryRunner, outbox activityrewards.Outbox) BlindboxDelivery {
	return rewardsBlindboxDelivery{service: service, runner: runner, outbox: outbox}
}

type rewardsBlindboxDelivery struct {
	service *activityrewards.Service
	runner  RewardDeliveryRunner
	outbox  activityrewards.Outbox
}

func (d rewardsBlindboxDelivery) PrepareForCheckin(ctx context.Context, userID, checkinID int64, streakDays int) (*PreparedBlindbox, error) {
	if d.service == nil {
		return nil, nil
	}
	triggered, err := d.service.ShouldTrigger(ctx, userID, streakDays)
	if err != nil {
		return nil, fmt.Errorf("evaluate blindbox trigger: %w", err)
	}
	if !triggered {
		return nil, nil
	}
	prepared, err := d.service.PrepareCheckinBlindbox(ctx, userID, checkinID, streakDays)
	if err != nil || prepared == nil || prepared.Delivery == nil {
		return nil, err
	}
	return &PreparedBlindbox{
		DeliveryID: prepared.Delivery.ID,
		Result: contract.BlindboxResult{
			PrizeName: prepared.Result.PrizeName, Rarity: string(prepared.Result.Rarity), RewardType: string(prepared.Result.RewardType),
			RewardValue: prepared.Result.RewardValue, SubscriptionDays: prepared.Result.SubscriptionDays,
		},
	}, nil
}

func (d rewardsBlindboxDelivery) Deliver(ctx context.Context, deliveryID int64) (*DeliveredBlindbox, error) {
	if d.runner == nil || d.outbox == nil || deliveryID <= 0 {
		return nil, nil
	}
	if err := d.runner.RunByID(ctx, deliveryID); err != nil {
		return nil, err
	}
	delivery, err := d.outbox.Get(ctx, deliveryID)
	if err != nil || delivery == nil {
		return nil, err
	}
	return &DeliveredBlindbox{RewardDetail: delivery.RewardDetail}, nil
}
