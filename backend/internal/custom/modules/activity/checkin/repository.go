package checkin

import (
	"context"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/checkin"
	"github.com/Wei-Shaw/sub2api/ent/checkinblindboxrecord"
	"github.com/Wei-Shaw/sub2api/ent/checkinprizeitem"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
)

// NewRepository builds the module-owned Ent repository over the compatibility
// checkins table. The table stays in place; only its business ownership moves.
func NewRepository(client *dbent.Client) Repository {
	return &entRepository{client: client}
}

type entRepository struct{ client *dbent.Client }

func (r *entRepository) FindToday(ctx context.Context, userID int64, today time.Time) (*Record, error) {
	item, err := r.clientFor(ctx).Checkin.Query().Where(checkin.UserID(userID), checkin.CheckinDateEQ(today)).Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return recordFromEnt(item), nil
}

func (r *entRepository) FindPrevious(ctx context.Context, userID int64, before time.Time) (*Record, error) {
	item, err := r.clientFor(ctx).Checkin.Query().
		Where(checkin.UserID(userID), checkin.CheckinDateLT(before)).
		Order(dbent.Desc(checkin.FieldCheckinDate)).
		First(ctx)
	if dbent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return recordFromEnt(item), nil
}

func (r *entRepository) Create(ctx context.Context, record *Record) error {
	if record == nil || record.UserID <= 0 || record.CheckinDate.IsZero() || !validFinite(record.RewardAmount) || record.StreakDays <= 0 {
		return fmt.Errorf("invalid checkin record")
	}
	saved, err := r.clientFor(ctx).Checkin.Create().
		SetUserID(record.UserID).
		SetCheckinDate(record.CheckinDate).
		SetRewardAmount(record.RewardAmount).
		SetStreakDays(record.StreakDays).
		SetCheckinType(record.CheckinType).
		SetBetAmount(record.BetAmount).
		SetMultiplier(record.Multiplier).
		Save(ctx)
	if err != nil {
		return err
	}
	record.ID = saved.ID
	return nil
}

func (r *entRepository) ListCalendar(ctx context.Context, userID int64, start, end time.Time) ([]Record, error) {
	items, err := r.clientFor(ctx).Checkin.Query().
		Where(checkin.UserID(userID), checkin.CheckinDateGTE(start), checkin.CheckinDateLTE(end)).
		Order(dbent.Asc(checkin.FieldCheckinDate)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]Record, 0, len(items))
	for _, item := range items {
		if record := recordFromEnt(item); record != nil {
			records = append(records, *record)
		}
	}
	return records, nil
}

func (r *entRepository) LockAccount(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrCheckinNotAllowed
	}
	rows, err := r.clientFor(ctx).QueryContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return ErrCheckinNotAllowed
	}
	return rows.Err()
}

func (r *entRepository) GetLockedAccount(ctx context.Context, userID int64) (contract.Account, error) {
	item, err := r.clientFor(ctx).User.Query().Where(user.IDEQ(userID)).Only(ctx)
	if err != nil {
		return contract.Account{}, err
	}
	return accountFromEnt(item), nil
}

func (r *entRepository) clientFor(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.client
}

func recordFromEnt(item *dbent.Checkin) *Record {
	if item == nil {
		return nil
	}
	return &Record{
		ID: item.ID, UserID: item.UserID, CheckinDate: item.CheckinDate, RewardAmount: item.RewardAmount,
		StreakDays: item.StreakDays, CheckinType: item.CheckinType, BetAmount: item.BetAmount, Multiplier: item.Multiplier,
	}
}

// NewEntAccountReader exposes only the account state activity needs. It is a
// platform adapter, not a dependency on the shared user repository.
func NewEntAccountReader(client *dbent.Client) contract.AccountReader {
	return entAccountReader{client: client}
}

type entAccountReader struct{ client *dbent.Client }

func (r entAccountReader) GetAccount(ctx context.Context, userID int64) (contract.Account, error) {
	if r.client == nil || userID <= 0 {
		return contract.Account{}, fmt.Errorf("invalid account lookup")
	}
	client := r.client
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	item, err := client.User.Query().Where(user.IDEQ(userID)).Only(ctx)
	if err != nil {
		return contract.Account{}, err
	}
	return accountFromEnt(item), nil
}

func accountFromEnt(item *dbent.User) contract.Account {
	if item == nil {
		return contract.Account{}
	}
	return contract.Account{ID: item.ID, Role: item.Role, Status: item.Status, Balance: item.Balance}
}

// NewEntBalanceWriter adapts the spendable users.balance column to the narrow
// activity accounting port. It deliberately never updates total_recharged.
func NewEntBalanceWriter(client *dbent.Client) contract.BalanceWriter {
	return entBalanceWriter{client: client}
}

type entBalanceWriter struct{ client *dbent.Client }

func (w entBalanceWriter) Credit(ctx context.Context, operation contract.BalanceOperation) error {
	if w.client == nil || operation.UserID <= 0 || !validFinite(operation.Amount) || operation.Amount <= 0 {
		return fmt.Errorf("invalid checkin balance credit")
	}
	updated, err := entClient(ctx, w.client).User.Update().Where(user.IDEQ(operation.UserID)).AddBalance(operation.Amount).Save(ctx)
	if err != nil {
		return err
	}
	if updated != 1 {
		return ErrCheckinNotAllowed
	}
	return nil
}

func (w entBalanceWriter) DebitIfSufficient(ctx context.Context, operation contract.BalanceOperation) (bool, error) {
	if w.client == nil || operation.UserID <= 0 || !validFinite(operation.Amount) || operation.Amount <= 0 {
		return false, fmt.Errorf("invalid checkin balance debit")
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

// EntCheckinLedger persists compatibility audit rows while hiding RedeemCode
// from the business service. Code creation is supplied as a narrow platform port.
type EntCheckinLedger struct {
	client   *dbent.Client
	codes    CheckinCodeGenerator
	metadata RedeemMetadataStore
}

func NewEntCheckinLedger(client *dbent.Client, codes CheckinCodeGenerator) *EntCheckinLedger {
	return &EntCheckinLedger{client: client, codes: codes, metadata: NewRedeemMetadataStore(client)}
}

func (l *EntCheckinLedger) RecordCheckinAdjustment(ctx context.Context, entry CheckinAuditEntry) error {
	if l == nil || l.client == nil || l.codes == nil || l.metadata == nil || entry.UserID <= 0 || entry.Type == "" || !validFinite(entry.Amount) {
		return fmt.Errorf("invalid checkin audit entry")
	}
	code, err := l.codes.GenerateCheckinCode(ctx, entry.Type)
	if err != nil {
		return err
	}
	usedAt := entry.OccurredAt
	if usedAt.IsZero() {
		usedAt = time.Now()
	}
	saved, err := entClient(ctx, l.client).RedeemCode.Create().
		SetCode(code).
		SetType(entry.Type).
		SetValue(entry.Amount).
		SetStatus("used").
		SetUsedBy(entry.UserID).
		SetUsedAt(usedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	return l.metadata.Store(ctx, saved.ID, entry.Multiplier, entry.BetAmount)
}

// NewEntBlindboxRecordsReader replaces the check-in route's dependency on the
// legacy BlindBoxService for read-only user history.
func NewEntBlindboxRecordsReader(client *dbent.Client) contract.BlindboxRecordsReader {
	return entBlindboxRecordsReader{client: client}
}

type entBlindboxRecordsReader struct{ client *dbent.Client }

func (r entBlindboxRecordsReader) GetUserRecords(ctx context.Context, userID int64, page, pageSize int) (*contract.BlindboxRecordList, error) {
	if r.client == nil || userID <= 0 {
		return nil, fmt.Errorf("invalid blindbox record query")
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	client := entClient(ctx, r.client)
	query := client.CheckinBlindboxRecord.Query().Where(checkinblindboxrecord.UserID(userID))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count records: %w", err)
	}
	items, err := query.Order(dbent.Desc(checkinblindboxrecord.FieldCreatedAt)).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query records: %w", err)
	}
	records := make([]contract.BlindboxRecord, 0, len(items))
	for _, item := range items {
		record := contract.BlindboxRecord{
			ID: item.ID, PrizeName: item.PrizeName, Rarity: item.Rarity, RewardType: item.RewardType,
			RewardValue: item.RewardValue, RewardDetail: item.RewardDetail, StreakDays: item.StreakDays,
			CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if item.RewardType == "subscription" && item.PrizeItemID > 0 {
			prize, prizeErr := client.CheckinPrizeItem.Query().Where(checkinprizeitem.IDEQ(item.PrizeItemID)).Only(ctx)
			if prizeErr == nil {
				record.SubscriptionDays = prize.SubscriptionDays
			}
		}
		records = append(records, record)
	}
	return &contract.BlindboxRecordList{Items: records, Total: int64(total)}, nil
}

func entClient(ctx context.Context, client *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return client
}

func validFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
