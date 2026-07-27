package platform

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// BalanceCacheInvalidator is the single cache operation required by Overlay
// modules after a balance mutation.
type BalanceCacheInvalidator interface {
	InvalidateUserBalance(ctx context.Context, userID int64) error
}

// UserConcurrencyWriter is the minimal account capability used by activity
// rewards. It deliberately excludes the broader core user repository API.
type UserConcurrencyWriter interface {
	UpdateConcurrency(ctx context.Context, userID int64, slots int) error
}

// SubscriptionGrant is the transport-neutral subscription adjustment used by
// activity rewards.
type SubscriptionGrant struct {
	UserID  int64
	GroupID int64
	Days    int
	Note    string
}

// SubscriptionManager exposes only the two subscription operations consumed by
// the Activity and Wallet overlays.
type SubscriptionManager interface {
	HasActiveSubscription(ctx context.Context, userID int64) (bool, error)
	AssignOrExtendSubscription(ctx context.Context, grant SubscriptionGrant) error
}

const (
	RedeemRecordStatusUnused = "unused"
	RedeemRecordStatusUsed   = "used"
)

// RedeemRecord is the Overlay-owned projection persisted for activity rewards.
// It prevents feature modules from depending on core service DTOs.
type RedeemRecord struct {
	Code         string
	Type         string
	Value        float64
	Status       string
	UsedBy       *int64
	UsedAt       *time.Time
	Notes        string
	GroupID      *int64
	ValidityDays int
	ExpiresAt    *time.Time
}

// RedeemRecordWriter persists the narrow redeem-code shapes used by activity
// rewards and audit records.
type RedeemRecordWriter interface {
	CreateRedeemRecord(ctx context.Context, record RedeemRecord) error
}

// LeaderLockCache provides the cross-instance lease operations used by the
// red-packet expiry worker.
type LeaderLockCache interface {
	TryAcquireLeaderLock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
	ReleaseLeaderLock(ctx context.Context, key, owner string) error
}

// ActivityWalletCapabilities groups generic application facilities consumed by
// the Activity and Wallet overlays. Feature modules receive only the small
// interfaces they need, never core service or repository types.
type ActivityWalletCapabilities struct {
	BalanceCache  BalanceCacheInvalidator
	Concurrency   UserConcurrencyWriter
	Subscriptions SubscriptionManager
	RedeemRecords RedeemRecordWriter
	LeaderLocks   LeaderLockCache
}

// ProvideActivityWalletCapabilities is the only custom-layer adapter that
// accepts core service/repository dependencies for Activity and Wallet.
func ProvideActivityWalletCapabilities(
	billingCache *service.BillingCacheService,
	users service.UserRepository,
	subscriptions *service.SubscriptionService,
	client *dbent.Client,
	leaderLocks service.LeaderLockCache,
) *ActivityWalletCapabilities {
	capabilities := &ActivityWalletCapabilities{
		Concurrency:   userConcurrencyWriter{users: users},
		Subscriptions: subscriptionManager{service: subscriptions},
		RedeemRecords: redeemRecordWriter{client: client},
	}
	if billingCache != nil {
		capabilities.BalanceCache = billingCache
	}
	if leaderLocks != nil {
		capabilities.LeaderLocks = leaderLocks
	}
	return capabilities
}

type userConcurrencyWriter struct{ users service.UserRepository }

func (w userConcurrencyWriter) UpdateConcurrency(ctx context.Context, userID int64, slots int) error {
	if w.users == nil {
		return service.ErrUserNotFound
	}
	return w.users.UpdateConcurrency(ctx, userID, slots)
}

type subscriptionManager struct{ service *service.SubscriptionService }

func (m subscriptionManager) HasActiveSubscription(ctx context.Context, userID int64) (bool, error) {
	if m.service == nil {
		return false, nil
	}
	subscriptions, err := m.service.ListActiveUserSubscriptions(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(subscriptions) > 0, nil
}

func (m subscriptionManager) AssignOrExtendSubscription(ctx context.Context, grant SubscriptionGrant) error {
	if m.service == nil {
		return service.ErrSubscriptionNotFound
	}
	_, _, err := m.service.AssignOrExtendSubscription(ctx, &service.AssignSubscriptionInput{
		UserID: grant.UserID, GroupID: grant.GroupID, ValidityDays: grant.Days, Notes: grant.Note,
	})
	return err
}

type redeemRecordWriter struct{ client *dbent.Client }

func (w redeemRecordWriter) CreateRedeemRecord(ctx context.Context, record RedeemRecord) error {
	client := w.client
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	if client == nil {
		return service.ErrRedeemCodeNotFound
	}
	_, err := client.RedeemCode.Create().
		SetCode(record.Code).
		SetType(record.Type).
		SetValue(record.Value).
		SetStatus(record.Status).
		SetNotes(record.Notes).
		SetValidityDays(record.ValidityDays).
		SetNillableExpiresAt(record.ExpiresAt).
		SetNillableUsedBy(record.UsedBy).
		SetNillableUsedAt(record.UsedAt).
		SetNillableGroupID(record.GroupID).
		Save(ctx)
	return err
}

var (
	_ BalanceCacheInvalidator = (*service.BillingCacheService)(nil)
	_ UserConcurrencyWriter   = userConcurrencyWriter{}
	_ SubscriptionManager     = subscriptionManager{}
	_ RedeemRecordWriter      = redeemRecordWriter{}
)
