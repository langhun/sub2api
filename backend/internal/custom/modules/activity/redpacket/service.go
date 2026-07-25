package redpacket

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

var (
	ErrUnavailable       = errors.New("activity red-packet service is unavailable")
	ErrDisabled          = errors.New("activity red packets are disabled")
	ErrInvalidAmount     = errors.New("invalid red-packet amount")
	ErrInvalidCount      = errors.New("invalid red-packet count")
	ErrInvalidType       = errors.New("invalid red-packet type")
	ErrAmountTooSmall    = errors.New("red-packet amount is too small")
	ErrInsufficient      = errors.New("insufficient balance")
	ErrNotFound          = errors.New("red packet not found")
	ErrExpired           = errors.New("red packet has expired")
	ErrExhausted         = errors.New("red packet has been fully claimed")
	ErrAlreadyClaimed    = errors.New("you have already claimed this red packet")
	ErrSelfClaim         = errors.New("cannot claim your own red packet")
	ErrDetailForbidden   = errors.New("red-packet detail is only available to participants")
	ErrInvalidPagination = errors.New("invalid red-packet pagination")
	errClaimConflict     = errors.New("red-packet claim already exists")
)

const (
	minimumClaimAmount = 0.01
	precision          = 1e8
)

// RandomSource makes random packet allocation testable while keeping the
// production implementation cryptographically secure.
type RandomSource interface {
	Int63n(max int64) (int64, error)
}

// Service is the module-owned implementation. It retains the legacy mutation
// ordering but obtains every external capability through narrow Activity ports.
type serviceRuntime struct {
	deps   Dependencies
	clock  Clock
	random RandomSource
}

func NewService(deps Dependencies) Service {
	clock := deps.Clock
	if clock == nil {
		clock = wallClock{}
	}
	random := deps.Random
	if random == nil {
		random = cryptoRandom{}
	}
	return &serviceRuntime{deps: deps, clock: clock, random: random}
}

func (s *serviceRuntime) Create(ctx context.Context, request CreateRequest) (*RedPacket, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	settings, err := s.deps.Settings.GetActivityRedPacketSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("read red-packet settings: %w", err)
	}
	if !settings.Enabled {
		return nil, ErrDisabled
	}
	if request.SenderID <= 0 || request.Count <= 0 || request.Count > settings.MaximumCount {
		return nil, ErrInvalidCount
	}
	amount := roundAmount(request.TotalAmount)
	if !validAmount(amount) {
		return nil, ErrInvalidAmount
	}
	if request.Type != TypeEqual && request.Type != TypeRandom {
		return nil, ErrInvalidType
	}
	if amount < float64(request.Count)*minimumClaimAmount {
		return nil, ErrAmountTooSmall
	}
	quote, err := s.deps.Fees.QuoteRedPacketFee(ctx, request.SenderID, amount)
	if err != nil {
		return nil, fmt.Errorf("quote red-packet fee: %w", err)
	}
	if !validNonNegativeAmount(quote.Rate) || !validNonNegativeAmount(quote.Amount) {
		return nil, fmt.Errorf("quote red-packet fee: %w", ErrInvalidAmount)
	}
	code, err := s.deps.Code.GenerateRedPacketCode(ctx)
	if err != nil || strings.TrimSpace(code) == "" {
		if err == nil {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("generate red-packet code: %w", err)
	}
	expireHours := settings.ExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}
	now := s.clock.Now()
	packet := &RedPacket{
		SenderID: request.SenderID, TotalAmount: amount, TotalCount: request.Count,
		RemainingAmount: amount, RemainingCount: request.Count, Type: request.Type,
		Fee: roundAmount(quote.Amount), FeeRate: quote.Rate, Code: code, Status: StatusActive,
		Memo: request.Memo, ExpiresAt: now.Add(time.Duration(expireHours) * time.Hour), CreatedAt: now,
	}
	gross := roundAmount(amount + packet.Fee)
	err = s.deps.Transactions.RunInTransaction(ctx, func(txCtx context.Context) error {
		ok, debitErr := s.deps.Balance.DebitIfSufficient(txCtx, contract.BalanceOperation{
			UserID: request.SenderID, Amount: gross, Reason: "activity_redpacket_create", IdempotencyKey: request.IdempotencyKey,
		})
		if debitErr != nil {
			return fmt.Errorf("debit red-packet sender: %w", debitErr)
		}
		if !ok {
			return ErrInsufficient
		}
		if createErr := s.deps.Repository.Create(txCtx, packet); createErr != nil {
			return fmt.Errorf("create red packet: %w", createErr)
		}
		return s.writeAudit(txCtx, request.SenderID, "activity_redpacket_create", amount, packet.ID, request.IdempotencyKey, now)
	})
	if err != nil {
		return nil, err
	}
	return packet, nil
}

func (s *serviceRuntime) Claim(ctx context.Context, request ClaimRequest) (*Claim, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	settings, err := s.deps.Settings.GetActivityRedPacketSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("read red-packet settings: %w", err)
	}
	if !settings.Enabled {
		return nil, ErrDisabled
	}
	if request.UserID <= 0 || strings.TrimSpace(request.Code) == "" {
		return nil, ErrNotFound
	}
	code := strings.TrimSpace(request.Code)
	packet, err := s.deps.Repository.FindByCode(ctx, code)
	if err != nil {
		return nil, ErrNotFound
	}
	if packet.SenderID == request.UserID {
		return nil, ErrSelfClaim
	}
	if packet.Status == StatusExpired || !s.clock.Now().Before(packet.ExpiresAt) {
		return nil, ErrExpired
	}
	if packet.Status != StatusActive || packet.RemainingCount <= 0 {
		return nil, ErrExhausted
	}

	var claim *Claim
	err = s.deps.Transactions.RunInTransaction(ctx, func(txCtx context.Context) error {
		locked, lockErr := s.deps.Repository.FindByCodeForUpdate(txCtx, code)
		if lockErr != nil {
			return ErrNotFound
		}
		if locked.SenderID == request.UserID {
			return ErrSelfClaim
		}
		now := s.clock.Now()
		if locked.Status == StatusExpired || !now.Before(locked.ExpiresAt) {
			return ErrExpired
		}
		if locked.Status != StatusActive || locked.RemainingCount <= 0 {
			return ErrExhausted
		}
		amount, amountErr := s.claimAmount(locked)
		if amountErr != nil {
			return fmt.Errorf("calculate red-packet claim: %w", amountErr)
		}
		if amount <= 0 {
			return ErrExhausted
		}
		updated, decrementErr := s.deps.Repository.DecrementClaim(txCtx, locked.ID, amount)
		if decrementErr != nil {
			return ErrExhausted
		}
		if creditErr := s.deps.Balance.Credit(txCtx, contract.BalanceOperation{
			UserID: request.UserID, Amount: amount, Reason: "activity_redpacket_claim", IdempotencyKey: request.IdempotencyKey,
		}); creditErr != nil {
			return fmt.Errorf("credit red-packet claim: %w", creditErr)
		}
		ledgerID, ledgerErr := s.deps.Ledger.RecordRedPacketClaim(txCtx, locked.ID, locked.SenderID, request.UserID, amount, now)
		if ledgerErr != nil {
			return fmt.Errorf("record red-packet claim ledger: %w", ledgerErr)
		}
		claim = &Claim{RedPacketID: locked.ID, UserID: request.UserID, Amount: amount, AuditID: &ledgerID, CreatedAt: now}
		if createErr := s.deps.Repository.CreateClaim(txCtx, claim); createErr != nil {
			if errors.Is(createErr, errClaimConflict) {
				return ErrAlreadyClaimed
			}
			return fmt.Errorf("create red-packet claim: %w", createErr)
		}
		if auditErr := s.writeAudit(txCtx, request.UserID, "activity_redpacket_claim", amount, locked.ID, request.IdempotencyKey, now); auditErr != nil {
			return auditErr
		}
		if updated.RemainingCount == 0 || updated.RemainingAmount <= 0 {
			return s.deps.Repository.MarkExhausted(txCtx, locked.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func (s *serviceRuntime) RefundExpired(ctx context.Context) (ExpiryRunResult, error) {
	if err := s.ready(); err != nil {
		return ExpiryRunResult{}, err
	}
	now := s.clock.Now()
	packets, err := s.deps.Repository.ListActiveExpired(ctx, now)
	if err != nil {
		return ExpiryRunResult{}, fmt.Errorf("list expired red packets: %w", err)
	}
	result := ExpiryRunResult{Refunds: make([]ExpiryRefund, 0, len(packets))}
	var errs []error
	for _, packet := range packets {
		packet := packet
		err := s.deps.Transactions.RunInTransaction(ctx, func(txCtx context.Context) error {
			remaining, refundErr := s.deps.Repository.ReturnRemainingIfExpired(txCtx, packet.ID, packet.SenderID, now)
			if refundErr != nil {
				return refundErr
			}
			if remaining <= 0 {
				return nil
			}
			if creditErr := s.deps.Balance.Credit(txCtx, contract.BalanceOperation{
				UserID: packet.SenderID, Amount: remaining, Reason: "activity_redpacket_expiry_refund", IdempotencyKey: fmt.Sprintf("redpacket-expiry-%d", packet.ID),
			}); creditErr != nil {
				return creditErr
			}
			if auditErr := s.writeAudit(txCtx, packet.SenderID, "activity_redpacket_expiry_refund", remaining, packet.ID, fmt.Sprintf("redpacket-expiry-%d", packet.ID), now); auditErr != nil {
				return auditErr
			}
			result.Refunds = append(result.Refunds, ExpiryRefund{RedPacketID: packet.ID, SenderID: packet.SenderID, ReturnedAmount: remaining, OccurredAt: now})
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("refund expired red packet %d: %w", packet.ID, err))
			continue
		}
		result.Processed++
	}
	return result, errors.Join(errs...)
}

func (s *serviceRuntime) Get(ctx context.Context, redPacketID int64) (*RedPacket, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, err
	}
	packet, err := s.deps.Repository.FindByID(ctx, redPacketID)
	if err != nil {
		return nil, ErrNotFound
	}
	return packet, nil
}

func (s *serviceRuntime) GetForParticipant(ctx context.Context, requesterID, redPacketID int64) (*RedPacket, []Claim, error) {
	packet, err := s.Get(ctx, redPacketID)
	if err != nil {
		return nil, nil, err
	}
	claims, err := s.deps.Repository.ListClaims(ctx, redPacketID)
	if err != nil {
		return nil, nil, fmt.Errorf("list red-packet claims: %w", err)
	}
	if packet.SenderID == requesterID {
		return packet, claims, nil
	}
	for _, claim := range claims {
		if claim.UserID == requesterID {
			return packet, []Claim{claim}, nil
		}
	}
	return nil, nil, ErrDetailForbidden
}

func (s *serviceRuntime) ListCreatedBy(ctx context.Context, senderID int64, page, pageSize int) ([]RedPacket, int, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, 0, err
	}
	if !validPage(page, pageSize) {
		return nil, 0, ErrInvalidPagination
	}
	return s.deps.Repository.ListCreatedBy(ctx, senderID, page, pageSize)
}

func (s *serviceRuntime) ListClaimedBy(ctx context.Context, userID int64, page, pageSize int) ([]RedPacket, int, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, 0, err
	}
	if !validPage(page, pageSize) {
		return nil, 0, ErrInvalidPagination
	}
	return s.deps.Repository.ListClaimedBy(ctx, userID, page, pageSize)
}

func (s *serviceRuntime) ListAll(ctx context.Context, page, pageSize int) ([]RedPacket, int, error) {
	if err := s.ready(); err != nil {
		return nil, 0, err
	}
	if !validPage(page, pageSize) {
		return nil, 0, ErrInvalidPagination
	}
	return s.deps.Repository.ListAll(ctx, page, pageSize)
}

func (s *serviceRuntime) enabled(ctx context.Context) error {
	if err := s.ready(); err != nil {
		return err
	}
	settings, err := s.deps.Settings.GetActivityRedPacketSettings(ctx)
	if err != nil {
		return fmt.Errorf("read red-packet settings: %w", err)
	}
	if !settings.Enabled {
		return ErrDisabled
	}
	return nil
}

func (s *serviceRuntime) ready() error {
	if s == nil {
		return ErrUnavailable
	}
	if err := s.deps.validateRuntime(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

func (s *serviceRuntime) writeAudit(ctx context.Context, userID int64, kind string, amount float64, packetID int64, key string, occurredAt time.Time) error {
	if s.deps.Audit == nil {
		return nil
	}
	if err := s.deps.Audit.WriteActivityAudit(ctx, contract.AuditEntry{
		UserID: userID, Type: kind, Amount: amount, ReferenceID: fmt.Sprintf("redpacket:%d", packetID), IdempotencyKey: key, OccurredAt: occurredAt,
	}); err != nil {
		return fmt.Errorf("write red-packet audit: %w", err)
	}
	return nil
}

func (s *serviceRuntime) claimAmount(packet *RedPacket) (float64, error) {
	if packet.RemainingCount <= 0 || packet.RemainingAmount <= 0 {
		return 0, nil
	}
	if packet.Type == TypeEqual {
		return roundAmount(packet.RemainingAmount / float64(packet.RemainingCount)), nil
	}
	if packet.Type != TypeRandom {
		return 0, ErrInvalidType
	}
	if packet.RemainingCount == 1 {
		return roundAmount(packet.RemainingAmount), nil
	}
	maxCents := int64(math.Floor((packet.RemainingAmount-float64(packet.RemainingCount-1)*minimumClaimAmount)*200 + 1e-9))
	if maxCents < 1 {
		maxCents = 1
	}
	value, err := s.random.Int63n(maxCents)
	if err != nil {
		return 0, err
	}
	if value < 0 || value >= maxCents {
		return 0, ErrInvalidAmount
	}
	amount := float64(value+1) / 100
	maximum := packet.RemainingAmount - float64(packet.RemainingCount-1)*minimumClaimAmount
	if amount > maximum {
		amount = maximum
	}
	return roundAmount(amount), nil
}

func validPage(page, pageSize int) bool { return page >= 1 && pageSize >= 1 && pageSize <= 100 }
func validAmount(value float64) bool    { return validNonNegativeAmount(value) && value > 0 }
func validNonNegativeAmount(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
func roundAmount(value float64) float64 { return math.Round(value*precision) / precision }

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

type cryptoRandom struct{}

func (cryptoRandom) Int63n(max int64) (int64, error) {
	if max <= 0 {
		return 0, ErrInvalidAmount
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return value.Int64(), nil
}
