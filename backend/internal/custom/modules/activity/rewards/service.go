package rewards

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"math/big"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

// Service prepares immutable blind-box reward deliveries. The caller supplies
// its transaction context; this service never starts or commits a transaction.
type Service struct {
	settings contract.SettingsReader
	checkins CheckinCounter
	prizes   PrizeCatalog
	outbox   Outbox
	codes    InvitationCodeGenerator
	random   RandomSource
}

type ServiceDependencies struct {
	Settings contract.SettingsReader
	Checkins CheckinCounter
	Prizes   PrizeCatalog
	Outbox   Outbox
	Codes    InvitationCodeGenerator
	Random   RandomSource
}

func NewService(deps ServiceDependencies) *Service {
	random := deps.Random
	if random == nil {
		random = cryptoRandomSource{}
	}
	return &Service{
		settings: deps.Settings,
		checkins: deps.Checkins,
		prizes:   deps.Prizes,
		outbox:   deps.Outbox,
		codes:    deps.Codes,
		random:   random,
	}
}

// ShouldTrigger evaluates the frozen legacy trigger rules but leaves database
// reads to the transaction-aware CheckinCounter adapter.
func (s *Service) ShouldTrigger(ctx context.Context, userID int64, streakDays int) (bool, error) {
	if s == nil || s.settings == nil {
		return false, ErrUnavailable
	}
	settings, err := s.settings.GetActivitySettings(ctx)
	if err != nil {
		return false, fmt.Errorf("read activity settings: %w", err)
	}
	if (!settings.Checkin.Enabled && !settings.Checkin.LuckEnabled) || !settings.Blindbox.Enabled || settings.Blindbox.Interval <= 0 {
		return false, nil
	}
	if settings.Blindbox.TriggerType != "total" {
		return streakDays > 0 && streakDays%settings.Blindbox.Interval == 0, nil
	}
	if s.checkins == nil {
		return false, ErrUnavailable
	}
	total, err := s.checkins.CountCheckins(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("count checkins: %w", err)
	}
	return total > 0 && total%settings.Blindbox.Interval == 0, nil
}

// PreparedDelivery is the response-safe representation of a queued prize.
// RewardDetail remains empty until the worker has committed delivery effects.
type PreparedDelivery struct {
	Delivery *Delivery
	Result   DrawResult
}

type DrawResult struct {
	PrizeName        string
	Rarity           Rarity
	RewardType       RewardType
	RewardValue      float64
	SubscriptionDays int
}

// PrepareCheckinBlindbox selects exactly one enabled prize, freezes all
// delivery fields, and enqueues it in the caller's current transaction.
func (s *Service) PrepareCheckinBlindbox(ctx context.Context, userID, checkinID int64, streakDays int) (*PreparedDelivery, error) {
	if s == nil || s.prizes == nil || s.outbox == nil || s.random == nil {
		return nil, ErrUnavailable
	}
	if userID <= 0 || checkinID <= 0 || streakDays < 0 {
		return nil, ErrInvalidDelivery
	}
	prize, value, err := s.selectPrize(ctx)
	if err != nil {
		return nil, err
	}
	if prize == nil {
		return nil, nil
	}
	snapshot := Snapshot{
		PrizeID: prize.ID, PrizeName: prize.Name, Rarity: prize.Rarity,
		RewardType: prize.RewardType, RewardValue: value,
		SubscriptionID: prize.SubscriptionID, SubscriptionDays: prize.SubscriptionDays,
		StreakDays: streakDays,
	}
	if snapshot.RewardType == RewardTypeInvitationCode {
		if s.codes == nil {
			return nil, ErrUnavailable
		}
		code, codeErr := s.codes.GenerateInvitationCode(ctx)
		if codeErr != nil {
			return nil, fmt.Errorf("freeze invitation code: %w", codeErr)
		}
		snapshot.InvitationCode = code
	}
	if err := snapshot.Validate(); err != nil {
		return nil, fmt.Errorf("freeze blind-box snapshot: %w", err)
	}
	payload, err := jsonMarshalSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	prizeID := prize.ID
	delivery, err := s.outbox.Enqueue(ctx, CreateDelivery{
		SourceType: SourceCheckinBlindbox, SourceID: checkinID, UserID: userID,
		PrizeID: &prizeID, RewardSnapshot: payload, RewardType: snapshot.RewardType,
		RewardValue: snapshot.RewardValue, RuleVersion: CheckinBlindboxRuleV1,
		IdempotencyKey: deliveryKey(SourceCheckinBlindbox, checkinID),
	})
	if err != nil {
		return nil, fmt.Errorf("enqueue blind-box reward delivery: %w", err)
	}
	return &PreparedDelivery{
		Delivery: delivery,
		Result: DrawResult{
			PrizeName: snapshot.PrizeName, Rarity: snapshot.Rarity, RewardType: snapshot.RewardType,
			RewardValue: snapshot.RewardValue, SubscriptionDays: snapshot.SubscriptionDays,
		},
	}, nil
}

func (s *Service) selectPrize(ctx context.Context) (*Prize, float64, error) {
	prizes, err := s.prizes.ListEnabled(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list enabled blind-box prizes: %w", err)
	}
	if len(prizes) == 0 {
		return nil, 0, nil
	}
	const maxInt = int(^uint(0) >> 1)
	totalWeight := 0
	for i := range prizes {
		if err := prizes[i].Validate(); err != nil {
			return nil, 0, fmt.Errorf("validate blind-box prize %d: %w", prizes[i].ID, err)
		}
		if prizes[i].Weight > maxInt-totalWeight {
			return nil, 0, ErrInvalidPrize
		}
		totalWeight += prizes[i].Weight
	}
	roll, err := s.random.IntN(totalWeight)
	if err != nil || roll < 0 || roll >= totalWeight {
		if err != nil {
			return nil, 0, fmt.Errorf("select blind-box prize: %w", err)
		}
		return nil, 0, ErrInvalidPrize
	}
	selected := &prizes[0]
	for cumulative, index := 0, 0; index < len(prizes); index++ {
		cumulative += prizes[index].Weight
		if roll < cumulative {
			selected = &prizes[index]
			break
		}
	}
	value := selected.RewardValue
	if selected.RewardType == RewardTypeBalance && selected.RewardValueMax > selected.RewardValue {
		fraction, randomErr := s.random.Float64()
		if randomErr != nil {
			return nil, 0, fmt.Errorf("select blind-box reward value: %w", randomErr)
		}
		if fraction < 0 || fraction >= 1 || math.IsNaN(fraction) || math.IsInf(fraction, 0) {
			return nil, 0, ErrInvalidPrize
		}
		value = math.Round((selected.RewardValue+fraction*(selected.RewardValueMax-selected.RewardValue))*100) / 100
	}
	return selected, value, nil
}

func jsonMarshalSnapshot(snapshot Snapshot) ([]byte, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return nil, fmt.Errorf("marshal blind-box snapshot: %w", err)
	}
	return payload, nil
}

type cryptoRandomSource struct{}

func (cryptoRandomSource) IntN(n int) (int, error) {
	if n <= 0 {
		return 0, ErrInvalidPrize
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func (cryptoRandomSource) Float64() (float64, error) {
	const precision = int64(1) << 53
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(precision))
	if err != nil {
		return 0, err
	}
	return float64(value.Int64()) / float64(precision), nil
}
