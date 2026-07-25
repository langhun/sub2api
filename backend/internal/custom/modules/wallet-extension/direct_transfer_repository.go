package walletextension

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/balancetransfer"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// directTransferRepository persists the direct-transfer slice in the existing
// balance_transfers ledger. That table is the immutable audit record for this
// compatibility phase; no schema ownership moves with this module.
type directTransferRepository struct{ client *dbent.Client }

// NewDirectTransferRepository constructs the direct-only ledger adapter.
func NewDirectTransferRepository(client *dbent.Client) DirectTransferRepository {
	return &directTransferRepository{client: client}
}

func (r *directTransferRepository) CommitDirectTransfer(ctx context.Context, plan DirectTransferCommitPlan) (DirectTransferRecord, error) {
	if r == nil || r.client == nil {
		return DirectTransferRecord{}, fmt.Errorf("direct transfer repository is unavailable")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return DirectTransferRecord{}, fmt.Errorf("begin direct transfer transaction: %w", err)
	}
	txCtx := dbent.NewTxContext(ctx, tx)
	defer func() { _ = tx.Rollback() }()

	client := directClientFromContext(txCtx, r.client)
	if err := lockDirectTransferSender(txCtx, client, plan.SenderID); err != nil {
		return DirectTransferRecord{}, err
	}
	dailyTotal, dailyCount, err := directTransferDailyUsage(txCtx, client, plan.SenderID, startOfLocalDay(), endOfLocalDay())
	if err != nil {
		return DirectTransferRecord{}, fmt.Errorf("check direct transfer daily limit: %w", err)
	}
	if plan.DailyLimit > 0 && dailyTotal+plan.GrossAmount > plan.DailyLimit {
		return DirectTransferRecord{}, ErrTransferDailyLimit
	}
	if plan.DailyCountLimit > 0 && dailyCount >= plan.DailyCountLimit {
		return DirectTransferRecord{}, ErrTransferDailyCount
	}
	if ok, err := debitDirectTransferBalance(txCtx, client, plan.SenderID, plan.GrossAmount); err != nil {
		return DirectTransferRecord{}, fmt.Errorf("debit direct transfer sender: %w", err)
	} else if !ok {
		return DirectTransferRecord{}, ErrTransferInsufficient
	}
	if err := creditDirectTransferBalance(txCtx, client, plan.ReceiverID, plan.Amount); err != nil {
		return DirectTransferRecord{}, fmt.Errorf("credit direct transfer recipient: %w", err)
	}
	record := DirectTransferRecord{
		SenderID: plan.SenderID, ReceiverID: plan.ReceiverID, Amount: plan.Amount, Fee: plan.Fee, FeeRate: plan.FeeRate,
		GrossAmount: plan.GrossAmount, TransferType: DirectTransferType, Status: "completed", Memo: plan.Memo, CreatedAt: time.Now(),
	}
	if err := r.create(txCtx, client, &record); err != nil {
		return DirectTransferRecord{}, err
	}
	if err := tx.Commit(); err != nil {
		return DirectTransferRecord{}, fmt.Errorf("commit direct transfer transaction: %w", err)
	}
	return r.withDisplays(ctx, directClientFromContext(ctx, r.client), record)
}

func (r *directTransferRepository) CreateDirectTransfer(ctx context.Context, record *DirectTransferRecord) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("direct transfer repository is unavailable")
	}
	return r.create(ctx, directClientFromContext(ctx, r.client), record)
}

func (r *directTransferRepository) create(ctx context.Context, client *dbent.Client, record *DirectTransferRecord) error {
	if record == nil {
		return fmt.Errorf("direct transfer record is required")
	}
	builder := client.BalanceTransfer.Create().
		SetSenderID(record.SenderID).
		SetReceiverID(record.ReceiverID).
		SetAmount(record.Amount).
		SetFee(record.Fee).
		SetFeeRate(record.FeeRate).
		SetGrossAmount(record.GrossAmount).
		SetTransferType(DirectTransferType).
		SetStatus(record.Status)
	if record.Memo != nil {
		builder.SetMemo(*record.Memo)
	}
	if !record.CreatedAt.IsZero() {
		builder.SetCreatedAt(record.CreatedAt)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("create direct transfer ledger entry: %w", err)
	}
	record.ID = item.ID
	record.TransferType = DirectTransferType
	return nil
}

func (r *directTransferRepository) GetDirectTransfer(ctx context.Context, transferID int64) (DirectTransferRecord, error) {
	if r == nil || r.client == nil {
		return DirectTransferRecord{}, fmt.Errorf("direct transfer repository is unavailable")
	}
	client := directClientFromContext(ctx, r.client)
	item, err := client.BalanceTransfer.Query().Where(
		balancetransfer.IDEQ(transferID),
		balancetransfer.TransferTypeEQ(DirectTransferType),
	).Only(ctx)
	if err != nil {
		return DirectTransferRecord{}, fmt.Errorf("get direct transfer %d: %w", transferID, err)
	}
	return r.withDisplays(ctx, client, directTransferRecordFromEntity(item))
}

func (r *directTransferRepository) ListDirectTransferHistory(ctx context.Context, query DirectTransferHistoryQuery) ([]DirectTransferRecord, int, error) {
	if r == nil || r.client == nil {
		return nil, 0, fmt.Errorf("direct transfer repository is unavailable")
	}
	client := directClientFromContext(ctx, r.client)
	predicates := []predicate.BalanceTransfer{balancetransfer.TransferTypeEQ(DirectTransferType)}
	switch query.Role {
	case "sender":
		predicates = append(predicates, balancetransfer.SenderIDEQ(query.AccountID))
	case "receiver":
		predicates = append(predicates, balancetransfer.ReceiverIDEQ(query.AccountID))
	default:
		predicates = append(predicates, balancetransfer.Or(balancetransfer.SenderIDEQ(query.AccountID), balancetransfer.ReceiverIDEQ(query.AccountID)))
	}
	base := client.BalanceTransfer.Query().Where(predicates...)
	total, err := base.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count direct transfer history: %w", err)
	}
	page, pageSize := normalizeDirectTransferPage(query.Page, query.PageSize)
	offset := (&pagination.PaginationParams{Page: page, PageSize: pageSize}).Offset()
	items, err := base.Order(dbent.Desc(balancetransfer.FieldCreatedAt)).Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list direct transfer history: %w", err)
	}
	result := make([]DirectTransferRecord, 0, len(items))
	for _, item := range items {
		record, err := r.withDisplays(ctx, client, directTransferRecordFromEntity(item))
		if err != nil {
			return nil, 0, err
		}
		result = append(result, record)
	}
	return result, total, nil
}

func (r *directTransferRepository) GetDirectTransferDailyUsage(ctx context.Context, senderID int64, start, end time.Time) (float64, int, error) {
	if r == nil || r.client == nil {
		return 0, 0, fmt.Errorf("direct transfer repository is unavailable")
	}
	return directTransferDailyUsage(ctx, directClientFromContext(ctx, r.client), senderID, start, end)
}

func (r *directTransferRepository) GetDirectTransferStats(ctx context.Context, accountID int64) (DirectTransferStats, error) {
	if r == nil || r.client == nil {
		return DirectTransferStats{}, fmt.Errorf("direct transfer repository is unavailable")
	}
	client := directClientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `SELECT
    COALESCE((SELECT SUM(amount) FROM balance_transfers WHERE sender_id = $1 AND transfer_type = $2 AND status != 'revoked'), 0),
    COALESCE((SELECT SUM(amount) FROM balance_transfers WHERE receiver_id = $1 AND transfer_type = $2 AND status != 'revoked'), 0),
	COALESCE((SELECT SUM(fee) FROM balance_transfers WHERE sender_id = $1 AND transfer_type = $2 AND status != 'revoked'), 0)`, accountID, DirectTransferType)
	if err != nil {
		return DirectTransferStats{}, fmt.Errorf("get direct transfer stats: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return DirectTransferStats{}, fmt.Errorf("get direct transfer stats: %w", err)
		}
		return DirectTransferStats{}, sql.ErrNoRows
	}
	var result DirectTransferStats
	if err := rows.Scan(&result.TotalSent, &result.TotalReceived, &result.TotalFeePaid); err != nil {
		return DirectTransferStats{}, fmt.Errorf("get direct transfer stats: %w", err)
	}
	return result, nil
}

func (r *directTransferRepository) withDisplays(ctx context.Context, client *dbent.Client, record DirectTransferRecord) (DirectTransferRecord, error) {
	sender, err := client.User.Get(ctx, record.SenderID)
	if err == nil && sender != nil {
		record.SenderDisplay = service.MaskedUserDisplayName(sender.Username, sender.Email, sender.ID)
	} else {
		record.SenderDisplay = service.UserDisplayName("", "", record.SenderID)
	}
	receiver, err := client.User.Get(ctx, record.ReceiverID)
	if err == nil && receiver != nil {
		record.ReceiverDisplay = service.MaskedUserDisplayName(receiver.Username, receiver.Email, receiver.ID)
	} else {
		record.ReceiverDisplay = service.UserDisplayName("", "", record.ReceiverID)
	}
	return record, nil
}

func directTransferRecordFromEntity(item *dbent.BalanceTransfer) DirectTransferRecord {
	return DirectTransferRecord{
		ID: item.ID, SenderID: item.SenderID, ReceiverID: item.ReceiverID, Amount: item.Amount, Fee: item.Fee,
		FeeRate: item.FeeRate, GrossAmount: item.GrossAmount, TransferType: item.TransferType, Status: item.Status,
		Memo: item.Memo, FrozenAt: item.FrozenAt, FrozenBy: item.FrozenBy, RevokeReason: item.RevokeReason, CreatedAt: item.CreatedAt,
	}
}

func directClientFromContext(ctx context.Context, fallback *dbent.Client) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return fallback
}

func lockDirectTransferSender(ctx context.Context, client *dbent.Client, senderID int64) error {
	rows, err := client.QueryContext(ctx, `SELECT id FROM users WHERE id = $1 AND deleted_at IS NULL FOR UPDATE`, senderID)
	if err != nil {
		return fmt.Errorf("lock direct transfer sender: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return rows.Err()
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return fmt.Errorf("direct transfer sender not found")
}

func directTransferDailyUsage(ctx context.Context, client *dbent.Client, senderID int64, start, end time.Time) (float64, int, error) {
	var amount float64
	var count int
	rows, err := client.QueryContext(ctx, `SELECT COALESCE(SUM(gross_amount), 0), COALESCE(COUNT(*), 0)
FROM balance_transfers
WHERE sender_id = $1 AND transfer_type = $2 AND status != 'revoked' AND created_at >= $3 AND created_at < $4`, senderID, DirectTransferType, start, end)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, 0, err
		}
		return 0, 0, sql.ErrNoRows
	}
	if err := rows.Scan(&amount, &count); err != nil {
		return 0, 0, err
	}
	return amount, count, rows.Err()
}

func debitDirectTransferBalance(ctx context.Context, client *dbent.Client, senderID int64, amount float64) (bool, error) {
	result, err := client.ExecContext(ctx, `UPDATE users SET balance = balance - $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, amount, senderID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected == 1, err
}

func creditDirectTransferBalance(ctx context.Context, client *dbent.Client, receiverID int64, amount float64) error {
	result, err := client.ExecContext(ctx, `UPDATE users SET balance = balance + $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL`, amount, receiverID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("direct transfer recipient not found")
	}
	return nil
}

func startOfLocalDay() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func endOfLocalDay() time.Time { return startOfLocalDay().AddDate(0, 0, 1) }

func normalizeDirectTransferPage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}
