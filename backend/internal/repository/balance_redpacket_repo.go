package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/balanceredpacket"
	"github.com/Wei-Shaw/sub2api/ent/balanceredpacketclaim"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type balanceRedPacketRepo struct {
	client *dbent.Client
	db     *sql.DB
}

func NewBalanceRedPacketRepository(client *dbent.Client, db *sql.DB) service.BalanceRedPacketRepository {
	return &balanceRedPacketRepo{client: client, db: db}
}

func (r *balanceRedPacketRepo) Create(ctx context.Context, rp *service.RedPacketRecord) error {
	client := clientFromContext(ctx, r.client)
	saved, err := client.BalanceRedPacket.Create().
		SetSenderID(rp.SenderID).
		SetTotalAmount(rp.TotalAmount).
		SetTotalCount(rp.TotalCount).
		SetRemainingAmount(rp.RemainingAmount).
		SetRemainingCount(rp.RemainingCount).
		SetRedpacketType(rp.RedPacketType).
		SetFee(rp.Fee).
		SetFeeRate(rp.FeeRate).
		SetCode(rp.Code).
		SetStatus(rp.Status).
		SetExpireAt(rp.ExpireAt).
		SetNillableMemo(rp.Memo).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create red packet: %w", err)
	}
	rp.ID = saved.ID
	return nil
}

func (r *balanceRedPacketRepo) GetByCode(ctx context.Context, code string) (*service.RedPacketRecord, error) {
	client := clientFromContext(ctx, r.client)
	rp, err := client.BalanceRedPacket.Query().Where(balanceredpacket.Code(code)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("get red packet by code %s: %w", code, err)
	}
	return toRedPacketRecord(rp), nil
}

func (r *balanceRedPacketRepo) GetByID(ctx context.Context, id int64) (*service.RedPacketRecord, error) {
	client := clientFromContext(ctx, r.client)
	rp, err := client.BalanceRedPacket.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get red packet %d: %w", id, err)
	}
	return toRedPacketRecord(rp), nil
}

func (r *balanceRedPacketRepo) DecrementClaim(ctx context.Context, id int64, amount float64) error {
	client := clientFromContext(ctx, r.client)
	result, err := client.ExecContext(ctx,
		"UPDATE balance_redpackets SET remaining_amount = remaining_amount - $1, remaining_count = remaining_count - 1 WHERE id = $2 AND remaining_count > 0 AND remaining_amount >= $1 AND status = 'active'",
		amount, id,
	)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("red packet exhausted or not available")
	}
	return nil
}

func (r *balanceRedPacketRepo) MarkExhausted(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.BalanceRedPacket.UpdateOneID(id).
		SetStatus("exhausted").
		SetRemainingAmount(0).
		SetRemainingCount(0).
		Save(ctx)
	return err
}

func (r *balanceRedPacketRepo) MarkExpired(ctx context.Context, id int64) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.BalanceRedPacket.UpdateOneID(id).
		SetStatus("expired").
		Save(ctx)
	return err
}

func (r *balanceRedPacketRepo) CreateClaim(ctx context.Context, claim *service.RedPacketClaimRecord) error {
	client := clientFromContext(ctx, r.client)
	saved, err := client.BalanceRedPacketClaim.Create().
		SetRedpacketID(claim.RedPacketID).
		SetUserID(claim.UserID).
		SetAmount(claim.Amount).
		SetNillableTransferID(claim.TransferID).
		Save(ctx)
	if err != nil {
		return fmt.Errorf("create red packet claim: %w", err)
	}
	claim.ID = saved.ID
	return nil
}

func (r *balanceRedPacketRepo) HasClaimed(ctx context.Context, redpacketID, userID int64) (bool, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.BalanceRedPacketClaim.Query().
		Where(
			balanceredpacketclaim.RedpacketID(redpacketID),
			balanceredpacketclaim.UserID(userID),
		).Count(ctx)
	return count > 0, err
}

func (r *balanceRedPacketRepo) GetClaims(ctx context.Context, redpacketID int64) ([]*service.RedPacketClaimRecord, error) {
	client := clientFromContext(ctx, r.client)
	items, err := client.BalanceRedPacketClaim.Query().
		Where(balanceredpacketclaim.RedpacketID(redpacketID)).
		Order(dbent.Asc(balanceredpacketclaim.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	claims := make([]*service.RedPacketClaimRecord, len(items))
	for i, c := range items {
		claims[i] = &service.RedPacketClaimRecord{
			ID:          c.ID,
			RedPacketID: c.RedpacketID,
			UserID:      c.UserID,
			Amount:      c.Amount,
			TransferID:  c.TransferID,
			CreatedAt:   c.CreatedAt,
		}
	}
	return claims, nil
}

func (r *balanceRedPacketRepo) ListBySender(ctx context.Context, senderID int64, page, pageSize int) ([]*service.RedPacketRecord, int, error) {
	client := clientFromContext(ctx, r.client)
	pred := balanceredpacket.SenderID(senderID)
	query := client.BalanceRedPacket.Query().Where(pred).Order(dbent.Desc(balanceredpacket.FieldCreatedAt))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset := (&pagination.PaginationParams{Page: page, PageSize: pageSize}).Offset()
	items, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	records := make([]*service.RedPacketRecord, len(items))
	for i, item := range items {
		records[i] = toRedPacketRecord(item)
	}
	return records, total, nil
}

func (r *balanceRedPacketRepo) ListActiveExpired(ctx context.Context) ([]*service.RedPacketRecord, error) {
	client := clientFromContext(ctx, r.client)
	items, err := client.BalanceRedPacket.Query().
		Where(
			balanceredpacket.StatusEQ("active"),
			balanceredpacket.ExpireAtLT(time.Now()),
			balanceredpacket.RemainingCountGT(0),
		).All(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]*service.RedPacketRecord, len(items))
	for i, item := range items {
		records[i] = toRedPacketRecord(item)
	}
	return records, nil
}

func (r *balanceRedPacketRepo) ListAll(ctx context.Context, page, pageSize int) ([]*service.RedPacketRecord, int, error) {
	client := clientFromContext(ctx, r.client)
	var preds []predicate.BalanceRedPacket
	query := client.BalanceRedPacket.Query().Where(preds...).Order(dbent.Desc(balanceredpacket.FieldCreatedAt))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset := (&pagination.PaginationParams{Page: page, PageSize: pageSize}).Offset()
	items, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	records := make([]*service.RedPacketRecord, len(items))
	for i, item := range items {
		records[i] = toRedPacketRecord(item)
	}
	return records, total, nil
}

func (r *balanceRedPacketRepo) ReturnRemaining(ctx context.Context, id int64, senderID int64) (float64, error) {
	client := clientFromContext(ctx, r.client)
	rows, err := client.QueryContext(ctx, `
WITH expired AS (
	SELECT id, remaining_amount
	FROM balance_redpackets
	WHERE id = $1
	  AND sender_id = $2
	  AND status = 'active'
	  AND remaining_amount > 0
	  AND remaining_count > 0
	FOR UPDATE
),
updated AS (
	UPDATE balance_redpackets rp
	SET status = 'expired', remaining_amount = 0, remaining_count = 0
	FROM expired
	WHERE rp.id = expired.id
	RETURNING expired.remaining_amount
)
SELECT remaining_amount FROM updated
`, id, senderID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return 0, err
		}
		return 0, nil
	}

	var remaining float64
	if err := rows.Scan(&remaining); err != nil {
		return 0, err
	}
	return math.Round(remaining*1e8) / 1e8, rows.Err()
}

func toRedPacketRecord(rp *dbent.BalanceRedPacket) *service.RedPacketRecord {
	return &service.RedPacketRecord{
		ID:              rp.ID,
		SenderID:        rp.SenderID,
		TotalAmount:     rp.TotalAmount,
		TotalCount:      rp.TotalCount,
		RemainingAmount: rp.RemainingAmount,
		RemainingCount:  rp.RemainingCount,
		RedPacketType:   rp.RedpacketType,
		Fee:             rp.Fee,
		FeeRate:         rp.FeeRate,
		Code:            rp.Code,
		Status:          rp.Status,
		Memo:            rp.Memo,
		ExpireAt:        rp.ExpireAt,
		CreatedAt:       rp.CreatedAt,
	}
}
