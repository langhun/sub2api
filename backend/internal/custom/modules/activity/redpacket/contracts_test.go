package redpacket

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

func TestLegacyExtractionBoundaryIsCompleteAndFocused(t *testing.T) {
	want := map[string]string{
		"BalanceTransferService.CreateRedPacket":           "redpacket.Creator.Create",
		"BalanceTransferService.ClaimRedPacket":            "redpacket.Claimer.Claim",
		"BalanceTransferService.ExpireRedPackets":          "redpacket.ExpiryRefunder.RefundExpired",
		"BalanceTransferService.GetRedPacketDetail":        "redpacket.QueryService.Get",
		"BalanceTransferService.GetRedPacketDetailForUser": "redpacket.QueryService.GetForParticipant",
		"BalanceTransferService.GetMyRedPackets":           "redpacket.QueryService.ListCreatedBy and ListClaimedBy",
		"BalanceTransferService.GetAllRedPackets":          "redpacket.QueryService.ListAll",
	}
	if len(LegacyExtractionBoundary) != len(want) {
		t.Fatalf("legacy boundary has %d methods, want %d", len(LegacyExtractionBoundary), len(want))
	}
	for _, method := range LegacyExtractionBoundary {
		if replacement, ok := want[method.Source]; !ok || replacement != method.Replacement {
			t.Fatalf("unexpected boundary %#v", method)
		}
		delete(want, method.Source)
	}
	if len(want) != 0 {
		t.Fatalf("legacy boundary is missing %#v", want)
	}
}

func TestDependenciesExposeTheRequiredCorePorts(t *testing.T) {
	var dependencies Dependencies
	if dependencies.Repository != nil || dependencies.Transactions != nil || dependencies.Balance != nil || dependencies.Audit != nil || dependencies.Code != nil || dependencies.Clock != nil {
		t.Fatal("zero-value dependencies must not install implicit core adapters")
	}

	var _ contract.TransactionRunner = transactionRunnerStub{}
	var _ contract.BalanceWriter = balanceWriterStub{}
	var _ contract.AuditWriter = auditWriterStub{}
	var _ contract.SingletonLeaseCoordinator = leaseCoordinatorStub{}
}

func TestServicePortsRemainNarrow(t *testing.T) {
	var _ Creator = serviceStub{}
	var _ Claimer = serviceStub{}
	var _ ExpiryRefunder = serviceStub{}
	var _ QueryService = serviceStub{}
	var _ Service = serviceStub{}
	var _ ExpiryWorker = workerStub{}
}

type serviceStub struct{}

func (serviceStub) Create(context.Context, CreateRequest) (*RedPacket, error) { return nil, nil }
func (serviceStub) Claim(context.Context, ClaimRequest) (*Claim, error)       { return nil, nil }
func (serviceStub) RefundExpired(context.Context) (ExpiryRunResult, error) {
	return ExpiryRunResult{}, nil
}
func (serviceStub) Get(context.Context, int64) (*RedPacket, error) { return nil, nil }
func (serviceStub) GetForParticipant(context.Context, int64, int64) (*RedPacket, []Claim, error) {
	return nil, nil, nil
}
func (serviceStub) ListCreatedBy(context.Context, int64, int, int) ([]RedPacket, int, error) {
	return nil, 0, nil
}
func (serviceStub) ListClaimedBy(context.Context, int64, int, int) ([]RedPacket, int, error) {
	return nil, 0, nil
}
func (serviceStub) ListAll(context.Context, int, int) ([]RedPacket, int, error) {
	return nil, 0, nil
}

type transactionRunnerStub struct{}

func (transactionRunnerStub) RunInTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type balanceWriterStub struct{}

func (balanceWriterStub) Credit(context.Context, contract.BalanceOperation) error { return nil }
func (balanceWriterStub) DebitIfSufficient(context.Context, contract.BalanceOperation) (bool, error) {
	return true, nil
}

type auditWriterStub struct{}

func (auditWriterStub) WriteActivityAudit(context.Context, contract.AuditEntry) error { return nil }

type leaseCoordinatorStub struct{}

func (leaseCoordinatorStub) AcquireSingletonLease(context.Context, string, string, time.Duration) (contract.Lease, bool, error) {
	return nil, false, nil
}

type workerStub struct{}

func (workerStub) Start() {}
func (workerStub) Stop()  {}
