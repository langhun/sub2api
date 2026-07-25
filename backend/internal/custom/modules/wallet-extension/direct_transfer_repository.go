package walletextension

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/balancetransfer"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/custom/platform"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

// directTransferRepository persists the direct-transfer slice in the existing
// balance_transfers ledger. That table is the immutable audit record for this
// compatibility phase; no schema ownership moves with this module.
type directTransferRepository struct{ client *dbent.Client }

var _ TransferAdministrationRepository = (*directTransferRepository)(nil)

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

func (r *directTransferRepository) ListAllTransfers(ctx context.Context, filter TransferFilter, page, pageSize int) ([]TransferRecord, int, error) {
	if r == nil || r.client == nil {
		return nil, 0, fmt.Errorf("transfer repository is unavailable")
	}
	client := directClientFromContext(ctx, r.client)
	predicates := make([]predicate.BalanceTransfer, 0, 5)
	if filter.Status != "" {
		predicates = append(predicates, balancetransfer.StatusEQ(filter.Status))
	}
	if filter.TransferType != "" {
		predicates = append(predicates, balancetransfer.TransferTypeEQ(filter.TransferType))
	}
	if filter.UserID != nil {
		predicates = append(predicates, balancetransfer.Or(
			balancetransfer.SenderIDEQ(*filter.UserID),
			balancetransfer.ReceiverIDEQ(*filter.UserID),
		))
	}
	if !filter.StartTime.IsZero() {
		predicates = append(predicates, balancetransfer.CreatedAtGTE(filter.StartTime))
	}
	if !filter.EndTime.IsZero() {
		predicates = append(predicates, balancetransfer.CreatedAtLTE(filter.EndTime))
	}
	query := client.BalanceTransfer.Query().Where(predicates...)
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count transfers: %w", err)
	}
	page, pageSize = normalizeDirectTransferPage(page, pageSize)
	offset := (&pagination.PaginationParams{Page: page, PageSize: pageSize}).Offset()
	items, err := query.Order(dbent.Desc(balancetransfer.FieldCreatedAt)).Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list transfers: %w", err)
	}
	result := make([]TransferRecord, 0, len(items))
	for _, item := range items {
		record, err := r.withTransferDisplays(ctx, client, transferRecordFromEntity(item))
		if err != nil {
			return nil, 0, err
		}
		result = append(result, record)
	}
	return result, total, nil
}

func (r *directTransferRepository) GetTransferForUpdate(ctx context.Context, transferID int64) (TransferRecord, error) {
	if r == nil || r.client == nil {
		return TransferRecord{}, fmt.Errorf("transfer repository is unavailable")
	}
	item, err := directClientFromContext(ctx, r.client).BalanceTransfer.Query().
		Where(balancetransfer.IDEQ(transferID)).ForUpdate().Only(ctx)
	if err != nil {
		return TransferRecord{}, fmt.Errorf("get transfer %d for update: %w", transferID, err)
	}
	return transferRecordFromEntity(item), nil
}

func (r *directTransferRepository) UpdateTransferStatus(ctx context.Context, transferID int64, status string, frozenAt *time.Time, frozenBy *int64, revokeReason *string) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("transfer repository is unavailable")
	}
	builder := directClientFromContext(ctx, r.client).BalanceTransfer.UpdateOneID(transferID).SetStatus(status)
	if frozenAt != nil {
		builder.SetFrozenAt(*frozenAt)
	}
	if frozenBy != nil {
		builder.SetFrozenBy(*frozenBy)
	}
	if revokeReason != nil {
		builder.SetRevokeReason(*revokeReason)
	}
	_, err := builder.Save(ctx)
	return err
}

func (r *directTransferRepository) DebitBalanceIfSufficient(ctx context.Context, userID int64, amount float64) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("transfer repository is unavailable")
	}
	result, err := directClientFromContext(ctx, r.client).ExecContext(ctx, `
UPDATE users
SET balance = balance - $1, updated_at = NOW()
WHERE id = $2 AND deleted_at IS NULL AND balance >= $1`, amount, userID)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected == 1, nil
}

func (r *directTransferRepository) CreateTransfer(ctx context.Context, record *TransferRecord) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("transfer repository is unavailable")
	}
	if record == nil || record.TransferType == "" {
		return fmt.Errorf("transfer record is required")
	}
	builder := directClientFromContext(ctx, r.client).BalanceTransfer.Create().
		SetSenderID(record.SenderID).
		SetReceiverID(record.ReceiverID).
		SetAmount(record.Amount).
		SetFee(record.Fee).
		SetFeeRate(record.FeeRate).
		SetGrossAmount(record.GrossAmount).
		SetTransferType(record.TransferType).
		SetStatus(record.Status)
	if record.Memo != nil {
		builder.SetMemo(*record.Memo)
	}
	if record.RedpacketID != nil {
		builder.SetRedpacketID(*record.RedpacketID)
	}
	if !record.CreatedAt.IsZero() {
		builder.SetCreatedAt(record.CreatedAt)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return fmt.Errorf("create transfer ledger entry: %w", err)
	}
	record.ID = item.ID
	return nil
}

func (r *directTransferRepository) GetTransferFeeStats(ctx context.Context, startTime, endTime time.Time) ([]DailyFeeStat, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("transfer repository is unavailable")
	}
	rows, err := directClientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT DATE(created_at) AS day, COALESCE(SUM(fee), 0) AS total_fee, COUNT(*) AS count
FROM balance_transfers
WHERE status = 'completed' AND created_at >= $1 AND created_at < $2
GROUP BY DATE(created_at)
ORDER BY day`, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	stats := make([]DailyFeeStat, 0)
	for rows.Next() {
		var stat DailyFeeStat
		if err := rows.Scan(&stat.Date, &stat.TotalFee, &stat.Count); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

func (r *directTransferRepository) GetTransferLeaderboard(ctx context.Context, startTime, endTime time.Time, limit int) ([]TransferRankEntry, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("transfer repository is unavailable")
	}
	rows, err := directClientFromContext(ctx, r.client).QueryContext(ctx, `
SELECT u.id, u.username, u.email, COALESCE(SUM(bt.amount), 0) AS total_amount, COUNT(*) AS total_count
FROM balance_transfers bt
JOIN users u ON u.id = bt.sender_id
WHERE bt.status = 'completed' AND bt.transfer_type = 'direct' AND bt.created_at >= $1 AND bt.created_at < $2
GROUP BY u.id, u.username, u.email
ORDER BY SUM(bt.amount) DESC
LIMIT $3`, startTime, endTime, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	entries := make([]TransferRankEntry, 0)
	for rank := 1; rows.Next(); rank++ {
		var entry TransferRankEntry
		var username string
		if err := rows.Scan(&entry.UserID, &username, &entry.Email, &entry.TotalAmount, &entry.TotalCount); err != nil {
			return nil, err
		}
		entry.Rank = rank
		entry.DisplayName = platform.MaskedUserDisplayName(username, entry.Email, entry.UserID)
		entry.Email = maskTransferLeaderboardEmail(entry.Email)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (r *directTransferRepository) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("transfer repository is unavailable")
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin transfer transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(dbent.NewTxContext(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *directTransferRepository) withDisplays(ctx context.Context, client *dbent.Client, record DirectTransferRecord) (DirectTransferRecord, error) {
	sender, err := client.User.Get(ctx, record.SenderID)
	if err == nil && sender != nil {
		record.SenderDisplay = platform.MaskedUserDisplayName(sender.Username, sender.Email, sender.ID)
	} else {
		record.SenderDisplay = platform.UserDisplayName("", "", record.SenderID)
	}
	receiver, err := client.User.Get(ctx, record.ReceiverID)
	if err == nil && receiver != nil {
		record.ReceiverDisplay = platform.MaskedUserDisplayName(receiver.Username, receiver.Email, receiver.ID)
	} else {
		record.ReceiverDisplay = platform.UserDisplayName("", "", record.ReceiverID)
	}
	return record, nil
}

func (r *directTransferRepository) withTransferDisplays(ctx context.Context, client *dbent.Client, record TransferRecord) (TransferRecord, error) {
	sender, err := client.User.Get(ctx, record.SenderID)
	if err == nil && sender != nil {
		record.SenderDisplay = platform.MaskedUserDisplayName(sender.Username, sender.Email, sender.ID)
	} else {
		record.SenderDisplay = platform.UserDisplayName("", "", record.SenderID)
	}
	receiver, err := client.User.Get(ctx, record.ReceiverID)
	if err == nil && receiver != nil {
		record.ReceiverDisplay = platform.MaskedUserDisplayName(receiver.Username, receiver.Email, receiver.ID)
	} else {
		record.ReceiverDisplay = platform.UserDisplayName("", "", record.ReceiverID)
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

func transferRecordFromEntity(item *dbent.BalanceTransfer) TransferRecord {
	return TransferRecord{
		ID: item.ID, SenderID: item.SenderID, ReceiverID: item.ReceiverID, Amount: item.Amount, Fee: item.Fee,
		FeeRate: item.FeeRate, GrossAmount: item.GrossAmount, TransferType: item.TransferType, Status: item.Status,
		Memo: item.Memo, RedpacketID: item.RedpacketID, FrozenAt: item.FrozenAt, FrozenBy: item.FrozenBy,
		RevokeReason: item.RevokeReason, CreatedAt: item.CreatedAt,
	}
}

func maskTransferLeaderboardEmail(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 0 {
		return "u***"
	}
	return email[:1] + "***" + email[at:]
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
