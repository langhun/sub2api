package redpacket

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

// Repository owns red-packet persistence. Methods that mutate a packet are
// deliberately split into locked operations: callers must invoke them inside a
// contract.TransactionRunner transaction and must not replace either lock with
// a read-then-write sequence.
type Repository interface {
	Create(ctx context.Context, packet *RedPacket) error
	FindByCode(ctx context.Context, code string) (*RedPacket, error)
	FindByCodeForUpdate(ctx context.Context, code string) (*RedPacket, error)
	FindByID(ctx context.Context, redPacketID int64) (*RedPacket, error)
	DecrementClaim(ctx context.Context, redPacketID int64, amount float64) (*RedPacket, error)
	MarkExhausted(ctx context.Context, redPacketID int64) error
	CreateClaim(ctx context.Context, claim *Claim) error
	HasClaimed(ctx context.Context, redPacketID, userID int64) (bool, error)
	ListClaims(ctx context.Context, redPacketID int64) ([]Claim, error)
	ListCreatedBy(ctx context.Context, senderID int64, page, pageSize int) ([]RedPacket, int, error)
	ListClaimedBy(ctx context.Context, userID int64, page, pageSize int) ([]RedPacket, int, error)
	ListActiveExpired(ctx context.Context, now time.Time) ([]RedPacket, error)
	ListAll(ctx context.Context, page, pageSize int) ([]RedPacket, int, error)
	ReturnRemainingIfExpired(ctx context.Context, redPacketID int64, senderID int64, now time.Time) (float64, error)
}

// CodeGenerator generates a share code before a packet is persisted. The
// repository remains responsible for enforcing code uniqueness.
type CodeGenerator interface {
	GenerateRedPacketCode(ctx context.Context) (string, error)
}

// FeeQuote is the fee applied when a sender funds a red packet. The amount is
// independent from the Activity feature toggle and is provided by a narrow
// core billing adapter so legacy VIP fee policy remains configurable.
type FeeQuote struct {
	Rate   float64
	Amount float64
}

type FeeQuoter interface {
	QuoteRedPacketFee(ctx context.Context, senderID int64, totalAmount float64) (FeeQuote, error)
}

// ClaimLedger preserves the existing balance-transfer history record created
// for a successful red-packet claim. It is not a direct-transfer service.
type ClaimLedger interface {
	RecordRedPacketClaim(ctx context.Context, redPacketID, senderID, receiverID int64, amount float64, occurredAt time.Time) (int64, error)
}

// Clock makes expiration decisions testable without coupling the module to the
// wall clock.
type Clock interface {
	Now() time.Time
}

// Dependencies are the core adapters required by a future red-packet runtime.
// The balance, audit, and repository writes for each mutation must share the
// transaction context supplied by Transactions. Cache invalidation and
// notifications, when needed, happen only after that transaction commits.
type Dependencies struct {
	Repository   Repository
	Transactions contract.TransactionRunner
	Balance      contract.BalanceWriter
	Audit        contract.AuditWriter
	Settings     contract.RedPacketSettingsReader
	Code         CodeGenerator
	Fees         FeeQuoter
	Ledger       ClaimLedger
	Clock        Clock
	Random       RandomSource
}

func (d Dependencies) validateRuntime() error {
	switch {
	case d.Repository == nil:
		return fmt.Errorf("red-packet repository is required")
	case d.Transactions == nil:
		return fmt.Errorf("red-packet transaction runner is required")
	case d.Balance == nil:
		return fmt.Errorf("red-packet balance writer is required")
	case d.Settings == nil:
		return fmt.Errorf("red-packet settings reader is required")
	case d.Code == nil:
		return fmt.Errorf("red-packet code generator is required")
	case d.Fees == nil:
		return fmt.Errorf("red-packet fee quoter is required")
	case d.Ledger == nil:
		return fmt.Errorf("red-packet claim ledger is required")
	}
	return nil
}

// ExpiryWorker owns only worker lifecycle. Runtime wiring will provide a
// singleton lease so multiple service instances do not refund the same packet.
type ExpiryWorker interface {
	Start()
	Stop()
}

// ExpiryWorkerDependencies are the adapters required to construct the runtime
// worker. Expire is intentionally a narrow port so the worker cannot reach
// balance or persistence directly.
type ExpiryWorkerDependencies struct {
	Expire contractExpiryRefunder
	Leases contract.SingletonLeaseCoordinator
	Clock  Clock
	Config ExpiryWorkerConfig
}

// ExpiryWorkerConfig records the operational values supplied by Runtime. The
// default lock key intentionally matches the legacy worker during a rolling
// migration, so two released versions cannot refund one packet concurrently.
type ExpiryWorkerConfig struct {
	Interval time.Duration
	LeaseKey string
	LeaseTTL time.Duration
}

// contractExpiryRefunder keeps the worker independent from query and mutation
// methods that are unrelated to periodic expiry.
type contractExpiryRefunder interface {
	RefundExpired(ctx context.Context) (ExpiryRunResult, error)
}
