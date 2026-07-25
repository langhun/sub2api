package redpacket

import (
	"context"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	coreservice "github.com/Wei-Shaw/sub2api/internal/service"
)

// NewTransactionRunner adapts Ent transactions to the Activity contract.
func NewTransactionRunner(client *dbent.Client) contract.TransactionRunner {
	return entTransactionRunner{client: client}
}

type entTransactionRunner struct{ client *dbent.Client }

func (r entTransactionRunner) RunInTransaction(ctx context.Context, operation func(context.Context) error) error {
	if r.client == nil {
		return fmt.Errorf("ent client is required")
	}
	if dbent.TxFromContext(ctx) != nil {
		return operation(ctx)
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin red-packet transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := operation(dbent.NewTxContext(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit()
}

// NewBalanceWriter adapts the existing users balance column without treating
// Activity rewards as customer recharge volume.
func NewBalanceWriter(client *dbent.Client) contract.BalanceWriter {
	return entBalanceWriter{client: client}
}

type entBalanceWriter struct{ client *dbent.Client }

func (w entBalanceWriter) Credit(ctx context.Context, operation contract.BalanceOperation) error {
	if w.client == nil || operation.UserID <= 0 || !validAmount(operation.Amount) {
		return fmt.Errorf("invalid red-packet balance credit")
	}
	updated, err := entClient(ctx, w.client).User.Update().Where(user.IDEQ(operation.UserID)).AddBalance(operation.Amount).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return fmt.Errorf("red-packet account %d not found", operation.UserID)
	}
	return nil
}

func (w entBalanceWriter) DebitIfSufficient(ctx context.Context, operation contract.BalanceOperation) (bool, error) {
	if w.client == nil || operation.UserID <= 0 || !validAmount(operation.Amount) {
		return false, fmt.Errorf("invalid red-packet balance debit")
	}
	result, err := entClient(ctx, w.client).ExecContext(ctx, `
		UPDATE users
		SET balance = balance - $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, operation.Amount, operation.UserID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

// NewClaimLedger records the exact compatibility history row that legacy
// handlers exposed as a balance transfer with transfer_type=redpacket.
func NewClaimLedger(client *dbent.Client) ClaimLedger {
	return entClaimLedger{client: client}
}

type entClaimLedger struct{ client *dbent.Client }

func (w entClaimLedger) RecordRedPacketClaim(ctx context.Context, redPacketID, senderID, receiverID int64, amount float64, occurredAt time.Time) (int64, error) {
	if w.client == nil || redPacketID <= 0 || senderID <= 0 || receiverID <= 0 || !validAmount(amount) {
		return 0, fmt.Errorf("invalid red-packet claim ledger entry")
	}
	saved, err := entClient(ctx, w.client).BalanceTransfer.Create().
		SetSenderID(senderID).SetReceiverID(receiverID).SetAmount(amount).
		SetFee(0).SetFeeRate(0).SetGrossAmount(amount).
		SetTransferType("redpacket").SetStatus("completed").SetRedpacketID(redPacketID).
		SetCreatedAt(occurredAt).Save(ctx)
	if err != nil {
		return 0, err
	}
	return saved.ID, nil
}

// SettingsAdapter reads only red-packet flags from the existing core settings
// service. It never delegates a business operation to BalanceTransferService.
type SettingsAdapter struct{ settings *coreservice.SettingService }

func NewSettingsAdapter(settings *coreservice.SettingService) *SettingsAdapter {
	return &SettingsAdapter{settings: settings}
}

func (a *SettingsAdapter) GetActivityRedPacketSettings(ctx context.Context) (contract.RedPacketSettings, error) {
	if a == nil || a.settings == nil {
		return contract.RedPacketSettings{}, fmt.Errorf("settings service is required")
	}
	settings, err := a.settings.GetAllSettings(ctx)
	if err != nil {
		return contract.RedPacketSettings{}, err
	}
	return contract.RedPacketSettings{Enabled: settings.RedPacketEnabled, MaximumCount: settings.RedPacketMaxCount, ExpireHours: settings.RedPacketExpireHours}, nil
}

// SettingsCodeGenerator keeps the configured code shape while making the
// generator an Activity dependency rather than a transfer-service helper.
type SettingsCodeGenerator struct{ settings *coreservice.SettingService }

func NewSettingsCodeGenerator(settings *coreservice.SettingService) SettingsCodeGenerator {
	return SettingsCodeGenerator{settings: settings}
}

func (g SettingsCodeGenerator) GenerateRedPacketCode(ctx context.Context) (string, error) {
	formats := coreservice.DefaultCodeFormatSettings()
	if g.settings != nil {
		formats = g.settings.GetCodeFormatSettings(ctx)
	}
	return formats.RedPacket.Generate()
}

// FeeAdapter preserves the existing transfer-fee and VIP exemption policy as a
// narrow pricing dependency. It does not use BalanceTransferService.
type FeeAdapter struct {
	settings      *coreservice.SettingService
	subscriptions *coreservice.SubscriptionService
}

func NewFeeAdapter(settings *coreservice.SettingService, subscriptions *coreservice.SubscriptionService) *FeeAdapter {
	return &FeeAdapter{settings: settings, subscriptions: subscriptions}
}

func (a *FeeAdapter) QuoteRedPacketFee(ctx context.Context, senderID int64, totalAmount float64) (FeeQuote, error) {
	if a == nil || a.settings == nil || senderID <= 0 || !validAmount(totalAmount) {
		return FeeQuote{}, fmt.Errorf("invalid red-packet fee request")
	}
	settings, err := a.settings.GetAllSettings(ctx)
	if err != nil {
		return FeeQuote{}, err
	}
	rate := settings.TransferFeeRate
	if settings.TransferVIPFeeExempt && a.subscriptions != nil {
		subscriptions, listErr := a.subscriptions.ListActiveUserSubscriptions(ctx, senderID)
		if listErr == nil && len(subscriptions) > 0 {
			rate = 0
		}
	}
	if math.IsNaN(rate) || math.IsInf(rate, 0) || rate < 0 {
		return FeeQuote{}, fmt.Errorf("invalid red-packet fee rate")
	}
	return FeeQuote{Rate: rate, Amount: roundAmount(totalAmount * rate)}, nil
}

func entClient(ctx context.Context, client *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return client
}
