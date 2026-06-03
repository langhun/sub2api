package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/shopspring/decimal"
)

var (
	ErrTransferDisabled         = infraerrors.Forbidden("TRANSFER_DISABLED", "transfer feature is disabled")
	ErrTransferSelf             = infraerrors.BadRequest("TRANSFER_SELF", "cannot transfer to yourself")
	ErrTransferAmountInvalid    = infraerrors.BadRequest("TRANSFER_AMOUNT_INVALID", "invalid transfer amount")
	ErrTransferInsufficient     = infraerrors.BadRequest("TRANSFER_INSUFFICIENT", "insufficient balance")
	ErrTransferDailyLimit       = infraerrors.Forbidden("TRANSFER_DAILY_LIMIT", "daily transfer limit exceeded")
	ErrTransferDailyCount       = infraerrors.Forbidden("TRANSFER_DAILY_COUNT", "daily transfer count limit exceeded")
	ErrTransferReceiverNotFound = infraerrors.NotFound("RECEIVER_NOT_FOUND", "receiver not found")
	ErrTransferNotFound         = infraerrors.NotFound("TRANSFER_NOT_FOUND", "transfer not found")
	ErrTransferAlreadyFrozen    = infraerrors.BadRequest("TRANSFER_ALREADY_FROZEN", "transfer already frozen")
	ErrTransferAlreadyRevoked   = infraerrors.BadRequest("TRANSFER_ALREADY_REVOKED", "transfer already revoked")
	ErrRedPacketDisabled        = infraerrors.Forbidden("REDPACKET_DISABLED", "red packet feature is disabled")
	ErrRedPacketNotFound        = infraerrors.NotFound("REDPACKET_NOT_FOUND", "red packet not found")
	ErrRedPacketExpired         = infraerrors.BadRequest("REDPACKET_EXPIRED", "red packet has expired")
	ErrRedPacketExhausted       = infraerrors.BadRequest("REDPACKET_EXHAUSTED", "red packet has been fully claimed")
	ErrRedPacketAlreadyClaimed  = infraerrors.BadRequest("REDPACKET_ALREADY_CLAIMED", "you have already claimed this red packet")
	ErrRedPacketSelfClaim       = infraerrors.BadRequest("REDPACKET_SELF_CLAIM", "cannot claim your own red packet")
	ErrRedPacketCountInvalid    = infraerrors.BadRequest("REDPACKET_COUNT_INVALID", "invalid red packet count")
)

const (
	userSearchMinQueryRunes = 2
	userSearchMaxQueryRunes = 64
	userSearchLimit         = 10
)

type BalanceTransferService struct {
	transferRepo   BalanceTransferRepository
	redPacketRepo  BalanceRedPacketRepository
	userRepo       UserRepository
	settingService *SettingService
	claimLocks     sync.Map
}

type balanceTransferTxGuard interface {
	GetByIDForUpdate(ctx context.Context, id int64) (*BalanceTransferRecord, error)
}

type balanceTransferUserSearchRepository interface {
	SearchForBalanceTransfer(ctx context.Context, query string, limit int) ([]User, error)
}

type balanceTransferUserEmailBatchRepository interface {
	GetEmailsByIDs(ctx context.Context, userIDs []int64) (map[int64]string, error)
}

func NewBalanceTransferService(
	transferRepo BalanceTransferRepository,
	redPacketRepo BalanceRedPacketRepository,
	userRepo UserRepository,
	settingService *SettingService,
) *BalanceTransferService {
	return &BalanceTransferService{
		transferRepo:   transferRepo,
		redPacketRepo:  redPacketRepo,
		userRepo:       userRepo,
		settingService: settingService,
	}
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
	if amount < cfg.MinAmount || (cfg.MaxAmount > 0 && amount > cfg.MaxAmount) || amount <= 0 {
		return nil, ErrTransferAmountInvalid
	}
	receiver, err := s.userRepo.GetByID(ctx, receiverID)
	if err != nil {
		return nil, ErrTransferReceiverNotFound
	}
	if receiver == nil {
		return nil, ErrTransferReceiverNotFound
	}
	feeRate := cfg.FeeRate
	fee := math.Round(amount*feeRate*1e8) / 1e8
	if fee < 0 {
		fee = 0
	}
	grossAmount := amount + fee
	var record *BalanceTransferRecord
	if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		senderBalance, err := s.lockBankAvailableCapacity(txCtx, senderID)
		if err != nil {
			return fmt.Errorf("lock sender balance: %w", err)
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
		if senderBalance < grossAmount {
			return ErrTransferInsufficient
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
		if err := s.transferRepo.Create(txCtx, record); err != nil {
			return err
		}
		return s.applyDirectTransferBankEntriesInTx(txCtx, senderID, receiverID, record)
	}); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *BalanceTransferService) applyDirectTransferBankEntriesInTx(
	ctx context.Context,
	senderID int64,
	receiverID int64,
	record *BalanceTransferRecord,
) error {
	if record == nil || record.ID <= 0 {
		return infraerrors.InternalServer("TRANSFER_RECORD_MISSING", "transfer record is missing")
	}
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return err
	}
	amount := decimal.NewFromFloat(record.Amount).RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrTransferAmountInvalid
	}

	bank := NewBankService(nil)
	referenceID := fmt.Sprintf("%d", record.ID)
	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           senderID,
		Amount:           amount,
		Type:             BankTxTypeTransferOut,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("余额转账给用户 %d", receiverID),
		IdempotencyScope: "balance_transfer",
		IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:sender:amount", record.ID),
		ReferenceType:    "balance_transfer",
		ReferenceID:      referenceID,
		Metadata: map[string]any{
			"receiver_id":   receiverID,
			"sender_id":     senderID,
			"transfer_id":   record.ID,
			"transfer_type": record.TransferType,
		},
	}); err != nil {
		return fmt.Errorf("debit sender bank balance: %w", err)
	}

	if record.Fee > 0 {
		fee := decimal.NewFromFloat(record.Fee).RoundBank(18)
		if fee.GreaterThan(decimal.Zero) {
			if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
				UserID:           senderID,
				Amount:           fee,
				Type:             BankTxTypeWithdraw,
				BusinessModule:   BankBusinessModuleTransfer,
				Description:      fmt.Sprintf("余额转账手续费 %d", record.ID),
				IdempotencyScope: "balance_transfer",
				IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:sender:fee", record.ID),
				ReferenceType:    "balance_transfer",
				ReferenceID:      referenceID,
				Metadata: map[string]any{
					"fee_rate":      record.FeeRate,
					"receiver_id":   receiverID,
					"sender_id":     senderID,
					"transfer_id":   record.ID,
					"transfer_type": record.TransferType,
				},
			}); err != nil {
				return fmt.Errorf("debit sender transfer fee: %w", err)
			}
		}
	}

	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           receiverID,
		Amount:           amount,
		Type:             BankTxTypeTransferIn,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("收到用户 %d 的余额转账", senderID),
		IdempotencyScope: "balance_transfer",
		IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:receiver:amount", record.ID),
		ReferenceType:    "balance_transfer",
		ReferenceID:      referenceID,
		Metadata: map[string]any{
			"receiver_id":   receiverID,
			"sender_id":     senderID,
			"transfer_id":   record.ID,
			"transfer_type": record.TransferType,
		},
	}); err != nil {
		return fmt.Errorf("credit receiver bank balance: %w", err)
	}
	return nil
}

func (s *BalanceTransferService) ValidateTransfer(ctx context.Context, senderID, receiverID int64, amount float64) (fee float64, feeRate float64, err error) {
	cfg := s.getTransferSettings(ctx)
	if !cfg.Enabled {
		return 0, 0, ErrTransferDisabled
	}
	if senderID == receiverID {
		return 0, 0, ErrTransferSelf
	}
	amount = math.Round(amount*1e8) / 1e8
	if amount < cfg.MinAmount || (cfg.MaxAmount > 0 && amount > cfg.MaxAmount) || amount <= 0 {
		return 0, 0, ErrTransferAmountInvalid
	}
	feeRate = cfg.FeeRate
	fee = math.Round(amount*feeRate*1e8) / 1e8
	return fee, feeRate, nil
}

func (s *BalanceTransferService) GetHistory(ctx context.Context, userID int64, role string, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	return s.transferRepo.ListByUserExcludeType(ctx, userID, role, "redpacket", page, pageSize)
}

func (s *BalanceTransferService) GetAllTransfers(ctx context.Context, filter *TransferFilter, page, pageSize int) ([]*BalanceTransferRecord, int, error) {
	records, total, err := s.transferRepo.ListAll(ctx, filter, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	userIDs := make([]int64, 0, len(records)*2)
	seenUserIDs := make(map[int64]struct{})
	for _, r := range records {
		if _, ok := seenUserIDs[r.SenderID]; !ok {
			seenUserIDs[r.SenderID] = struct{}{}
			userIDs = append(userIDs, r.SenderID)
		}
		if _, ok := seenUserIDs[r.ReceiverID]; !ok {
			seenUserIDs[r.ReceiverID] = struct{}{}
			userIDs = append(userIDs, r.ReceiverID)
		}
	}
	emails, err := s.getTransferUserEmails(ctx, userIDs)
	if err != nil {
		return nil, 0, err
	}
	for _, r := range records {
		r.SenderEmail = emails[r.SenderID]
		r.ReceiverEmail = emails[r.ReceiverID]
	}
	return records, total, nil
}

func (s *BalanceTransferService) getTransferUserEmails(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	emails := make(map[int64]string, len(userIDs))
	if len(userIDs) == 0 {
		return emails, nil
	}
	if repo, ok := s.userRepo.(balanceTransferUserEmailBatchRepository); ok {
		batched, err := repo.GetEmailsByIDs(ctx, userIDs)
		if err != nil {
			return nil, fmt.Errorf("batch get transfer user emails: %w", err)
		}
		return batched, nil
	}

	for _, uid := range userIDs {
		u, err := s.userRepo.GetByID(ctx, uid)
		if err == nil && u != nil {
			emails[uid] = u.Email
		}
	}
	return emails, nil
}

func (s *BalanceTransferService) FreezeTransfer(ctx context.Context, adminID, transferID int64) error {
	record, err := s.transferRepo.GetByID(ctx, transferID)
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
	return s.transferRepo.UpdateStatus(ctx, transferID, "frozen", &now, &adminID, nil)
}

func (s *BalanceTransferService) RevokeTransfer(ctx context.Context, adminID, transferID int64, reason string) error {
	return s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		record, err := s.getTransferForUpdate(txCtx, transferID)
		if err != nil {
			return ErrTransferNotFound
		}
		if record.Status == "revoked" {
			return ErrTransferAlreadyRevoked
		}
		if err := s.applyRevokeTransferBankEntriesInTx(txCtx, record, adminID); err != nil {
			return fmt.Errorf("apply revoke transfer bank entries: %w", err)
		}
		return s.transferRepo.UpdateStatus(txCtx, transferID, "revoked", record.FrozenAt, &adminID, &reason)
	})
}

func (s *BalanceTransferService) applyRevokeTransferBankEntriesInTx(ctx context.Context, record *BalanceTransferRecord, adminID int64) error {
	if record == nil || record.ID <= 0 {
		return infraerrors.InternalServer("TRANSFER_RECORD_MISSING", "transfer record is missing")
	}
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return err
	}
	amount := decimal.NewFromFloat(record.Amount).RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrTransferAmountInvalid
	}
	bank := NewBankService(nil)
	referenceID := fmt.Sprintf("%d", record.ID)
	baseMetadata := map[string]any{
		"admin_id":      adminID,
		"receiver_id":   record.ReceiverID,
		"sender_id":     record.SenderID,
		"transfer_id":   record.ID,
		"transfer_type": record.TransferType,
	}
	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           record.ReceiverID,
		Amount:           amount,
		Type:             BankTxTypeTransferOut,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("撤销转账扣回 %d", record.ID),
		IdempotencyScope: "balance_transfer",
		IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:revoke:receiver:amount", record.ID),
		ReferenceType:    "balance_transfer",
		ReferenceID:      referenceID,
		Metadata:         baseMetadata,
	}); err != nil {
		return fmt.Errorf("debit revoked receiver bank balance: %w", err)
	}
	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           record.SenderID,
		Amount:           amount,
		Type:             BankTxTypeTransferIn,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("撤销转账返还 %d", record.ID),
		IdempotencyScope: "balance_transfer",
		IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:revoke:sender:amount", record.ID),
		ReferenceType:    "balance_transfer",
		ReferenceID:      referenceID,
		Metadata:         baseMetadata,
	}); err != nil {
		return fmt.Errorf("return revoked sender bank balance: %w", err)
	}
	if record.Fee <= 0 {
		return nil
	}
	fee := decimal.NewFromFloat(record.Fee).RoundBank(18)
	if fee.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           record.SenderID,
		Amount:           fee,
		Type:             BankTxTypeRefund,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("撤销转账退手续费 %d", record.ID),
		IdempotencyScope: "balance_transfer",
		IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:revoke:sender:fee", record.ID),
		ReferenceType:    "balance_transfer",
		ReferenceID:      referenceID,
		Metadata: map[string]any{
			"admin_id":      adminID,
			"fee_rate":      record.FeeRate,
			"receiver_id":   record.ReceiverID,
			"sender_id":     record.SenderID,
			"transfer_id":   record.ID,
			"transfer_type": record.TransferType,
		},
	}); err != nil {
		return fmt.Errorf("refund revoked transfer fee: %w", err)
	}
	return nil
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
			if err := s.applyBatchDistributeBankEntryInTx(txCtx, adminID, record); err != nil {
				return fmt.Errorf("apply batch distribute bank entry for user %d: %w", t.UserID, err)
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

// applyBatchDistributeBankEntryInTx 将后台批量发放写入银行账本，必须复用外层事务。
func (s *BalanceTransferService) applyBatchDistributeBankEntryInTx(ctx context.Context, adminID int64, record *BalanceTransferRecord) error {
	if record == nil || record.ID <= 0 {
		return infraerrors.InternalServer("TRANSFER_RECORD_MISSING", "transfer record is missing")
	}
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return err
	}
	amount := decimal.NewFromFloat(record.Amount).RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrTransferAmountInvalid
	}

	_, err = NewBankService(nil).ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           record.ReceiverID,
		Amount:           amount,
		Type:             BankTxTypeReward,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("批量余额发放 %d", record.ID),
		IdempotencyScope: "balance_transfer",
		IdempotencyKey:   fmt.Sprintf("balance-transfer:%d:batch:reward", record.ID),
		ReferenceType:    "balance_transfer",
		ReferenceID:      fmt.Sprintf("%d", record.ID),
		Metadata: map[string]any{
			"admin_id":      adminID,
			"receiver_id":   record.ReceiverID,
			"transfer_id":   record.ID,
			"transfer_type": record.TransferType,
		},
	})
	if err != nil {
		return fmt.Errorf("credit batch distributed bank balance: %w", err)
	}
	return nil
}

func (s *BalanceTransferService) GetFeeStats(ctx context.Context, startTime, endTime time.Time) ([]*DailyFeeStat, error) {
	return s.transferRepo.GetFeeStats(ctx, startTime, endTime)
}

func (s *BalanceTransferService) GetLeaderboard(ctx context.Context, period string, limit int) ([]*TransferRankEntry, error) {
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
	if !cfg.Enabled || !cfg.RedPacketEnabled {
		return nil, ErrRedPacketDisabled
	}
	if count <= 0 || count > cfg.RedPacketMaxCount {
		return nil, ErrRedPacketCountInvalid
	}
	totalAmount = math.Round(totalAmount*1e8) / 1e8
	minRequired := float64(count) * 0.01
	if totalAmount < minRequired {
		return nil, infraerrors.BadRequest("REDPACKET_AMOUNT_TOO_SMALL", fmt.Sprintf("minimum amount for %d packets is %.2f", count, minRequired))
	}
	feeRate := cfg.FeeRate
	fee := math.Round(totalAmount*feeRate*1e8) / 1e8
	grossAmount := totalAmount + fee
	code, err := generateRedPacketCode()
	if err != nil {
		return nil, fmt.Errorf("generate code: %w", err)
	}
	expireHours := cfg.RedPacketExpireHours
	if expireHours <= 0 {
		expireHours = 24
	}
	var rp *RedPacketRecord
	if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		senderBalance, err := s.lockBankAvailableCapacity(txCtx, senderID)
		if err != nil {
			return fmt.Errorf("lock sender balance: %w", err)
		}
		if senderBalance < grossAmount {
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
		if err := s.redPacketRepo.Create(txCtx, rp); err != nil {
			return err
		}
		if err := s.applyCreateRedPacketBankEntriesInTx(txCtx, senderID, rp); err != nil {
			return fmt.Errorf("apply red packet create bank entries: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return rp, nil
}

func (s *BalanceTransferService) ClaimRedPacket(ctx context.Context, userID int64, code string) (*RedPacketClaimRecord, error) {
	cfg := s.getTransferSettings(ctx)
	if !cfg.Enabled || !cfg.RedPacketEnabled {
		return nil, ErrRedPacketDisabled
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
	claimed, err := s.redPacketRepo.HasClaimed(ctx, rp.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("check claimed: %w", err)
	}
	if claimed {
		return nil, ErrRedPacketAlreadyClaimed
	}

	lockKey := fmt.Sprintf("rp:%d", rp.ID)
	actual, _ := s.claimLocks.LoadOrStore(lockKey, &sync.Mutex{})
	mu, ok := actual.(*sync.Mutex)
	if !ok {
		return nil, fmt.Errorf("claim lock type mismatch for %s", lockKey)
	}
	mu.Lock()
	defer mu.Unlock()

	freshRp, err := s.redPacketRepo.GetByID(ctx, rp.ID)
	if err != nil {
		return nil, ErrRedPacketNotFound
	}
	if freshRp.Status != "active" || freshRp.RemainingCount <= 0 || freshRp.RemainingAmount <= 0 {
		return nil, ErrRedPacketExhausted
	}

	amount := s.calculateClaimAmount(freshRp)
	if amount <= 0 {
		return nil, ErrRedPacketExhausted
	}
	remainingCount := freshRp.RemainingCount
	var claimRecord *RedPacketClaimRecord
	if err := s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.redPacketRepo.DecrementClaim(txCtx, freshRp.ID, amount); err != nil {
			return ErrRedPacketExhausted
		}
		claimRecord = &RedPacketClaimRecord{
			RedPacketID: freshRp.ID,
			UserID:      userID,
			Amount:      amount,
			CreatedAt:   time.Now(),
		}
		transferRecord := &BalanceTransferRecord{
			SenderID:     freshRp.SenderID,
			ReceiverID:   userID,
			Amount:       amount,
			Fee:          0,
			FeeRate:      0,
			GrossAmount:  amount,
			TransferType: "redpacket",
			Status:       "completed",
			RedpacketID:  &freshRp.ID,
			CreatedAt:    time.Now(),
		}
		if err := s.transferRepo.Create(txCtx, transferRecord); err != nil {
			return fmt.Errorf("create transfer record: %w", err)
		}
		if err := s.applyClaimRedPacketBankEntryInTx(txCtx, userID, freshRp, transferRecord); err != nil {
			return fmt.Errorf("apply red packet claim bank entry: %w", err)
		}
		claimRecord.TransferID = &transferRecord.ID
		if err := s.redPacketRepo.CreateClaim(txCtx, claimRecord); err != nil {
			return fmt.Errorf("create claim record: %w", err)
		}
		if remainingCount <= 1 {
			return s.redPacketRepo.MarkExhausted(txCtx, freshRp.ID)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return claimRecord, nil
}

func (s *BalanceTransferService) GetRedPacketDetail(ctx context.Context, viewerID, redPacketID int64) (*RedPacketRecord, []*RedPacketClaimRecord, error) {
	rp, err := s.redPacketRepo.GetByID(ctx, redPacketID)
	if err != nil {
		return nil, nil, ErrRedPacketNotFound
	}
	if rp.SenderID != viewerID {
		return nil, nil, ErrRedPacketNotFound
	}
	claims, err := s.redPacketRepo.GetClaims(ctx, redPacketID)
	if err != nil {
		claims = []*RedPacketClaimRecord{}
	}
	for _, c := range claims {
		if u, err := s.userRepo.GetByID(ctx, c.UserID); err == nil {
			c.UserEmail = u.Email
		}
	}
	return rp, claims, nil
}

func (s *BalanceTransferService) GetMyRedPackets(ctx context.Context, senderID int64, page, pageSize int) ([]*RedPacketRecord, int, error) {
	return s.redPacketRepo.ListBySender(ctx, senderID, page, pageSize)
}

func (s *BalanceTransferService) ExpireRedPacket(ctx context.Context, redPacketID int64) error {
	rp, err := s.redPacketRepo.GetByID(ctx, redPacketID)
	if err != nil {
		return ErrRedPacketNotFound
	}
	return s.expireRedPacket(ctx, rp)
}

func (s *BalanceTransferService) ExpireRedPackets(ctx context.Context) error {
	rps, err := s.redPacketRepo.ListActiveExpired(ctx)
	if err != nil {
		return err
	}
	var joinedErr error
	for _, rp := range rps {
		if err := s.expireRedPacket(ctx, rp); err != nil {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("expire red packet %d: %w", rp.ID, err))
		}
	}
	return joinedErr
}

func (s *BalanceTransferService) GetAllRedPackets(ctx context.Context, page, pageSize int) ([]*RedPacketRecord, int, error) {
	return s.redPacketRepo.ListAll(ctx, page, pageSize)
}

func (s *BalanceTransferService) expireRedPacket(ctx context.Context, rp *RedPacketRecord) error {
	if rp == nil {
		return ErrRedPacketNotFound
	}
	return s.transferRepo.RunInTx(ctx, func(txCtx context.Context) error {
		remaining, err := s.redPacketRepo.ReturnRemaining(txCtx, rp.ID, rp.SenderID)
		if err != nil {
			return err
		}
		if remaining > 0 {
			return s.applyExpireRedPacketRefundBankEntryInTx(txCtx, rp, remaining)
		}
		return nil
	})
}

func (s *BalanceTransferService) applyCreateRedPacketBankEntriesInTx(ctx context.Context, senderID int64, rp *RedPacketRecord) error {
	if rp == nil || rp.ID <= 0 {
		return infraerrors.InternalServer("REDPACKET_RECORD_MISSING", "red packet record is missing")
	}
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return err
	}
	bank := NewBankService(nil)
	referenceID := fmt.Sprintf("%d", rp.ID)
	amount := decimal.NewFromFloat(rp.TotalAmount).RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrTransferAmountInvalid
	}
	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           senderID,
		Amount:           amount,
		Type:             BankTxTypeTransferOut,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("创建红包 %d", rp.ID),
		IdempotencyScope: "balance_redpacket",
		IdempotencyKey:   fmt.Sprintf("balance-redpacket:%d:create:amount", rp.ID),
		ReferenceType:    "balance_redpacket",
		ReferenceID:      referenceID,
		Metadata: map[string]any{
			"redpacket_id":   rp.ID,
			"redpacket_type": rp.RedPacketType,
			"sender_id":      senderID,
		},
	}); err != nil {
		return fmt.Errorf("debit red packet amount: %w", err)
	}
	if rp.Fee <= 0 {
		return nil
	}
	fee := decimal.NewFromFloat(rp.Fee).RoundBank(18)
	if fee.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	if _, err := bank.ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           senderID,
		Amount:           fee,
		Type:             BankTxTypeWithdraw,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("红包手续费 %d", rp.ID),
		IdempotencyScope: "balance_redpacket",
		IdempotencyKey:   fmt.Sprintf("balance-redpacket:%d:create:fee", rp.ID),
		ReferenceType:    "balance_redpacket",
		ReferenceID:      referenceID,
		Metadata: map[string]any{
			"fee_rate":       rp.FeeRate,
			"redpacket_id":   rp.ID,
			"redpacket_type": rp.RedPacketType,
			"sender_id":      senderID,
		},
	}); err != nil {
		return fmt.Errorf("debit red packet fee: %w", err)
	}
	return nil
}

func (s *BalanceTransferService) applyClaimRedPacketBankEntryInTx(ctx context.Context, userID int64, rp *RedPacketRecord, record *BalanceTransferRecord) error {
	if rp == nil || rp.ID <= 0 || record == nil || record.ID <= 0 {
		return infraerrors.InternalServer("REDPACKET_CLAIM_RECORD_MISSING", "red packet claim record is missing")
	}
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return err
	}
	amount := decimal.NewFromFloat(record.Amount).RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return ErrTransferAmountInvalid
	}
	_, err = NewBankService(nil).ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           userID,
		Amount:           amount,
		Type:             BankTxTypeTransferIn,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("领取红包 %d", rp.ID),
		IdempotencyScope: "balance_redpacket",
		IdempotencyKey:   fmt.Sprintf("balance-redpacket:%d:claim:%d", rp.ID, record.ID),
		ReferenceType:    "balance_redpacket",
		ReferenceID:      fmt.Sprintf("%d", rp.ID),
		Metadata: map[string]any{
			"claimant_id":   userID,
			"redpacket_id":  rp.ID,
			"sender_id":     rp.SenderID,
			"transfer_id":   record.ID,
			"transfer_type": record.TransferType,
		},
	})
	if err != nil {
		return fmt.Errorf("credit red packet claim: %w", err)
	}
	return nil
}

func (s *BalanceTransferService) applyExpireRedPacketRefundBankEntryInTx(ctx context.Context, rp *RedPacketRecord, remaining float64) error {
	if rp == nil || rp.ID <= 0 {
		return infraerrors.InternalServer("REDPACKET_RECORD_MISSING", "red packet record is missing")
	}
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return err
	}
	amount := decimal.NewFromFloat(remaining).RoundBank(18)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil
	}
	_, err = NewBankService(nil).ApplyTransferInTx(ctx, client, TransferFundsRequest{
		UserID:           rp.SenderID,
		Amount:           amount,
		Type:             BankTxTypeRefund,
		BusinessModule:   BankBusinessModuleTransfer,
		Description:      fmt.Sprintf("红包过期退回 %d", rp.ID),
		IdempotencyScope: "balance_redpacket",
		IdempotencyKey:   fmt.Sprintf("balance-redpacket:%d:expire:refund", rp.ID),
		ReferenceType:    "balance_redpacket",
		ReferenceID:      fmt.Sprintf("%d", rp.ID),
		Metadata: map[string]any{
			"redpacket_id":     rp.ID,
			"redpacket_type":   rp.RedPacketType,
			"remaining_amount": amount.String(),
			"sender_id":        rp.SenderID,
		},
	})
	if err != nil {
		return fmt.Errorf("refund expired red packet remaining: %w", err)
	}
	return nil
}

func (s *BalanceTransferService) calculateClaimAmount(rp *RedPacketRecord) float64 {
	if rp.RemainingCount <= 0 || rp.RemainingAmount <= 0 {
		return 0
	}
	if rp.RedPacketType == "equal" {
		return math.Round(rp.RemainingAmount/float64(rp.RemainingCount)*1e8) / 1e8
	}
	if rp.RemainingCount == 1 {
		return math.Round(rp.RemainingAmount*1e8) / 1e8
	}
	maxAllowed := rp.RemainingAmount - float64(rp.RemainingCount-1)*0.01
	upperBound := maxAllowed / float64(rp.RemainingCount) * 2
	if upperBound <= 0.01 {
		return 0.01
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(upperBound*100)))
	if err != nil {
		return math.Round(maxAllowed/float64(rp.RemainingCount)*1e8) / 1e8
	}
	amount := float64(n.Int64())/100 + 0.01
	if amount < 0.01 {
		amount = 0.01
	}
	if amount > maxAllowed {
		amount = maxAllowed
	}
	return math.Round(amount*1e8) / 1e8
}

func generateRedPacketCode() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *BalanceTransferService) GetTransferStats(ctx context.Context, userID int64) (sent float64, received float64, feePaid float64, err error) {
	return s.transferRepo.GetUserTransferStats(ctx, userID)
}

type UserSearchResult struct {
	ID       int64  `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

func (s *BalanceTransferService) SearchUsers(ctx context.Context, query string) ([]*UserSearchResult, error) {
	query = normalizeUserSearchQuery(query)
	if utf8.RuneCountInString(query) < userSearchMinQueryRunes {
		return []*UserSearchResult{}, nil
	}

	var (
		users []User
		err   error
	)
	if repo, ok := s.userRepo.(balanceTransferUserSearchRepository); ok {
		users, err = repo.SearchForBalanceTransfer(ctx, query, userSearchLimit)
	} else {
		users, _, err = s.userRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: userSearchLimit}, UserListFilters{Search: query})
	}
	if err != nil {
		return nil, err
	}

	results := make([]*UserSearchResult, 0, len(users))
	for _, u := range users {
		if len(results) >= userSearchLimit {
			break
		}
		if !matchesBalanceUserSearch(u, query) {
			continue
		}
		results = append(results, &UserSearchResult{
			ID:       u.ID,
			Email:    maskEmailForUserSearch(u.Email),
			Username: maskUserSearchName(u.Username),
		})
	}
	return results, nil
}

func normalizeUserSearchQuery(query string) string {
	query = strings.Join(strings.Fields(query), " ")
	if utf8.RuneCountInString(query) <= userSearchMaxQueryRunes {
		return query
	}
	runes := []rune(query)
	return string(runes[:userSearchMaxQueryRunes])
}

func matchesBalanceUserSearch(user User, query string) bool {
	query = strings.ToLower(query)
	return strings.Contains(strings.ToLower(user.Email), query) ||
		strings.Contains(strings.ToLower(user.Username), query)
}

func maskEmailForUserSearch(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 {
		return maskUserSearchName(email)
	}
	local := []rune(email[:at])
	domain := []rune(email[at+1:])
	if len(local) == 0 || len(domain) == 0 {
		return maskUserSearchName(email)
	}
	return maskUserSearchSegment(local) + "@" + maskUserSearchDomain(domain)
}

func maskUserSearchName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return maskUserSearchSegment([]rune(value))
}

func maskUserSearchSegment(runes []rune) string {
	switch len(runes) {
	case 0:
		return ""
	case 1:
		return string(runes[0])
	case 2:
		return string(runes[0]) + "*"
	default:
		return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
	}
}

func maskUserSearchDomain(runes []rune) string {
	domain := string(runes)
	dot := strings.LastIndex(domain, ".")
	if dot <= 0 {
		return maskUserSearchSegment(runes)
	}
	name := []rune(domain[:dot])
	suffix := domain[dot:]
	return maskUserSearchSegment(name) + suffix
}

func (s *BalanceTransferService) getTransferForUpdate(ctx context.Context, transferID int64) (*BalanceTransferRecord, error) {
	if repo, ok := s.transferRepo.(balanceTransferTxGuard); ok {
		return repo.GetByIDForUpdate(ctx, transferID)
	}
	return s.transferRepo.GetByID(ctx, transferID)
}

func (s *BalanceTransferService) lockBankAvailableCapacity(ctx context.Context, userID int64) (float64, error) {
	client, err := bankClientFromTxContext(ctx)
	if err != nil {
		return 0, err
	}
	account, err := lockBankAccountForUpdate(ctx, client, userID)
	if err != nil {
		return 0, err
	}
	if account.Status != BankAccountStatusActive {
		return 0, ErrBankAccountNotActive
	}
	available := (&BankAccountView{
		Balance:       account.Balance,
		CreditLimit:   account.CreditLimit,
		DebtPrincipal: account.DebtPrincipal,
		DebtInterest:  account.DebtInterest,
		TotalDebt:     account.TotalDebt,
		Status:        account.Status,
	}).AvailableCapacity()
	if available.IsNegative() {
		return 0, nil
	}
	return available.InexactFloat64(), nil
}

func bankClientFromTxContext(ctx context.Context) (*dbent.Client, error) {
	tx := dbent.TxFromContext(ctx)
	if tx == nil {
		return nil, infraerrors.InternalServer("BANK_TRANSACTION_REQUIRED", "bank operation requires an active transaction")
	}
	return tx.Client(), nil
}
