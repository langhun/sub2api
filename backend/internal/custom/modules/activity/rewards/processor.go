package rewards

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

// DeliveryProcessor applies a frozen snapshot through core ports only. The
// Outbox adapter supplies a transaction context; this processor must never
// create its own transaction or access a legacy user repository directly.
type DeliveryProcessor struct {
	balance      contract.BalanceWriter
	concurrency  contract.ConcurrencyGranter
	subscription contract.SubscriptionGranter
	invitation   contract.InvitationCodeIssuer
	audit        contract.AuditWriter
	history      BlindboxRecordWriter
}

type ProcessorDependencies struct {
	Balance      contract.BalanceWriter
	Concurrency  contract.ConcurrencyGranter
	Subscription contract.SubscriptionGranter
	Invitation   contract.InvitationCodeIssuer
	Audit        contract.AuditWriter
	History      BlindboxRecordWriter
}

func NewDeliveryProcessor(deps ProcessorDependencies) *DeliveryProcessor {
	return &DeliveryProcessor{
		balance:      deps.Balance,
		concurrency:  deps.Concurrency,
		subscription: deps.Subscription,
		invitation:   deps.Invitation,
		audit:        deps.Audit,
		history:      deps.History,
	}
}

// ProcessDelivery is passed to Worker or Outbox.ExecuteClaimed. Every mutating
// core port receives the immutable delivery key and the supplied transaction
// context, preserving exactly-once behavior across retry attempts.
func (p *DeliveryProcessor) ProcessDelivery(ctx context.Context, delivery Delivery) (string, error) {
	if p == nil || p.audit == nil || p.history == nil {
		return "", ErrUnavailable
	}
	if delivery.SourceType != SourceCheckinBlindbox || delivery.RuleVersion != CheckinBlindboxRuleV1 ||
		delivery.ID <= 0 || delivery.UserID <= 0 || delivery.PrizeID == nil || *delivery.PrizeID <= 0 ||
		delivery.IdempotencyKey == "" || !delivery.RewardType.Valid() || invalidRewardValue(delivery.RewardValue) {
		return "", ErrInvalidDelivery
	}
	var snapshot Snapshot
	if err := json.Unmarshal(delivery.RewardSnapshot, &snapshot); err != nil {
		return "", fmt.Errorf("decode blind-box reward snapshot: %w", err)
	}
	if err := snapshot.Validate(); err != nil || snapshot.PrizeID != *delivery.PrizeID || snapshot.RewardType != delivery.RewardType ||
		!sameRewardValue(snapshot.RewardValue, delivery.RewardValue) {
		return "", ErrInvalidDelivery
	}

	detail, err := p.applySnapshot(ctx, delivery.UserID, delivery.IdempotencyKey, snapshot)
	if err != nil {
		return "", err
	}
	auditAmount := snapshot.RewardValue
	if snapshot.RewardType == RewardTypeSubscription {
		auditAmount = float64(snapshot.SubscriptionDays)
	}
	if err := p.audit.WriteActivityAudit(ctx, contract.AuditEntry{
		UserID:         delivery.UserID,
		Type:           SourceCheckinBlindbox,
		Amount:         auditAmount,
		ReferenceID:    deliveryKey(delivery.SourceType, delivery.SourceID),
		IdempotencyKey: delivery.IdempotencyKey,
	}); err != nil {
		return "", fmt.Errorf("write blind-box audit: %w", err)
	}
	if err := p.history.RecordBlindboxDelivery(ctx, BlindboxRecord{
		DeliveryID: delivery.ID, UserID: delivery.UserID, PrizeID: snapshot.PrizeID,
		PrizeName: snapshot.PrizeName, Rarity: snapshot.Rarity, RewardType: snapshot.RewardType,
		RewardValue: snapshot.RewardValue, RewardDetail: detail,
		SubscriptionDays: snapshot.SubscriptionDays, StreakDays: snapshot.StreakDays,
	}); err != nil {
		return "", fmt.Errorf("record blind-box delivery: %w", err)
	}
	return detail, nil
}

func (p *DeliveryProcessor) applySnapshot(ctx context.Context, userID int64, idempotencyKey string, snapshot Snapshot) (string, error) {
	switch snapshot.RewardType {
	case RewardTypeBalance:
		if snapshot.RewardValue > 0 {
			if p.balance == nil {
				return "", ErrUnavailable
			}
			if err := p.balance.Credit(ctx, contract.BalanceOperation{
				UserID: userID, Amount: snapshot.RewardValue, Reason: SourceCheckinBlindbox, IdempotencyKey: idempotencyKey,
			}); err != nil {
				return "", fmt.Errorf("credit blind-box balance: %w", err)
			}
		}
	case RewardTypeConcurrency:
		if p.concurrency == nil {
			return "", ErrUnavailable
		}
		if err := p.concurrency.GrantConcurrency(ctx, contract.ConcurrencyGrant{
			UserID: userID, Slots: int(snapshot.RewardValue), Reason: SourceCheckinBlindbox, IdempotencyKey: idempotencyKey,
		}); err != nil {
			return "", fmt.Errorf("grant blind-box concurrency: %w", err)
		}
	case RewardTypeSubscription:
		if p.subscription == nil {
			return "", ErrUnavailable
		}
		if err := p.subscription.GrantOrExtendSubscription(ctx, contract.SubscriptionGrant{
			UserID: userID, SubscriptionID: *snapshot.SubscriptionID, Days: snapshot.SubscriptionDays,
			IdempotencyKey: idempotencyKey, Note: "check-in blind box reward",
		}); err != nil {
			return "", fmt.Errorf("grant blind-box subscription: %w", err)
		}
	case RewardTypeInvitationCode:
		if p.invitation == nil {
			return "", ErrUnavailable
		}
		code, err := p.invitation.IssueInvitationCode(ctx, contract.InvitationCodeRequest{
			UserID: userID, Code: snapshot.InvitationCode, IdempotencyKey: idempotencyKey,
		})
		if err != nil {
			return "", fmt.Errorf("issue blind-box invitation code: %w", err)
		}
		if code != "" && code != snapshot.InvitationCode {
			return "", ErrInvalidDelivery
		}
		return snapshot.InvitationCode, nil
	default:
		return "", ErrInvalidDelivery
	}
	return "", nil
}
