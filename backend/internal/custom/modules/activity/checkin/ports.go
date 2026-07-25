package checkin

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

const (
	checkinTypeNormal   = "normal"
	checkinTypeLuck     = "luck"
	accountStatusActive = "active"

	adjustmentTypeCheckin     = "checkin"
	adjustmentTypeCheckinLuck = "checkin_luck"
)

// Record is the activity-owned representation of a persisted check-in.
// Keeping the Ent entity behind this type lets the module retain its storage
// contract while the existing table remains an upgrade compatibility surface.
type Record struct {
	ID           int64
	UserID       int64
	CheckinDate  time.Time
	RewardAmount float64
	StreakDays   int
	CheckinType  string
	BetAmount    float64
	Multiplier   float64
}

// Repository owns check-in persistence and the row lock that serializes a
// user's daily check-in. All mutating calls must receive the transaction
// context supplied by contract.TransactionRunner.
type Repository interface {
	FindToday(ctx context.Context, userID int64, today time.Time) (*Record, error)
	FindPrevious(ctx context.Context, userID int64, before time.Time) (*Record, error)
	Create(ctx context.Context, record *Record) error
	ListCalendar(ctx context.Context, userID int64, start, end time.Time) ([]Record, error)
	LockAccount(ctx context.Context, userID int64) error
	GetLockedAccount(ctx context.Context, userID int64) (contract.Account, error)
}

// CheckinAuditEntry is the narrow ledger record needed for normal and luck
// check-ins. Implementations store it in the caller's transaction context.
type CheckinAuditEntry struct {
	UserID         int64
	Type           string
	Amount         float64
	Multiplier     float64
	BetAmount      float64
	IdempotencyKey string
	OccurredAt     time.Time
}

// CheckinLedger records balance-adjustment history without exposing a shared
// redeem-code repository to the check-in service.
type CheckinLedger interface {
	RecordCheckinAdjustment(ctx context.Context, entry CheckinAuditEntry) error
}

// CheckinCodeGenerator creates a ledger code for a check-in adjustment. It is
// a platform port so this module does not depend on the shared settings service.
type CheckinCodeGenerator interface {
	GenerateCheckinCode(ctx context.Context, adjustmentType string) (string, error)
}

// PreparedBlindbox is the response-safe result of scheduling a blind-box
// reward. RewardDetail remains empty until delivery has completed.
type PreparedBlindbox struct {
	DeliveryID int64
	Result     contract.BlindboxResult
}

// DeliveredBlindbox describes the only post-delivery value the check-in
// response needs. Delivery errors are intentionally best effort after the
// check-in transaction has committed.
type DeliveredBlindbox struct {
	RewardDetail string
}

// BlindboxDelivery keeps check-in independent of any legacy blind-box service.
// Its implementation may use the activity rewards module, but it must preserve
// the caller's transaction context while preparing a delivery.
type BlindboxDelivery interface {
	PrepareForCheckin(ctx context.Context, userID, checkinID int64, streakDays int) (*PreparedBlindbox, error)
	Deliver(ctx context.Context, deliveryID int64) (*DeliveredBlindbox, error)
}

// Clock separates local activity-day boundaries from wall-clock audit times.
type Clock interface {
	Today() time.Time
	Now() time.Time
}

// RandomSource provides the uniform random value used for rewards and luck
// multipliers. Implementations must return a value in [0, 1).
type RandomSource interface {
	Float64() (float64, error)
}

// Dependencies are the platform capabilities required by the operational
// check-in module. None of these ports is a legacy check-in or blind-box service.
type Dependencies struct {
	Repository   Repository
	Transactions contract.TransactionRunner
	Accounts     contract.AccountReader
	Balance      contract.BalanceWriter
	Ledger       CheckinLedger
	Settings     contract.SettingsReader
	Cache        contract.BalanceCacheInvalidator
	Blindbox     BlindboxDelivery
	Clock        Clock
	Random       RandomSource
}

func (d Dependencies) validate() error {
	switch {
	case d.Repository == nil:
		return fmt.Errorf("check-in repository is required")
	case d.Transactions == nil:
		return fmt.Errorf("check-in transaction runner is required")
	case d.Accounts == nil:
		return fmt.Errorf("check-in account reader is required")
	case d.Balance == nil:
		return fmt.Errorf("check-in balance writer is required")
	case d.Ledger == nil:
		return fmt.Errorf("check-in ledger is required")
	case d.Settings == nil:
		return fmt.Errorf("check-in settings reader is required")
	}
	return nil
}
