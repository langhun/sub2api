package rewards

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// ProductionDependencies are the narrow core ports required to assemble the
// extracted activity reward module.
type ProductionDependencies struct {
	Client        *dbent.Client
	DB            *sql.DB
	Settings      *customsettings.Registry
	CodeGenerator CodeGenerator
	Users         service.UserRepository
	Subscriptions *service.SubscriptionService
	RedeemCodes   service.RedeemCodeRepository
	BillingCache  *service.BillingCacheService
}

// NewProduction returns the activity-owned HTTP module and worker lifecycle.
// The caller starts and stops the returned Runtime with the existing server
// lifecycle, exactly as the legacy reward worker was managed.
func NewProduction(deps ProductionDependencies, options WorkerOptions) (*Module, *Runtime) {
	prizes := NewEntPrizeCatalog(deps.Client)
	outbox := NewOutboxRepository(deps.Client, deps.DB)
	processor := NewDeliveryProcessor(ProcessorDependencies{
		Balance:      coreBalanceWriter{users: deps.Users},
		Concurrency:  coreConcurrencyGranter{users: deps.Users},
		Subscription: coreSubscriptionGranter{service: deps.Subscriptions},
		Invitation:   coreInvitationIssuer{codes: deps.RedeemCodes},
		Audit:        coreAuditWriter{codeGenerator: deps.CodeGenerator, codes: deps.RedeemCodes},
		History:      entHistoryWriter{client: deps.Client},
		Cache:        coreBalanceCacheInvalidator{cache: deps.BillingCache},
	})
	worker := NewWorker(outbox, processor, options)
	module := NewModule(NewHandler(NewAdminService(prizes, outbox, worker)))
	module.Outbox = outbox
	module.Runner = worker
	module.Rewards = NewService(ServiceDependencies{
		Settings: NewRegistrySettingsAdapter(deps.Settings),
		Checkins: NewCheckinCounter(deps.Client, deps.DB),
		Prizes:   prizes,
		Outbox:   outbox,
		Codes:    codeFormatInvitationGenerator{source: deps.CodeGenerator},
	})
	return module, NewRuntime(worker)
}

// NewRegistrySettingsAdapter projects the activity-owned slice of the Overlay
// settings registry into the rewards contract. The reward module therefore
// never reads feature values from the legacy SettingService.
func NewRegistrySettingsAdapter(registry *customsettings.Registry) contract.SettingsReader {
	return registrySettingsAdapter{registry: registry}
}

type registrySettingsAdapter struct{ registry *customsettings.Registry }

func (a registrySettingsAdapter) GetActivitySettings(ctx context.Context) (contract.Settings, error) {
	if a.registry == nil {
		return contract.Settings{}, ErrUnavailable
	}
	snapshot, err := a.registry.Read(ctx)
	if err != nil {
		return contract.Settings{}, fmt.Errorf("read custom activity settings: %w", err)
	}
	settings := snapshot.Activity
	return contract.Settings{
		Checkin: contract.CheckinSettings{
			Enabled:           settings.CheckinEnabled,
			MinimumReward:     settings.CheckinMinBalance,
			MaximumReward:     settings.CheckinMaxBalance,
			LuckEnabled:       settings.CheckinLuckEnabled,
			MinimumMultiplier: settings.CheckinLuckMinMultiplier,
			MaximumMultiplier: settings.CheckinLuckMaxMultiplier,
		},
		Blindbox: contract.BlindboxSettings{
			Enabled:     settings.CheckinBlindboxEnabled,
			TriggerType: settings.CheckinBlindboxTriggerType,
			Interval:    settings.CheckinBlindboxInterval,
		},
	}, nil
}

// CodeGenerator is the sole platform capability required for code-format
// generation. The composition root may adapt SettingService to this port.
type CodeGenerator interface {
	GenerateCode(context.Context, string) (string, error)
}

type codeFormatInvitationGenerator struct{ source CodeGenerator }

func (g codeFormatInvitationGenerator) GenerateInvitationCode(ctx context.Context) (string, error) {
	if g.source == nil {
		return "", ErrUnavailable
	}
	return g.source.GenerateCode(ctx, domain.RedeemTypeInvitation)
}

type checkinCounter struct {
	client *dbent.Client
	db     *sql.DB
}

func NewCheckinCounter(client *dbent.Client, db *sql.DB) CheckinCounter {
	return checkinCounter{client: client, db: db}
}

func (c checkinCounter) CountCheckins(ctx context.Context, userID int64) (int, error) {
	if userID <= 0 {
		return 0, ErrInvalidDelivery
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		rows, err := tx.Client().QueryContext(ctx, `SELECT COUNT(*) FROM checkins WHERE user_id = $1`, userID)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		if !rows.Next() {
			return 0, rows.Err()
		}
		var total int
		return total, rows.Scan(&total)
	}
	if c.db == nil {
		return 0, ErrUnavailable
	}
	var total int
	if err := c.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM checkins WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}

type nonRechargeBalanceUpdater interface {
	UpdateBalanceWithoutRecharge(context.Context, int64, float64) error
}

type coreBalanceWriter struct{ users service.UserRepository }

func (w coreBalanceWriter) Credit(ctx context.Context, operation contract.BalanceOperation) error {
	if w.users == nil || operation.UserID <= 0 || operation.Amount < 0 {
		return ErrUnavailable
	}
	if updater, ok := w.users.(nonRechargeBalanceUpdater); ok {
		return updater.UpdateBalanceWithoutRecharge(ctx, operation.UserID, operation.Amount)
	}
	return w.users.UpdateBalance(ctx, operation.UserID, operation.Amount)
}

func (w coreBalanceWriter) DebitIfSufficient(context.Context, contract.BalanceOperation) (bool, error) {
	return false, fmt.Errorf("reward balance writer does not debit")
}

type coreConcurrencyGranter struct{ users service.UserRepository }

func (g coreConcurrencyGranter) GrantConcurrency(ctx context.Context, grant contract.ConcurrencyGrant) error {
	if g.users == nil || grant.UserID <= 0 || grant.Slots < 0 {
		return ErrUnavailable
	}
	return g.users.UpdateConcurrency(ctx, grant.UserID, grant.Slots)
}

type coreSubscriptionGranter struct{ service *service.SubscriptionService }

func (g coreSubscriptionGranter) GrantOrExtendSubscription(ctx context.Context, grant contract.SubscriptionGrant) error {
	if g.service == nil {
		return ErrUnavailable
	}
	_, _, err := g.service.AssignOrExtendSubscription(ctx, &service.AssignSubscriptionInput{
		UserID: grant.UserID, GroupID: grant.SubscriptionID, ValidityDays: grant.Days, Notes: grant.Note,
	})
	return err
}

type coreInvitationIssuer struct{ codes service.RedeemCodeRepository }

func (i coreInvitationIssuer) IssueInvitationCode(ctx context.Context, request contract.InvitationCodeRequest) (string, error) {
	if i.codes == nil || request.UserID <= 0 || request.Code == "" {
		return "", ErrUnavailable
	}
	if err := i.codes.Create(ctx, &service.RedeemCode{
		Code: request.Code, Type: domain.RedeemTypeInvitation, Value: 0, Status: service.StatusUnused, ExpiresAt: request.ExpiresAt,
	}); err != nil {
		return "", err
	}
	return request.Code, nil
}

type coreAuditWriter struct {
	codeGenerator CodeGenerator
	codes         service.RedeemCodeRepository
}

func (w coreAuditWriter) WriteActivityAudit(ctx context.Context, entry contract.AuditEntry) error {
	if w.codes == nil || entry.UserID <= 0 {
		return ErrUnavailable
	}
	codeType := entry.CodeType
	if codeType == "" {
		codeType = entry.Type
	}
	if w.codeGenerator == nil {
		return ErrUnavailable
	}
	code, err := w.codeGenerator.GenerateCode(ctx, codeType)
	if err != nil {
		return err
	}
	now := time.Now()
	return w.codes.Create(ctx, &service.RedeemCode{
		Code: code, Type: SourceCheckinBlindbox, Value: entry.Amount, Status: service.StatusUsed,
		UsedBy: &entry.UserID, UsedAt: &now, Notes: entry.Notes,
		GroupID: entry.GroupID, ValidityDays: entry.ValidityDays,
	})
}

type entHistoryWriter struct{ client *dbent.Client }

func (w entHistoryWriter) RecordBlindboxDelivery(ctx context.Context, record BlindboxRecord) error {
	if w.client == nil || record.UserID <= 0 || record.PrizeID <= 0 {
		return ErrUnavailable
	}
	client := w.client
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	_, err := client.CheckinBlindboxRecord.Create().
		SetUserID(record.UserID).
		SetPrizeItemID(record.PrizeID).
		SetPrizeName(record.PrizeName).
		SetRarity(string(record.Rarity)).
		SetRewardType(string(record.RewardType)).
		SetRewardValue(record.RewardValue).
		SetRewardDetail(record.RewardDetail).
		SetStreakDays(record.StreakDays).
		Save(ctx)
	return err
}

type coreBalanceCacheInvalidator struct{ cache *service.BillingCacheService }

func (i coreBalanceCacheInvalidator) InvalidateBalance(ctx context.Context, userID int64) error {
	if i.cache == nil {
		return nil
	}
	return i.cache.InvalidateUserBalance(ctx, userID)
}

var (
	_ contract.SettingsReader          = registrySettingsAdapter{}
	_ CheckinCounter                   = checkinCounter{}
	_ contract.BalanceWriter           = coreBalanceWriter{}
	_ contract.ConcurrencyGranter      = coreConcurrencyGranter{}
	_ contract.SubscriptionGranter     = coreSubscriptionGranter{}
	_ contract.InvitationCodeIssuer    = coreInvitationIssuer{}
	_ contract.AuditWriter             = coreAuditWriter{}
	_ BlindboxRecordWriter             = entHistoryWriter{}
	_ contract.BalanceCacheInvalidator = coreBalanceCacheInvalidator{}
)
