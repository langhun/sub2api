package rewards

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	customsettings "github.com/Wei-Shaw/sub2api/internal/custom/settings"
	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// ProductionDependencies are the narrow core ports required to assemble the
// extracted activity reward module.
type ProductionDependencies struct {
	Client        *dbent.Client
	DB            *sql.DB
	Settings      *customsettings.Registry
	CodeGenerator CodeGenerator
	Concurrency   platform.UserConcurrencyWriter
	Subscriptions platform.SubscriptionManager
	RedeemRecords platform.RedeemRecordWriter
	BalanceCache  platform.BalanceCacheInvalidator
}

// NewProduction returns the activity-owned HTTP module and worker lifecycle.
// The caller starts and stops the returned Runtime with the existing server
// lifecycle, exactly as the legacy reward worker was managed.
func NewProduction(deps ProductionDependencies, options WorkerOptions) (*Module, *Runtime) {
	prizes := NewEntPrizeCatalog(deps.Client)
	outbox := NewOutboxRepository(deps.Client, deps.DB)
	processor := NewDeliveryProcessor(ProcessorDependencies{
		Balance:      NewEntBalanceWriter(deps.Client),
		Concurrency:  platformConcurrencyGranter{writer: deps.Concurrency},
		Subscription: platformSubscriptionGranter{manager: deps.Subscriptions},
		Invitation:   platformInvitationIssuer{writer: deps.RedeemRecords},
		Audit:        platformAuditWriter{codeGenerator: deps.CodeGenerator, writer: deps.RedeemRecords},
		History:      entHistoryWriter{client: deps.Client},
		Cache:        platformBalanceCacheInvalidator{cache: deps.BalanceCache},
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

type platformConcurrencyGranter struct {
	writer platform.UserConcurrencyWriter
}

func (g platformConcurrencyGranter) GrantConcurrency(ctx context.Context, grant contract.ConcurrencyGrant) error {
	if g.writer == nil || grant.UserID <= 0 || grant.Slots < 0 {
		return ErrUnavailable
	}
	return g.writer.UpdateConcurrency(ctx, grant.UserID, grant.Slots)
}

type platformSubscriptionGranter struct{ manager platform.SubscriptionManager }

func (g platformSubscriptionGranter) GrantOrExtendSubscription(ctx context.Context, grant contract.SubscriptionGrant) error {
	if g.manager == nil {
		return ErrUnavailable
	}
	return g.manager.AssignOrExtendSubscription(ctx, platform.SubscriptionGrant{
		UserID: grant.UserID, GroupID: grant.SubscriptionID, Days: grant.Days, Note: grant.Note,
	})
}

type platformInvitationIssuer struct{ writer platform.RedeemRecordWriter }

func (i platformInvitationIssuer) IssueInvitationCode(ctx context.Context, request contract.InvitationCodeRequest) (string, error) {
	if i.writer == nil || request.UserID <= 0 || request.Code == "" {
		return "", ErrUnavailable
	}
	if err := i.writer.CreateRedeemRecord(ctx, platform.RedeemRecord{
		Code: request.Code, Type: domain.RedeemTypeInvitation, Value: 0, Status: platform.RedeemRecordStatusUnused, ExpiresAt: request.ExpiresAt,
	}); err != nil {
		return "", err
	}
	return request.Code, nil
}

type platformAuditWriter struct {
	codeGenerator CodeGenerator
	writer        platform.RedeemRecordWriter
}

func (w platformAuditWriter) WriteActivityAudit(ctx context.Context, entry contract.AuditEntry) error {
	if w.writer == nil || entry.UserID <= 0 {
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
	return w.writer.CreateRedeemRecord(ctx, platform.RedeemRecord{
		Code: code, Type: SourceCheckinBlindbox, Value: entry.Amount, Status: platform.RedeemRecordStatusUsed,
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

type platformBalanceCacheInvalidator struct {
	cache platform.BalanceCacheInvalidator
}

func (i platformBalanceCacheInvalidator) InvalidateBalance(ctx context.Context, userID int64) error {
	if i.cache == nil {
		return nil
	}
	return i.cache.InvalidateUserBalance(ctx, userID)
}

var (
	_ contract.SettingsReader          = registrySettingsAdapter{}
	_ CheckinCounter                   = checkinCounter{}
	_ contract.ConcurrencyGranter      = platformConcurrencyGranter{}
	_ contract.SubscriptionGranter     = platformSubscriptionGranter{}
	_ contract.InvitationCodeIssuer    = platformInvitationIssuer{}
	_ contract.AuditWriter             = platformAuditWriter{}
	_ BlindboxRecordWriter             = entHistoryWriter{}
	_ contract.BalanceCacheInvalidator = platformBalanceCacheInvalidator{}
)
