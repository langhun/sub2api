package redpacket

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/balanceredpacket"
	"github.com/Wei-Shaw/sub2api/ent/balanceredpacketclaim"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type entRepository struct{ entClient *dbent.Client }

// NewRepository returns the module-owned Ent implementation over the existing
// red-packet tables. No schema or migration change is required for extraction.
func NewRepository(client *dbent.Client) Repository {
	return &entRepository{entClient: client}
}

func (r *entRepository) Create(ctx context.Context, packet *RedPacket) error {
	if packet == nil {
		return fmt.Errorf("red packet is required")
	}
	saved, err := r.client(ctx).BalanceRedPacket.Create().
		SetSenderID(packet.SenderID).
		SetTotalAmount(packet.TotalAmount).
		SetTotalCount(packet.TotalCount).
		SetRemainingAmount(packet.RemainingAmount).
		SetRemainingCount(packet.RemainingCount).
		SetRedpacketType(string(packet.Type)).
		SetFee(packet.Fee).
		SetFeeRate(packet.FeeRate).
		SetCode(packet.Code).
		SetStatus(string(packet.Status)).
		SetNillableMemo(packet.Memo).
		SetExpireAt(packet.ExpiresAt).
		SetCreatedAt(packet.CreatedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	packet.ID = saved.ID
	return nil
}

func (r *entRepository) FindByCode(ctx context.Context, code string) (*RedPacket, error) {
	item, err := r.client(ctx).BalanceRedPacket.Query().Where(balanceredpacket.CodeEqualFold(code)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return packetFromEnt(item), nil
}

func (r *entRepository) FindByCodeForUpdate(ctx context.Context, code string) (*RedPacket, error) {
	item, err := r.client(ctx).BalanceRedPacket.Query().Where(balanceredpacket.CodeEqualFold(code)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, err
	}
	return packetFromEnt(item), nil
}

func (r *entRepository) FindByID(ctx context.Context, redPacketID int64) (*RedPacket, error) {
	item, err := r.client(ctx).BalanceRedPacket.Get(ctx, redPacketID)
	if err != nil {
		return nil, err
	}
	return packetFromEnt(item), nil
}

func (r *entRepository) DecrementClaim(ctx context.Context, redPacketID int64, amount float64) (*RedPacket, error) {
	client := r.client(ctx)
	rows, err := client.QueryContext(ctx, `
		UPDATE balance_redpackets
		SET remaining_amount = remaining_amount - $1, remaining_count = remaining_count - 1
		WHERE id = $2 AND remaining_count > 0 AND remaining_amount >= $1 AND status = 'active'
		RETURNING id, sender_id, total_amount, total_count, remaining_amount, remaining_count,
		          redpacket_type, fee, fee_rate, code, status, memo, expire_at, created_at`, amount, redPacketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, ErrExhausted
	}
	var packet RedPacket
	var packetType, status string
	if err := rows.Scan(&packet.ID, &packet.SenderID, &packet.TotalAmount, &packet.TotalCount,
		&packet.RemainingAmount, &packet.RemainingCount, &packetType, &packet.Fee, &packet.FeeRate,
		&packet.Code, &status, &packet.Memo, &packet.ExpiresAt, &packet.CreatedAt); err != nil {
		return nil, err
	}
	packet.Type = Type(packetType)
	packet.Status = Status(status)
	return &packet, rows.Err()
}

func (r *entRepository) MarkExhausted(ctx context.Context, redPacketID int64) error {
	_, err := r.client(ctx).BalanceRedPacket.UpdateOneID(redPacketID).
		SetStatus(string(StatusExhausted)).
		SetRemainingAmount(0).
		SetRemainingCount(0).
		Save(ctx)
	return err
}

func (r *entRepository) CreateClaim(ctx context.Context, claim *Claim) error {
	if claim == nil {
		return fmt.Errorf("red-packet claim is required")
	}
	saved, err := r.client(ctx).BalanceRedPacketClaim.Create().
		SetRedpacketID(claim.RedPacketID).
		SetUserID(claim.UserID).
		SetAmount(claim.Amount).
		SetNillableTransferID(claim.AuditID).
		SetCreatedAt(claim.CreatedAt).
		Save(ctx)
	if err != nil {
		if dbent.IsConstraintError(err) {
			return errClaimConflict
		}
		return err
	}
	claim.ID = saved.ID
	return nil
}

func (r *entRepository) HasClaimed(ctx context.Context, redPacketID, userID int64) (bool, error) {
	count, err := r.client(ctx).BalanceRedPacketClaim.Query().Where(
		balanceredpacketclaim.RedpacketID(redPacketID), balanceredpacketclaim.UserID(userID),
	).Count(ctx)
	return count > 0, err
}

func (r *entRepository) ListClaims(ctx context.Context, redPacketID int64) ([]Claim, error) {
	client := r.client(ctx)
	items, err := client.BalanceRedPacketClaim.Query().Where(balanceredpacketclaim.RedpacketID(redPacketID)).
		Order(dbent.Asc(balanceredpacketclaim.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	claims := make([]Claim, 0, len(items))
	userIDs := make([]int64, 0, len(items))
	for _, item := range items {
		claims = append(claims, Claim{ID: item.ID, RedPacketID: item.RedpacketID, UserID: item.UserID,
			Amount: item.Amount, AuditID: item.TransferID, CreatedAt: item.CreatedAt})
		userIDs = append(userIDs, item.UserID)
	}
	displays, err := r.userDisplays(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	for index := range claims {
		claims[index].UserDisplay = displays[claims[index].UserID]
		if claims[index].UserDisplay == "" {
			claims[index].UserDisplay = service.UserDisplayName("", "", claims[index].UserID)
		}
	}
	return claims, nil
}

func (r *entRepository) ListCreatedBy(ctx context.Context, senderID int64, page, pageSize int) ([]RedPacket, int, error) {
	return r.list(ctx, r.client(ctx).BalanceRedPacket.Query().Where(balanceredpacket.SenderID(senderID)), page, pageSize)
}

func (r *entRepository) ListClaimedBy(ctx context.Context, userID int64, page, pageSize int) ([]RedPacket, int, error) {
	query := r.client(ctx).BalanceRedPacket.Query().Where(balanceredpacket.HasClaimsWith(balanceredpacketclaim.UserID(userID)))
	items, total, err := r.list(ctx, query, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	for index := range items {
		// A recipient must not receive the share code from the history endpoint.
		items[index].Code = ""
	}
	return items, total, nil
}

func (r *entRepository) ListActiveExpired(ctx context.Context, now time.Time) ([]RedPacket, error) {
	items, err := r.client(ctx).BalanceRedPacket.Query().Where(
		balanceredpacket.StatusEQ(string(StatusActive)),
		balanceredpacket.ExpireAtLT(now),
		balanceredpacket.RemainingCountGT(0),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	return packetsFromEnt(items), nil
}

func (r *entRepository) ListAll(ctx context.Context, page, pageSize int) ([]RedPacket, int, error) {
	return r.list(ctx, r.client(ctx).BalanceRedPacket.Query(), page, pageSize)
}

func (r *entRepository) ReturnRemainingIfExpired(ctx context.Context, redPacketID, senderID int64, now time.Time) (float64, error) {
	client := r.client(ctx)
	packet, err := client.BalanceRedPacket.Query().Where(balanceredpacket.IDEQ(redPacketID)).ForUpdate().Only(ctx)
	if err != nil {
		return 0, err
	}
	if packet.SenderID != senderID || packet.Status != string(StatusActive) || !now.After(packet.ExpireAt) {
		return 0, nil
	}
	remaining := roundAmount(packet.RemainingAmount)
	if remaining <= 0 {
		return 0, nil
	}
	_, err = client.BalanceRedPacket.UpdateOneID(packet.ID).
		SetStatus(string(StatusExpired)).SetRemainingAmount(0).SetRemainingCount(0).Save(ctx)
	if err != nil {
		return 0, err
	}
	return remaining, nil
}

func (r *entRepository) list(ctx context.Context, query *dbent.BalanceRedPacketQuery, page, pageSize int) ([]RedPacket, int, error) {
	query = query.Order(dbent.Desc(balanceredpacket.FieldCreatedAt))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset := (&pagination.PaginationParams{Page: page, PageSize: pageSize}).Offset()
	items, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return packetsFromEnt(items), total, nil
}

func (r *entRepository) userDisplays(ctx context.Context, userIDs []int64) (map[int64]string, error) {
	if len(userIDs) == 0 {
		return map[int64]string{}, nil
	}
	items, err := r.client(ctx).User.Query().Where(user.IDIn(userIDs...)).All(ctx)
	if err != nil {
		return nil, err
	}
	displays := make(map[int64]string, len(items))
	for _, item := range items {
		displays[item.ID] = service.UserDisplayName(item.Username, item.Email, item.ID)
	}
	return displays, nil
}

func (r *entRepository) client(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.entClient
}

func packetFromEnt(item *dbent.BalanceRedPacket) *RedPacket {
	if item == nil {
		return nil
	}
	return &RedPacket{ID: item.ID, SenderID: item.SenderID, TotalAmount: item.TotalAmount, TotalCount: item.TotalCount,
		RemainingAmount: item.RemainingAmount, RemainingCount: item.RemainingCount, Type: Type(item.RedpacketType), Fee: item.Fee,
		FeeRate: item.FeeRate, Code: item.Code, Status: Status(item.Status), Memo: item.Memo, ExpiresAt: item.ExpireAt, CreatedAt: item.CreatedAt}
}

func packetsFromEnt(items []*dbent.BalanceRedPacket) []RedPacket {
	packets := make([]RedPacket, 0, len(items))
	for _, item := range items {
		if packet := packetFromEnt(item); packet != nil {
			packets = append(packets, *packet)
		}
	}
	return packets
}
