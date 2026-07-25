package redpacket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/stretchr/testify/require"
)

func TestServiceCreateUsesActivityPortsInOneTransaction(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	repository := &serviceRepositoryStub{}
	balance := &balanceStub{debitOK: true}
	audit := &auditStub{}
	service := NewService(Dependencies{
		Repository: repository, Transactions: immediateTransaction{}, Balance: balance, Audit: audit,
		Settings: settingsStub{settings: contract.RedPacketSettings{Enabled: true, MaximumCount: 10, ExpireHours: 2}},
		Code:     codeStub("RP-123"), Fees: feeStub{quote: FeeQuote{Rate: 0.1, Amount: 1}}, Ledger: ledgerStub{}, Clock: fixedClock{now: now},
	})

	packet, err := service.Create(context.Background(), CreateRequest{SenderID: 7, TotalAmount: 10, Count: 2, Type: TypeEqual, IdempotencyKey: "create-1"})

	require.NoError(t, err)
	require.Equal(t, int64(91), packet.ID)
	require.Equal(t, 11.0, balance.debits[0].Amount)
	require.Equal(t, "activity_redpacket_create", balance.debits[0].Reason)
	require.Equal(t, "RP-123", repository.created.Code)
	require.Equal(t, now.Add(2*time.Hour), repository.created.ExpiresAt)
	require.Len(t, audit.entries, 1)
	require.Equal(t, "activity_redpacket_create", audit.entries[0].Type)
}

func TestServiceClaimKeepsLockBalanceLedgerAndClaimAtomic(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	packet := &RedPacket{ID: 11, SenderID: 1, RemainingAmount: 2, RemainingCount: 1, Type: TypeEqual, Status: StatusActive, ExpiresAt: now.Add(time.Hour)}
	repository := &serviceRepositoryStub{found: packet, locked: packet, decremented: &RedPacket{ID: 11, RemainingAmount: 0, RemainingCount: 0}}
	balance := &balanceStub{debitOK: true}
	audit := &auditStub{}
	service := NewService(Dependencies{
		Repository: repository, Transactions: immediateTransaction{}, Balance: balance, Audit: audit,
		Settings: settingsStub{settings: contract.RedPacketSettings{Enabled: true, MaximumCount: 10}},
		Code:     codeStub("unused"), Fees: feeStub{}, Ledger: ledgerStub{id: 73}, Clock: fixedClock{now: now},
	})

	claim, err := service.Claim(context.Background(), ClaimRequest{UserID: 2, Code: "packet", IdempotencyKey: "claim-1"})

	require.NoError(t, err)
	require.Equal(t, 2.0, claim.Amount)
	require.Equal(t, int64(73), *claim.AuditID)
	require.Len(t, balance.credits, 1)
	require.Equal(t, 2.0, balance.credits[0].Amount)
	require.NotNil(t, repository.createdClaim)
	require.True(t, repository.exhausted)
	require.Len(t, audit.entries, 1)
	require.Equal(t, "activity_redpacket_claim", audit.entries[0].Type)
}

func TestServiceRefundExpiredReturnsOnlyLockedRemainingAmount(t *testing.T) {
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	repository := &serviceRepositoryStub{expired: []RedPacket{{ID: 9, SenderID: 3}}, returned: 1.25}
	balance := &balanceStub{debitOK: true}
	service := NewService(Dependencies{
		Repository: repository, Transactions: immediateTransaction{}, Balance: balance,
		Settings: settingsStub{settings: contract.RedPacketSettings{Enabled: true, MaximumCount: 10}},
		Code:     codeStub("unused"), Fees: feeStub{}, Ledger: ledgerStub{}, Clock: fixedClock{now: now},
	})

	result, err := service.RefundExpired(context.Background())

	require.NoError(t, err)
	require.Equal(t, 1, result.Processed)
	require.Equal(t, []ExpiryRefund{{RedPacketID: 9, SenderID: 3, ReturnedAmount: 1.25, OccurredAt: now}}, result.Refunds)
	require.Len(t, balance.credits, 1)
	require.Equal(t, "activity_redpacket_expiry_refund", balance.credits[0].Reason)
}

func TestExpiryWorkerSkipsWhenPeerOwnsLease(t *testing.T) {
	expire := &expiryStub{}
	worker := NewExpiryWorker(ExpiryWorkerDependencies{Expire: expire, Leases: leaseCoordinator{acquired: false}}).(*expiryWorker)

	worker.runOnce(context.Background())

	require.Zero(t, expire.calls)
}

type serviceRepositoryStub struct {
	created      *RedPacket
	found        *RedPacket
	locked       *RedPacket
	decremented  *RedPacket
	createdClaim *Claim
	exhausted    bool
	expired      []RedPacket
	returned     float64
}

func (r *serviceRepositoryStub) Create(_ context.Context, packet *RedPacket) error {
	packet.ID = 91
	r.created = packet
	return nil
}
func (r *serviceRepositoryStub) FindByCode(context.Context, string) (*RedPacket, error) {
	if r.found == nil {
		return nil, errors.New("not found")
	}
	return r.found, nil
}
func (r *serviceRepositoryStub) FindByCodeForUpdate(context.Context, string) (*RedPacket, error) {
	if r.locked == nil {
		return nil, errors.New("not found")
	}
	return r.locked, nil
}
func (r *serviceRepositoryStub) FindByID(context.Context, int64) (*RedPacket, error) {
	return r.found, nil
}
func (r *serviceRepositoryStub) DecrementClaim(context.Context, int64, float64) (*RedPacket, error) {
	return r.decremented, nil
}
func (r *serviceRepositoryStub) MarkExhausted(context.Context, int64) error {
	r.exhausted = true
	return nil
}
func (r *serviceRepositoryStub) CreateClaim(_ context.Context, claim *Claim) error {
	claim.ID = 92
	r.createdClaim = claim
	return nil
}
func (r *serviceRepositoryStub) HasClaimed(context.Context, int64, int64) (bool, error) {
	return false, nil
}
func (r *serviceRepositoryStub) ListClaims(context.Context, int64) ([]Claim, error) { return nil, nil }
func (r *serviceRepositoryStub) ListCreatedBy(context.Context, int64, int, int) ([]RedPacket, int, error) {
	return nil, 0, nil
}
func (r *serviceRepositoryStub) ListClaimedBy(context.Context, int64, int, int) ([]RedPacket, int, error) {
	return nil, 0, nil
}
func (r *serviceRepositoryStub) ListActiveExpired(context.Context, time.Time) ([]RedPacket, error) {
	return r.expired, nil
}
func (r *serviceRepositoryStub) ListAll(context.Context, int, int) ([]RedPacket, int, error) {
	return nil, 0, nil
}
func (r *serviceRepositoryStub) ReturnRemainingIfExpired(context.Context, int64, int64, time.Time) (float64, error) {
	return r.returned, nil
}

type immediateTransaction struct{}

func (immediateTransaction) RunInTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type balanceStub struct {
	debitOK         bool
	debits, credits []contract.BalanceOperation
}

func (b *balanceStub) Credit(_ context.Context, operation contract.BalanceOperation) error {
	b.credits = append(b.credits, operation)
	return nil
}
func (b *balanceStub) DebitIfSufficient(_ context.Context, operation contract.BalanceOperation) (bool, error) {
	b.debits = append(b.debits, operation)
	return b.debitOK, nil
}

type auditStub struct{ entries []contract.AuditEntry }

func (a *auditStub) WriteActivityAudit(_ context.Context, entry contract.AuditEntry) error {
	a.entries = append(a.entries, entry)
	return nil
}

type settingsStub struct {
	settings contract.RedPacketSettings
	err      error
}

func (s settingsStub) GetActivityRedPacketSettings(context.Context) (contract.RedPacketSettings, error) {
	return s.settings, s.err
}

type codeStub string

func (c codeStub) GenerateRedPacketCode(context.Context) (string, error) { return string(c), nil }

type feeStub struct {
	quote FeeQuote
	err   error
}

func (f feeStub) QuoteRedPacketFee(context.Context, int64, float64) (FeeQuote, error) {
	return f.quote, f.err
}

type ledgerStub struct{ id int64 }

func (l ledgerStub) RecordRedPacketClaim(context.Context, int64, int64, int64, float64, time.Time) (int64, error) {
	if l.id == 0 {
		return 1, nil
	}
	return l.id, nil
}

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type expiryStub struct{ calls int }

func (s *expiryStub) RefundExpired(context.Context) (ExpiryRunResult, error) {
	s.calls++
	return ExpiryRunResult{}, nil
}

type leaseCoordinator struct{ acquired bool }

func (l leaseCoordinator) AcquireSingletonLease(context.Context, string, string, time.Duration) (contract.Lease, bool, error) {
	return nil, l.acquired, nil
}
