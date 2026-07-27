package walletextension

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
)

const batchTransferType = "batch"

// TransferRecord is the administrative projection of the shared transfer
// ledger. It retains the established response fields for every transfer type.
type TransferRecord struct {
	ID              int64      `json:"id"`
	SenderID        int64      `json:"sender_id"`
	ReceiverID      int64      `json:"receiver_id"`
	SenderDisplay   string     `json:"sender_display"`
	ReceiverDisplay string     `json:"receiver_display"`
	Amount          float64    `json:"amount"`
	Fee             float64    `json:"fee"`
	FeeRate         float64    `json:"fee_rate"`
	GrossAmount     float64    `json:"gross_amount"`
	TransferType    string     `json:"transfer_type"`
	Status          string     `json:"status"`
	Memo            *string    `json:"memo"`
	RedpacketID     *int64     `json:"redpacket_id"`
	FrozenAt        *time.Time `json:"frozen_at"`
	FrozenBy        *int64     `json:"frozen_by"`
	RevokeReason    *string    `json:"revoke_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

// TransferFilter scopes the established administrator transfer query.
type TransferFilter struct {
	Status       string
	TransferType string
	UserID       *int64
	StartTime    time.Time
	EndTime      time.Time
}

// BatchDistributeTarget is one recipient of an administrator balance grant.
type BatchDistributeTarget struct {
	UserID int64   `json:"user_id"`
	Amount float64 `json:"amount"`
}

// DailyFeeStat preserves the established transfer-fee report response.
type DailyFeeStat struct {
	Date     time.Time `json:"date"`
	TotalFee float64   `json:"total_fee"`
	Count    int       `json:"count"`
}

// TransferRankEntry preserves the public direct-transfer leaderboard response.
type TransferRankEntry struct {
	Rank        int     `json:"rank"`
	UserID      int64   `json:"user_id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	TotalAmount float64 `json:"total_amount"`
	TotalCount  int     `json:"total_count"`
}

// TransferLeaderboardSettings narrows the shared settings projection to the
// one feature gate required by this module endpoint.
type TransferLeaderboardSettings struct {
	Enabled bool
}

type transferLeaderboardSettingsReader interface {
	GetWalletTransferLeaderboardSettings(context.Context) (TransferLeaderboardSettings, error)
}

// TransferAdministrationRepository is the module-owned adapter for the
// transfer-ledger operations outside point-to-point transfer execution.
type TransferAdministrationRepository interface {
	ListAllTransfers(context.Context, TransferFilter, int, int) ([]TransferRecord, int, error)
	GetTransferForUpdate(context.Context, int64) (TransferRecord, error)
	UpdateTransferStatus(context.Context, int64, string, *time.Time, *int64, *string) error
	DebitBalanceIfSufficient(context.Context, int64, float64) (bool, error)
	CreateTransfer(context.Context, *TransferRecord) error
	GetTransferFeeStats(context.Context, time.Time, time.Time) ([]DailyFeeStat, error)
	GetTransferLeaderboard(context.Context, time.Time, time.Time, int) ([]TransferRankEntry, error)
	RunInTransaction(context.Context, func(context.Context) error) error
}

func (s *Service) transferAdministrationRepository() (TransferAdministrationRepository, error) {
	if s == nil || s.repository == nil {
		return nil, fmt.Errorf("wallet transfer administration is unavailable")
	}
	repository, ok := s.repository.(TransferAdministrationRepository)
	if !ok {
		return nil, fmt.Errorf("wallet transfer administration repository is unavailable")
	}
	return repository, nil
}

// ListTransfers returns the administrative ledger view across direct, batch,
// and compatibility ledger records, preserving the previous route contract.
func (s *Service) ListTransfers(ctx context.Context, filter TransferFilter, page, pageSize int) ([]TransferRecord, int, error) {
	repository, err := s.transferAdministrationRepository()
	if err != nil {
		return nil, 0, err
	}
	return repository.ListAllTransfers(ctx, filter, page, pageSize)
}

// GetFeeStats returns completed transfer fees for the requested period.
func (s *Service) GetFeeStats(ctx context.Context, startTime, endTime time.Time) ([]DailyFeeStat, error) {
	repository, err := s.transferAdministrationRepository()
	if err != nil {
		return nil, err
	}
	return repository.GetTransferFeeStats(ctx, startTime, endTime)
}

// FreezeTransfer preserves the legacy final-state validation and locking.
func (s *Service) FreezeTransfer(ctx context.Context, adminID, transferID int64) error {
	repository, err := s.transferAdministrationRepository()
	if err != nil {
		return err
	}
	return repository.RunInTransaction(ctx, func(txCtx context.Context) error {
		record, err := repository.GetTransferForUpdate(txCtx, transferID)
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
		return repository.UpdateTransferStatus(txCtx, transferID, "frozen", &now, &adminID, nil)
	})
}

// RevokeTransfer claws back a completed transfer exactly once. Batch grants
// have no sender refund because their sender is only an audit principal.
func (s *Service) RevokeTransfer(ctx context.Context, adminID, transferID int64, reason string) error {
	repository, err := s.transferAdministrationRepository()
	if err != nil {
		return err
	}
	return repository.RunInTransaction(ctx, func(txCtx context.Context) error {
		record, err := repository.GetTransferForUpdate(txCtx, transferID)
		if err != nil {
			return ErrTransferNotFound
		}
		if record.Status == "revoked" {
			return nil
		}
		ok, err := repository.DebitBalanceIfSufficient(txCtx, record.ReceiverID, record.Amount)
		if err != nil {
			return fmt.Errorf("deduct receiver balance: %w", err)
		}
		if !ok {
			return ErrTransferInsufficient
		}
		if record.TransferType != batchTransferType {
			if err := s.creditInternalBalance(txCtx, record.SenderID, record.GrossAmount); err != nil {
				return fmt.Errorf("return sender balance: %w", err)
			}
		}
		return repository.UpdateTransferStatus(txCtx, transferID, "revoked", record.FrozenAt, &adminID, &reason)
	})
}

// BatchDistribute grants balances and records each grant atomically. Invalid
// target rows retain the previous behavior and are skipped.
func (s *Service) BatchDistribute(ctx context.Context, adminID int64, targets []BatchDistributeTarget, memo *string) ([]TransferRecord, error) {
	repository, err := s.transferAdministrationRepository()
	if err != nil {
		return nil, err
	}
	records := make([]TransferRecord, 0, len(targets))
	err = repository.RunInTransaction(ctx, func(txCtx context.Context) error {
		for _, target := range targets {
			if target.UserID <= 0 || target.Amount <= 0 {
				continue
			}
			if s.accounts == nil || s.balances == nil {
				continue
			}
			if _, err := s.accounts.GetAccount(txCtx, target.UserID); err != nil {
				continue
			}
			if err := s.creditInternalBalance(txCtx, target.UserID, target.Amount); err != nil {
				return fmt.Errorf("update balance for user %d: %w", target.UserID, err)
			}
			record := TransferRecord{
				SenderID: adminID, ReceiverID: target.UserID, Amount: target.Amount, Fee: 0, FeeRate: 0,
				GrossAmount: target.Amount, TransferType: batchTransferType, Status: "completed", Memo: memo, CreatedAt: time.Now(),
			}
			if err := repository.CreateTransfer(txCtx, &record); err != nil {
				return fmt.Errorf("create transfer record for user %d: %w", target.UserID, err)
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

// GetLeaderboard returns the enabled direct-transfer leaderboard for day,
// week, or month. Other values retain the previous day-default behavior.
func (s *Service) GetLeaderboard(ctx context.Context, period string, limit int) ([]TransferRankEntry, error) {
	if _, err := s.directTransferSettings(ctx); err != nil {
		return nil, err
	}
	reader, ok := s.settings.(transferLeaderboardSettingsReader)
	if !ok {
		return nil, ErrTransferLeaderboardDisabled
	}
	settings, err := reader.GetWalletTransferLeaderboardSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, ErrTransferLeaderboardDisabled
	}
	repository, err := s.transferAdministrationRepository()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	start := now.AddDate(0, 0, -1)
	switch period {
	case "week":
		start = now.AddDate(0, 0, -7)
	case "month":
		start = now.AddDate(0, -1, 0)
	}
	return repository.GetTransferLeaderboard(ctx, start, now, limit)
}

func (s *Service) creditInternalBalance(ctx context.Context, userID int64, amount float64) error {
	if s.balances != nil {
		return s.balances.Credit(ctx, contract.BalanceOperation{AccountID: userID, Amount: amount, Reason: "wallet_transfer"})
	}
	return fmt.Errorf("wallet balance writer is unavailable")
}
