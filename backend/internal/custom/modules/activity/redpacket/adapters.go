package redpacket

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
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

// RegistrySettingsAdapter reads red-packet policy from the Overlay registry.
// It never delegates a business operation to a legacy settings or transfer service.
type RegistrySettingsAdapter struct{ registry *customsettings.Registry }

func NewRegistrySettingsAdapter(registry *customsettings.Registry) *RegistrySettingsAdapter {
	return &RegistrySettingsAdapter{registry: registry}
}

func (a *RegistrySettingsAdapter) GetActivityRedPacketSettings(ctx context.Context) (contract.RedPacketSettings, error) {
	if a == nil || a.registry == nil {
		return contract.RedPacketSettings{}, fmt.Errorf("custom settings registry is required")
	}
	snapshot, err := a.registry.Read(ctx)
	if err != nil {
		return contract.RedPacketSettings{}, fmt.Errorf("read custom activity settings: %w", err)
	}
	settings := snapshot.Activity
	return contract.RedPacketSettings{Enabled: settings.RedPacketEnabled, MaximumCount: settings.RedPacketMaxCount, ExpireHours: settings.RedPacketExpireHours}, nil
}

// CodeFormatSettingsSource is the narrow platform capability Activity needs
// to create a red-packet code. It intentionally exposes no settings object.
type CodeFormatSettingsSource interface {
	GenerateRedPacketCode(context.Context) (string, error)
}

type SettingsCodeGenerator struct{ settings CodeFormatSettingsSource }

func NewSettingsCodeGenerator(settings CodeFormatSettingsSource) SettingsCodeGenerator {
	return SettingsCodeGenerator{settings: settings}
}

func (g SettingsCodeGenerator) GenerateRedPacketCode(ctx context.Context) (string, error) {
	if g.settings == nil {
		return "", fmt.Errorf("red-packet code generator is unavailable")
	}
	return g.settings.GenerateRedPacketCode(ctx)
}

// ZeroFeeAdapter keeps red-packet pricing independent of the removed balance
// transfer module. Existing packets and ledgers retain their historical fees.
type ZeroFeeAdapter struct{}

func NewZeroFeeAdapter() ZeroFeeAdapter { return ZeroFeeAdapter{} }

func (ZeroFeeAdapter) QuoteRedPacketFee(_ context.Context, senderID int64, totalAmount float64) (FeeQuote, error) {
	if senderID <= 0 || !validAmount(totalAmount) {
		return FeeQuote{}, fmt.Errorf("invalid red-packet fee request")
	}
	return FeeQuote{}, nil
}

func entClient(ctx context.Context, client *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return client
}
