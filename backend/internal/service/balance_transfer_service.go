package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrTransferDisabled             = infraerrors.Forbidden("TRANSFER_DISABLED", "transfer feature is disabled")
	ErrTransferSelf                 = infraerrors.BadRequest("TRANSFER_SELF", "cannot transfer to yourself")
	ErrTransferAmountInvalid        = infraerrors.BadRequest("TRANSFER_AMOUNT_INVALID", "invalid transfer amount")
	ErrTransferInsufficient         = infraerrors.BadRequest("TRANSFER_INSUFFICIENT", "insufficient balance")
	ErrTransferDailyLimit           = infraerrors.Forbidden("TRANSFER_DAILY_LIMIT", "daily transfer limit exceeded")
	ErrTransferDailyCount           = infraerrors.Forbidden("TRANSFER_DAILY_COUNT", "daily transfer count limit exceeded")
	ErrTransferReceiverNotFound     = infraerrors.NotFound("RECEIVER_NOT_FOUND", "receiver not found")
	ErrTransferReceiverQueryInvalid = infraerrors.BadRequest("RECEIVER_QUERY_INVALID", "receiver query must be a positive user ID or at least 2 characters")
	ErrTransferReceiverAmbiguous    = infraerrors.Conflict("RECEIVER_AMBIGUOUS", "receiver query matches multiple users; use a user ID or exact email")
	ErrTransferLeaderboardDisabled  = infraerrors.NotFound("TRANSFER_LEADERBOARD_DISABLED", "transfer leaderboard is disabled")
	ErrTransferNotFound             = infraerrors.NotFound("TRANSFER_NOT_FOUND", "transfer not found")
	ErrTransferAlreadyFrozen        = infraerrors.BadRequest("TRANSFER_ALREADY_FROZEN", "transfer already frozen")
	ErrTransferAlreadyRevoked       = infraerrors.BadRequest("TRANSFER_ALREADY_REVOKED", "transfer already revoked")
	ErrRedPacketDisabled            = infraerrors.Forbidden("REDPACKET_DISABLED", "red packet feature is disabled")
	ErrRedPacketNotFound            = infraerrors.NotFound("REDPACKET_NOT_FOUND", "red packet not found")
	ErrRedPacketExpired             = infraerrors.BadRequest("REDPACKET_EXPIRED", "red packet has expired")
	ErrRedPacketExhausted           = infraerrors.BadRequest("REDPACKET_EXHAUSTED", "red packet has been fully claimed")
	ErrRedPacketAlreadyClaimed      = infraerrors.BadRequest("REDPACKET_ALREADY_CLAIMED", "you have already claimed this red packet")
	ErrRedPacketSelfClaim           = infraerrors.BadRequest("REDPACKET_SELF_CLAIM", "cannot claim your own red packet")
	ErrRedPacketCountInvalid        = infraerrors.BadRequest("REDPACKET_COUNT_INVALID", "invalid red packet count")
	ErrRedPacketDetailForbidden     = infraerrors.Forbidden("REDPACKET_DETAIL_FORBIDDEN", "red packet detail is only available to its sender or claimants")
)

type transferReceiverResolver interface {
	ResolveActiveTransferReceiver(ctx context.Context, query string, numericID *int64) (*User, error)
}

type BalanceTransferService struct {
	transferRepo        BalanceTransferRepository
	redPacketRepo       BalanceRedPacketRepository
	userRepo            UserRepository
	settingService      *SettingService
	subscriptionService *SubscriptionService
}

type nonRechargeBalanceUpdater interface {
	UpdateBalanceWithoutRecharge(ctx context.Context, id int64, amount float64) error
}

type nonnegativeNonRechargeBalanceUpdater interface {
	UpdateBalanceWithoutRechargeIfNonnegative(ctx context.Context, id int64, amount float64) (bool, error)
}

func updateBalanceWithoutRechargeIfNonnegative(ctx context.Context, repo UserRepository, userID int64, amount float64) (bool, error) {
	if updater, ok := repo.(nonnegativeNonRechargeBalanceUpdater); ok {
		return updater.UpdateBalanceWithoutRechargeIfNonnegative(ctx, userID, amount)
	}
	// Compatibility path for test doubles and alternate repositories. The
	// production repository implements the atomic conditional update above.
	user, err := repo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || user.Balance+amount < 0 {
		return false, nil
	}
	return true, updateBalanceWithoutRecharge(ctx, repo, userID, amount)
}

func updateBalanceWithoutRecharge(ctx context.Context, repo UserRepository, userID int64, amount float64) error {
	if updater, ok := repo.(nonRechargeBalanceUpdater); ok {
		return updater.UpdateBalanceWithoutRecharge(ctx, userID, amount)
	}
	return repo.UpdateBalance(ctx, userID, amount)
}

func NewBalanceTransferService(
	transferRepo BalanceTransferRepository,
	redPacketRepo BalanceRedPacketRepository,
	userRepo UserRepository,
	settingService *SettingService,
	subscriptionService *SubscriptionService,
) *BalanceTransferService {
	return &BalanceTransferService{
		transferRepo:        transferRepo,
		redPacketRepo:       redPacketRepo,
		userRepo:            userRepo,
		settingService:      settingService,
		subscriptionService: subscriptionService,
	}
}

func (s *BalanceTransferService) transferFeeRate(ctx context.Context, userID int64, cfg *TransferSettings) float64 {
	if !cfg.VIPFeeExempt || s.subscriptionService == nil {
		return cfg.FeeRate
	}
	subscriptions, err := s.subscriptionService.ListActiveUserSubscriptions(ctx, userID)
	if err == nil && len(subscriptions) > 0 {
		return 0
	}
	return cfg.FeeRate
}

func (s *BalanceTransferService) getTransferSettings(ctx context.Context) *TransferSettings {
	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return &TransferSettings{}
	}
	return &TransferSettings{
		Enabled:              settings.TransferEnabled,
		FeeRate:              settings.TransferFeeRate,
		MinAmount:            settings.TransferMinAmount,
		MaxAmount:            settings.TransferMaxAmount,
		DailyLimit:           settings.TransferDailyLimit,
		DailyCountLimit:      settings.TransferDailyCountLimit,
		VIPFeeExempt:         settings.TransferVIPFeeExempt,
		RedPacketEnabled:     settings.RedPacketEnabled,
		RedPacketMaxCount:    settings.RedPacketMaxCount,
		RedPacketExpireHours: settings.RedPacketExpireHours,
	}
}

func (s *BalanceTransferService) Transfer(ctx context.Context, senderID, receiverID int64, amount float64, memo *string) (*BalanceTransferRecord, error) {
	cfg := s.getTransferSettings(ctx)
	if !cfg.Enabled {
		return nil, ErrTransferDisabled
	}
	if senderID == receiverID {
		return nil, ErrTransferSelf
	}
	amount = math.Round(amount*1e8) / 1e8
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < cfg.MinAmount || (cfg.MaxAmount > 0 && amount > cfg.MaxAmount) || amount <= 0 {
		return nil, ErrTransferAmountInvalid
	}
	receiver, err := s.userRepo.GetByID(ctx, receiverID)
	if err != nil {
		return nil, ErrTransferReceiverNotFound
	}
	if receiver == nil {
		return nil, ErrTransferReceiverNotFound
	}
	feeRate := s.transferFeeRate(ctx, senderID, cfg)
	fee := math.Round(amount*feeRate*1e8) / 1e8
	if fee < 0 {
		fee = 0
	}
	grossAmount := math.Round((amount+fee)*1e8) / 1e8
	var record *BalanceTransferRecord
	if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Serializing on the sender row makes the daily limits authoritative for
		// concurrent requests from the same account.
		if err := s.transferRepo.LockUser(txCtx, senderID); err != nil {
			return fmt.Errorf("lock sender: %w", err)
		}
		dailyTotal, dailyCount, err := s.transferRepo.GetDailyTransferTotal(txCtx, senderID)
		if err != nil {
			return fmt.Errorf("check daily limit: %w", err)
		}
		if cfg.DailyLimit > 0 && dailyTotal+grossAmount > cfg.DailyLimit {
			return ErrTransferDailyLimit
		}
		if cfg.DailyCountLimit > 0 && dailyCount >= cfg.DailyCountLimit {
			return ErrTransferDailyCount
		}
		ok, err := s.transferRepo.DeductBalanceIfSufficient(txCtx, senderID, grossAmount)
		if err != nil {
			return fmt.Errorf("deduct sender balance: %w", err)
		}
		if !ok {
			return ErrTransferInsufficient
		}
		if err := updateBalanceWithoutRecharge(txCtx, s.userRepo, receiverID, amount); err != nil {
			return fmt.Errorf("credit receiver balance: %w", err)
		}
		record = &BalanceTransferRecord{
			SenderID:     senderID,
			ReceiverID:   receiverID,
			Amount:       amount,
			Fee:          fee,
			FeeRate:      feeRate,
			GrossAmount:  grossAmount,
			TransferType: "direct",
			Status:       "completed",
			Memo:         memo,
			CreatedAt:    time.Now(),
		}
		return s.transferRepo.Create(txCtx, record)
	}); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *BalanceTransferService) ValidateTransfer(ctx context.Context, senderID, receiverID int64, amount float64) (*TransferValidation, error) {
	cfg := s.getTransferSettings(ctx)
	if !cfg.Enabled {
		return nil, ErrTransferDisabled
	}
	if senderID == receiverID {
		return nil, ErrTransferSelf
	}
	amount = math.Round(amount*1e8) / 1e8
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount < cfg.MinAmount || (cfg.MaxAmount > 0 && amount > cfg.MaxAmount) || amount <= 0 {
		return nil, ErrTransferAmountInvalid
	}
	receiver, err := s.userRepo.GetByID(ctx, receiverID)
	if err != nil || receiver == nil {
		return nil, ErrTransferReceiverNotFound
	}
	dailyTotal, dailyCount, err := s.transferRepo.GetDailyTransferTotal(ctx, senderID)
	if err != nil {
		return nil, fmt.Errorf("check daily limit: %w", err)
	}
	feeRate := s.transferFeeRate(ctx, senderID, cfg)
	fee := math.Max(0, math.Round(amount*feeRate*1e8)/1e8)
	grossAmount := math.Round((amount+fee)*1e8) / 1e8
	if cfg.DailyLimit > 0 && dailyTotal+grossAmount > cfg.DailyLimit {
		return nil, ErrTransferDailyLimit
	}
	if cfg.DailyCountLimit > 0 && dailyCount >= cfg.DailyCountLimit {
		return nil, ErrTransferDailyCount
	}
	remainingAmount := float64(0)
	if cfg.DailyLimit > 0 {
		remainingAmount = math.Max(0, math.Round((cfg.DailyLimit-dailyTotal)*1e8)/1e8)
	}
	remainingCount := 0
	if cfg.DailyCountLimit > 0 {
		remainingCount = max(0, cfg.DailyCountLimit-dailyCount)
	}
	return &TransferValidation{
		Fee:                  fee,
		FeeRate:              feeRate,
		GrossAmount:          grossAmount,
		ReceiverID:           receiverID,
		ReceiverDisplay:      transferReceiverDisplay(receiver),
		DailyRemainingAmount: remainingAmount,
		DailyRemainingCount:  remainingCount,
	}, nil
}

func transferReceiverDisplay(user *User) string {
	return UserDisplayName(user.Username, user.Email, user.ID)
}

func (s *BalanceTransferService) ResolveReceiver(ctx context.Context, requesterID int64, rawQuery string) (*TransferReceiver, error) {
	if !s.getTransferSettings(ctx).Enabled {
		return nil, ErrTransferDisabled
	}
	query := strings.TrimSpace(rawQuery)
	var numericID *int64
	if query != "" {
		if id, err := strconv.ParseInt(query, 10, 64); err == nil && id > 0 {
			numericID = &id
		} else if isASCIIDigits(query) {
			return nil, ErrTransferReceiverQueryInvalid
		}
	}
	if numericID == nil && utf8.RuneCountInString(query) < 2 {
		return nil, ErrTransferReceiverQueryInvalid
	}
	resolver, ok := s.userRepo.(transferReceiverResolver)
	if !ok {
		return nil, ErrTransferReceiverNotFound
	}
	user, err := resolver.ResolveActiveTransferReceiver(ctx, query, numericID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.ID == requesterID || user.Status != StatusActive {
		return nil, ErrTransferReceiverNotFound
	}
	return &TransferReceiver{ReceiverID: user.ID, ReceiverDisplay: transferReceiverDisplay(user)}, nil
}

func isASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (s *BalanceTransferService) GetHistory(ctx context.Context, userID int64, role string, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	if !s.getTransferSettings(ctx).Enabled {
		return nil, 0, ErrTransferDisabled
	}
	return s.transferRepo.ListByUser(ctx, userID, role, page, pageSize)
}

func (s *BalanceTransferService) GetAllTransfers(ctx context.Context, filter *TransferFilter, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	return s.transferRepo.ListAll(ctx, filter, page, pageSize)
}

func (s *BalanceTransferService) FreezeTransfer(ctx context.Context, adminID, transferID int64) error {
	return s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		record, err := s.transferRepo.GetByIDForUpdate(txCtx, transferID)
		if err != nil {
			return ErrTransferNotFound
		}
		if record.Status == "frozen" {
			return ErrTransferAlreadyFrozen
		}
		if record.Status == "revoked" {
			return ErrTransferAlreadyRevoked
		}
		now := time.Now()
		return s.transferRepo.UpdateStatus(txCtx, transferID, "frozen", &now, &adminID, nil)
	})
}

func (s *BalanceTransferService) RevokeTransfer(ctx context.Context, adminID, transferID int64, reason string) error {
	return s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		record, err := s.transferRepo.GetByIDForUpdate(txCtx, transferID)
		if err != nil {
			return ErrTransferNotFound
		}
		// Revocation is idempotent: a retry observes the locked final state and
		// must not refund the sender twice.
		if record.Status == "revoked" {
			return nil
		}
		ok, err := s.transferRepo.DeductBalanceIfSufficient(txCtx, record.ReceiverID, record.Amount)
		if err != nil {
			return fmt.Errorf("deduct receiver balance: %w", err)
		}
		if !ok {
			return ErrTransferInsufficient
		}
		// Batch distribution mints an administrator-authorized reward and never
		// debits the admin. Revocation only claws the reward back; refunding the
		// synthetic sender would create new spendable balance.
		if record.TransferType != "batch" {
			if err := updateBalanceWithoutRecharge(txCtx, s.userRepo, record.SenderID, record.GrossAmount); err != nil {
				return fmt.Errorf("return sender balance: %w", err)
			}
		}
		return s.transferRepo.UpdateStatus(txCtx, transferID, "revoked", record.FrozenAt, &adminID, &reason)
	})
}

func (s *BalanceTransferService) BatchDistribute(ctx context.Context, adminID int64, targets []BatchDistributeTarget, memo *string) ([]*BalanceTransferRecord, error) {
	var records []*BalanceTransferRecord
	err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		for _, t := range targets {
			if t.Amount <= 0 || t.UserID <= 0 {
				continue
			}
			if _, err := s.userRepo.GetByID(txCtx, t.UserID); err != nil {
				continue
			}
			if err := updateBalanceWithoutRecharge(txCtx, s.userRepo, t.UserID, t.Amount); err != nil {
				return fmt.Errorf("update balance for user %d: %w", t.UserID, err)
			}
			record := &BalanceTransferRecord{
				SenderID:     adminID,
				ReceiverID:   t.UserID,
				Amount:       t.Amount,
				Fee:          0,
				FeeRate:      0,
				GrossAmount:  t.Amount,
				TransferType: "batch",
				Status:       "completed",
				Memo:         memo,
				CreatedAt:    time.Now(),
			}
			if err := s.transferRepo.Create(txCtx, record); err != nil {
				return fmt.Errorf("create transfer record for user %d: %w", t.UserID, err)
			}
			records = append(records, record)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (s *BalanceTransferService) GetFeeStats(ctx context.Context, startTime, endTime time.Time) ([]*DailyFeeStat, error) {
	return s.transferRepo.GetFeeStats(ctx, startTime, endTime)
}

func (s *BalanceTransferService) GetLeaderboard(ctx context.Context, period string, limit int) ([]*TransferRankEntry, error) {
	if !s.getTransferSettings(ctx).Enabled {
		return nil, ErrTransferDisabled
	}
	leaderboardSettings := s.settingService.GetLeaderboardSettings(ctx)
	if !leaderboardSettings.LeaderboardEnabled || !leaderboardSettings.LeaderboardTransferEnabled {
		return nil, ErrTransferLeaderboardDisabled
	}
	now := time.Now()
	var start time.Time
	switch period {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	default:
		start = now.AddDate(0, 0, -1)
	}
	return s.transferRepo.GetLeaderboard(ctx, start, now, limit, "amount")
}

type BatchDistributeTarget struct {
	UserID int64   `json:"user_id"`
	Amount float64 `json:"amount"`
}

func (s *BalanceTransferService) CreateRedPacket(ctx context.Context, senderID int64, totalAmount float64, count int, redPacketType string, memo *string) (*RedPacketRecord, error) {
	cfg := s.getTransferSettings(ctx)
	if !cfg.RedPacketEnabled {
		return nil, ErrRedPacketDisabled
	}
	if count <= 0 || count > cfg.RedPacketMaxCount {
		return nil, ErrRedPacketCountInvalid
	}
	totalAmount = math.Round(totalAmount*1e8) / 1e8
	if math.IsNaN(totalAmount) || math.IsInf(totalAmount, 0) || totalAmount <= 0 {
		return nil, ErrTransferAmountInvalid
	}
	if redPacketType != "equal" && redPacketType != "random" {
		return nil, infraerrors.BadRequest("REDPACKET_TYPE_INVALID", "red packet type must be equal or random")
	}
	minRequired := float64(count) * 0.01
	if totalAmount < minRequired {
		return nil, infraerrors.BadRequest("REDPACKET_AMOUNT_TOO_SMALL", fmt.Sprintf("minimum amount for %d packets is %.2f", count, minRequired))
	}
	feeRate := s.transferFeeRate(ctx, senderID, cfg)
	fee := math.Round(totalAmount*feeRate*1e8) / 1e8
	grossAmount := totalAmount + fee
	code, err := s.generateRedPacketCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}
	expireHours := cfg.RedPacketExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}
	var rp *RedPacketRecord
	if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		ok, err := s.transferRepo.DeductBalanceIfSufficient(txCtx, senderID, grossAmount)
		if err != nil {
			return fmt.Errorf("deduct sender balance: %w", err)
		}
		if !ok {
			return ErrTransferInsufficient
		}
		rp = &RedPacketRecord{
			SenderID:        senderID,
			TotalAmount:     totalAmount,
			TotalCount:      count,
			RemainingAmount: totalAmount,
			RemainingCount:  count,
			RedPacketType:   redPacketType,
			Fee:             fee,
			FeeRate:         feeRate,
			Code:            code,
			Status:          "active",
			Memo:            memo,
			ExpireAt:        time.Now().Add(time.Duration(expireHours) * time.Hour),
			CreatedAt:       time.Now(),
		}
		return s.redPacketRepo.Create(txCtx, rp)
	}); err != nil {
		return nil, err
	}
	return rp, nil
}

func (s *BalanceTransferService) ClaimRedPacket(ctx context.Context, userID int64, code string) (*RedPacketClaimRecord, error) {
	cfg := s.getTransferSettings(ctx)
	if !cfg.RedPacketEnabled {
		return nil, ErrRedPacketDisabled
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrRedPacketNotFound
	}
	rp, err := s.redPacketRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, ErrRedPacketNotFound
	}
	if rp.SenderID == userID {
		return nil, ErrRedPacketSelfClaim
	}
	if rp.Status != "active" {
		if rp.Status == "expired" {
			return nil, ErrRedPacketExpired
		}
		return nil, ErrRedPacketExhausted
	}
	if time.Now().After(rp.ExpireAt) {
		return nil, ErrRedPacketExpired
	}
	var claimRecord *RedPacketClaimRecord
	if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		locked, err := s.redPacketRepo.GetByCodeForUpdate(txCtx, code)
		if err != nil {
			return ErrRedPacketNotFound
		}
		if locked.SenderID == userID {
			return ErrRedPacketSelfClaim
		}
		if locked.Status == "expired" || !time.Now().Before(locked.ExpireAt) {
			return ErrRedPacketExpired
		}
		if locked.Status != "active" || locked.RemainingCount <= 0 {
			return ErrRedPacketExhausted
		}
		amount, err := s.calculateClaimAmount(locked)
		if err != nil {
			return fmt.Errorf("calculate red packet claim: %w", err)
		}
		if amount <= 0 {
			return ErrRedPacketExhausted
		}
		updated, err := s.redPacketRepo.DecrementClaim(txCtx, locked.ID, amount)
		if err != nil {
			return ErrRedPacketExhausted
		}
		if err := updateBalanceWithoutRecharge(txCtx, s.userRepo, userID, amount); err != nil {
			return fmt.Errorf("credit balance: %w", err)
		}
		claimRecord = &RedPacketClaimRecord{
			RedPacketID: locked.ID,
			UserID:      userID,
			Amount:      amount,
			CreatedAt:   time.Now(),
		}
		transferRecord := &BalanceTransferRecord{
			SenderID:     locked.SenderID,
			ReceiverID:   userID,
			Amount:       amount,
			Fee:          0,
			FeeRate:      0,
			GrossAmount:  amount,
			TransferType: "redpacket",
			Status:       "completed",
			RedpacketID:  &locked.ID,
			CreatedAt:    time.Now(),
		}
		if err := s.transferRepo.Create(txCtx, transferRecord); err != nil {
			return fmt.Errorf("create transfer record: %w", err)
		}
		claimRecord.TransferID = &transferRecord.ID
		if err := s.redPacketRepo.CreateClaim(txCtx, claimRecord); err != nil {
			return fmt.Errorf("create claim record: %w", err)
		}
		if updated.RemainingCount == 0 || updated.RemainingAmount <= 0 {
			return s.redPacketRepo.MarkExhausted(txCtx, locked.ID)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return claimRecord, nil
}

func (s *BalanceTransferService) GetRedPacketDetail(ctx context.Context, redPacketID int64) (*RedPacketRecord, []*RedPacketClaimRecord, error) {
	if !s.getTransferSettings(ctx).RedPacketEnabled {
		return nil, nil, ErrRedPacketDisabled
	}
	rp, err := s.redPacketRepo.GetByID(ctx, redPacketID)
	if err != nil {
		return nil, nil, ErrRedPacketNotFound
	}
	claims, err := s.redPacketRepo.GetClaims(ctx, redPacketID)
	if err != nil {
		return nil, nil, fmt.Errorf("get red packet claims: %w", err)
	}
	return rp, claims, nil
}

// GetRedPacketDetailForUser limits claim identity and amounts to participants.
// Administrative callers can use GetRedPacketDetail after enforcing admin auth.
func (s *BalanceTransferService) GetRedPacketDetailForUser(ctx context.Context, requesterID, redPacketID int64) (*RedPacketRecord, []*RedPacketClaimRecord, error) {
	rp, claims, err := s.GetRedPacketDetail(ctx, redPacketID)
	if err != nil {
		return nil, nil, err
	}
	if rp.SenderID == requesterID {
		return rp, claims, nil
	}
	for _, claim := range claims {
		if claim.UserID == requesterID {
			return rp, []*RedPacketClaimRecord{claim}, nil
		}
	}
	return nil, nil, ErrRedPacketDetailForbidden
}

func (s *BalanceTransferService) GetMyRedPackets(ctx context.Context, userID int64, role string, page, pageSize int) ([]*RedPacketRecord, int, error) {
	if !s.getTransferSettings(ctx).RedPacketEnabled {
		return nil, 0, ErrRedPacketDisabled
	}
	if role == "received" {
		return s.redPacketRepo.ListClaimedByUser(ctx, userID, page, pageSize)
	}
	return s.redPacketRepo.ListBySender(ctx, userID, page, pageSize)
}

func (s *BalanceTransferService) ExpireRedPackets(ctx context.Context) error {
	rps, err := s.redPacketRepo.ListActiveExpired(ctx)
	if err != nil {
		return err
	}
	var expireErrors []error
	for _, rp := range rps {
		if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
			remaining, err := s.redPacketRepo.ReturnRemaining(txCtx, rp.ID, rp.SenderID)
			if err != nil {
				return err
			}
			if remaining > 0 {
				return updateBalanceWithoutRecharge(txCtx, s.userRepo, rp.SenderID, remaining)
			}
			return nil
		}); err != nil {
			expireErrors = append(expireErrors, fmt.Errorf("expire red packet %d: %w", rp.ID, err))
		}
	}
	return errors.Join(expireErrors...)
}

func (s *BalanceTransferService) GetAllRedPackets(ctx context.Context, page, pageSize int) ([]*RedPacketRecord, int, error) {
	return s.redPacketRepo.ListAll(ctx, page, pageSize)
}

func (s *BalanceTransferService) calculateClaimAmount(rp *RedPacketRecord) (float64, error) {
	if rp.RemainingCount <= 0 || rp.RemainingAmount <= 0 {
		return 0, nil
	}
	if rp.RedPacketType == "equal" {
		return math.Round(rp.RemainingAmount/float64(rp.RemainingCount)*1e8) / 1e8, nil
	}
	if rp.RemainingCount == 1 {
		return math.Round(rp.RemainingAmount*1e8) / 1e8, nil
	}
	maxCents := int64(math.Floor((rp.RemainingAmount-float64(rp.RemainingCount-1)*0.01)*2*100 + 1e-9))
	if maxCents < 1 {
		maxCents = 1
	}
	randomCent, err := rand.Int(rand.Reader, big.NewInt(maxCents))
	if err != nil {
		return 0, err
	}
	amount := float64(randomCent.Int64()+1) / 100
	if amount > rp.RemainingAmount-float64(rp.RemainingCount-1)*0.01 {
		amount = rp.RemainingAmount - float64(rp.RemainingCount-1)*0.01
	}
	return math.Round(amount*1e8) / 1e8, nil
}

func generateRedPacketCode() (string, error) {
	return DefaultCodeFormatSettings().RedPacket.Generate()
}

func (s *BalanceTransferService) generateRedPacketCode(ctx context.Context) (string, error) {
	if s.settingService == nil {
		return generateRedPacketCode()
	}
	return s.settingService.GetCodeFormatSettings(ctx).RedPacket.Generate()
}

func (s *BalanceTransferService) GetTransferStats(ctx context.Context, userID int64) (sent float64, received float64, feePaid float64, err error) {
	if !s.getTransferSettings(ctx).Enabled {
		return 0, 0, 0, ErrTransferDisabled
	}
	return s.transferRepo.GetUserTransferStats(ctx, userID)
}
