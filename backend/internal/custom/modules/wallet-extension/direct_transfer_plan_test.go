package walletextension

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/wallet-extension/contract"
)

func TestDirectTransferMigrationPlanIsBoundedAndOrdered(t *testing.T) {
	plan := DirectTransferMigrationPlan
	if plan.Name != "wallet-extension-transfer-ledger" {
		t.Fatalf("plan name = %q, want wallet-extension-transfer-ledger", plan.Name)
	}
	if len(plan.Steps) != 3 {
		t.Fatalf("plan has %d steps, want 3", len(plan.Steps))
	}
	for index, want := range []MigrationLayer{MigrationLayerRepository, MigrationLayerService, MigrationLayerHandler} {
		if plan.Steps[index].Layer != want {
			t.Fatalf("step %d layer = %q, want %q", index, plan.Steps[index].Layer, want)
		}
		if plan.Steps[index].LegacySource == "" || plan.Steps[index].Target == "" {
			t.Fatalf("step %d must name legacy source and target", index)
		}
	}

	excluded := make(map[string]struct{}, len(plan.ExcludedCapabilities))
	for _, capability := range plan.ExcludedCapabilities {
		excluded[capability] = struct{}{}
	}
	for _, capability := range []string{"red packet", "blind box"} {
		if _, exists := excluded[capability]; !exists {
			t.Fatalf("plan must explicitly exclude %q", capability)
		}
	}
}

func TestDirectTransferContractsCompile(t *testing.T) {
	var _ DirectTransferService = directTransferServiceStub{}
	var _ DirectTransferHandler = directTransferHandlerStub{}
	var _ DirectTransferRepository = directTransferRepositoryStub{}
	var _ contract.AccountReader = accountReaderStub{}
	var _ contract.RecipientResolver = recipientResolverStub{}
	var _ contract.BalanceWriter = balanceWriterStub{}
	var _ contract.AuditWriter = auditWriterStub{}
	var _ contract.SettingsReader = settingsReaderStub{}
	var _ contract.TransactionRunner = transactionRunnerStub{}
}

type directTransferServiceStub struct{}

func (directTransferServiceStub) Transfer(context.Context, int64, DirectTransferRequest) (DirectTransferRecord, error) {
	return DirectTransferRecord{}, nil
}
func (directTransferServiceStub) Preview(context.Context, int64, int64, float64) (DirectTransferPreview, error) {
	return DirectTransferPreview{}, nil
}
func (directTransferServiceStub) ResolveRecipient(context.Context, int64, string) (contract.Recipient, error) {
	return contract.Recipient{}, nil
}
func (directTransferServiceStub) SearchRecipients(context.Context, int64, string) ([]contract.RecipientCandidate, error) {
	return nil, nil
}
func (directTransferServiceStub) ListHistory(context.Context, DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error) {
	return nil, 0, nil
}
func (directTransferServiceStub) GetStats(context.Context, int64) (DirectTransferStats, error) {
	return DirectTransferStats{}, nil
}

type directTransferHandlerStub struct{}

func (directTransferHandlerStub) HandleTransfer(context.Context, int64, DirectTransferRequest) (DirectTransferRecord, error) {
	return DirectTransferRecord{}, nil
}
func (directTransferHandlerStub) HandlePreview(context.Context, int64, int64, float64) (DirectTransferPreview, error) {
	return DirectTransferPreview{}, nil
}
func (directTransferHandlerStub) HandleResolveRecipient(context.Context, int64, string) (contract.Recipient, error) {
	return contract.Recipient{}, nil
}
func (directTransferHandlerStub) HandleSearchRecipients(context.Context, int64, string) ([]contract.RecipientCandidate, error) {
	return nil, nil
}
func (directTransferHandlerStub) HandleHistory(context.Context, DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error) {
	return nil, 0, nil
}
func (directTransferHandlerStub) HandleStats(context.Context, int64) (DirectTransferStats, error) {
	return DirectTransferStats{}, nil
}

type directTransferRepositoryStub struct{}

func (directTransferRepositoryStub) CommitDirectTransfer(context.Context, DirectTransferCommitPlan) (DirectTransferRecord, error) {
	return DirectTransferRecord{}, nil
}
func (directTransferRepositoryStub) CreateDirectTransfer(context.Context, *DirectTransferRecord) error {
	return nil
}
func (directTransferRepositoryStub) GetDirectTransfer(context.Context, int64) (DirectTransferRecord, error) {
	return DirectTransferRecord{}, nil
}
func (directTransferRepositoryStub) ListDirectTransferHistory(context.Context, DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error) {
	return nil, 0, nil
}
func (directTransferRepositoryStub) GetDirectTransferDailyUsage(context.Context, int64, time.Time, time.Time) (float64, int, error) {
	return 0, 0, nil
}
func (directTransferRepositoryStub) GetDirectTransferStats(context.Context, int64) (DirectTransferStats, error) {
	return DirectTransferStats{}, nil
}

type accountReaderStub struct{}

func (accountReaderStub) GetAccount(context.Context, int64) (contract.Account, error) {
	return contract.Account{}, nil
}

type recipientResolverStub struct{}

func (recipientResolverStub) ResolveDirectTransferRecipient(context.Context, int64, string) (contract.Recipient, error) {
	return contract.Recipient{}, nil
}
func (recipientResolverStub) SearchDirectTransferRecipients(context.Context, int64, string, int) ([]contract.RecipientCandidate, error) {
	return nil, nil
}

type balanceWriterStub struct{}

func (balanceWriterStub) Credit(context.Context, contract.BalanceOperation) error { return nil }
func (balanceWriterStub) DebitIfSufficient(context.Context, contract.BalanceOperation) (bool, error) {
	return true, nil
}

type auditWriterStub struct{}

func (auditWriterStub) WriteBalanceAudit(context.Context, contract.BalanceAuditEntry) error {
	return nil
}

type settingsReaderStub struct{}

func (settingsReaderStub) GetWalletExtensionSettings(context.Context) (contract.Settings, error) {
	return contract.Settings{}, nil
}

type transactionRunnerStub struct{}

func (transactionRunnerStub) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
