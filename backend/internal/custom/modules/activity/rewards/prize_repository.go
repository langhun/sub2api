package rewards

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/checkinprizeitem"
)

// EntPrizeCatalog owns activity access to the existing blind-box prize and
// history tables. The tables remain compatible records; no schema change is
// required to move their behavior behind the activity boundary.
type EntPrizeCatalog struct{ client *dbent.Client }

func NewEntPrizeCatalog(client *dbent.Client) *EntPrizeCatalog {
	return &EntPrizeCatalog{client: client}
}

func (r *EntPrizeCatalog) ListEnabled(ctx context.Context) ([]Prize, error) {
	if r == nil || r.client == nil {
		return nil, ErrUnavailable
	}
	items, err := r.entClient(ctx).CheckinPrizeItem.Query().Where(
		checkinprizeitem.IsEnabled(true),
		checkinprizeitem.DeletedAtIsNil(),
	).All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query enabled prize items: %w", err)
	}
	return prizesFromEnt(items), nil
}

func (r *EntPrizeCatalog) List(ctx context.Context) ([]Prize, error) {
	if r == nil || r.client == nil {
		return nil, ErrUnavailable
	}
	items, err := r.entClient(ctx).CheckinPrizeItem.Query().
		Where(checkinprizeitem.DeletedAtIsNil()).
		Order(dbent.Desc(checkinprizeitem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query prize items: %w", err)
	}
	return prizesFromEnt(items), nil
}

func (r *EntPrizeCatalog) Get(ctx context.Context, prizeID int64) (*Prize, error) {
	if r == nil || r.client == nil {
		return nil, ErrUnavailable
	}
	item, err := r.entClient(ctx).CheckinPrizeItem.Get(ctx, prizeID)
	if err != nil {
		return nil, err
	}
	prize := prizeFromEnt(item)
	return &prize, nil
}

func (r *EntPrizeCatalog) Save(ctx context.Context, prize Prize) (Prize, error) {
	if r == nil || r.client == nil {
		return Prize{}, ErrUnavailable
	}
	if err := prize.Validate(); err != nil {
		return Prize{}, err
	}
	client := r.entClient(ctx)
	if prize.ID == 0 {
		item, err := client.CheckinPrizeItem.Create().
			SetName(prize.Name).
			SetRarity(string(prize.Rarity)).
			SetRewardType(string(prize.RewardType)).
			SetRewardValue(prize.RewardValue).
			SetRewardValueMax(prize.RewardValueMax).
			SetNillableSubscriptionID(prize.SubscriptionID).
			SetSubscriptionDays(prize.SubscriptionDays).
			SetWeight(prize.Weight).
			SetIsEnabled(prize.Enabled).
			Save(ctx)
		if err != nil {
			return Prize{}, fmt.Errorf("create prize item: %w", err)
		}
		return prizeFromEnt(item), nil
	}
	item, err := client.CheckinPrizeItem.UpdateOneID(prize.ID).
		SetName(prize.Name).
		SetRarity(string(prize.Rarity)).
		SetRewardType(string(prize.RewardType)).
		SetRewardValue(prize.RewardValue).
		SetRewardValueMax(prize.RewardValueMax).
		SetNillableSubscriptionID(prize.SubscriptionID).
		SetSubscriptionDays(prize.SubscriptionDays).
		SetWeight(prize.Weight).
		SetIsEnabled(prize.Enabled).
		Save(ctx)
	if err != nil {
		return Prize{}, fmt.Errorf("update prize item: %w", err)
	}
	return prizeFromEnt(item), nil
}

func (r *EntPrizeCatalog) Archive(ctx context.Context, prizeID int64) error {
	if r == nil || r.client == nil {
		return ErrUnavailable
	}
	return r.entClient(ctx).CheckinPrizeItem.UpdateOneID(prizeID).SetDeletedAt(time.Now()).Exec(ctx)
}

func (r *EntPrizeCatalog) Stats(ctx context.Context) (PrizeStats, error) {
	if r == nil || r.client == nil {
		return PrizeStats{}, ErrUnavailable
	}
	client := r.entClient(ctx)
	totalItems, err := client.CheckinPrizeItem.Query().Where(checkinprizeitem.DeletedAtIsNil()).Count(ctx)
	if err != nil {
		return PrizeStats{}, fmt.Errorf("count prize items: %w", err)
	}
	enabledItems, err := client.CheckinPrizeItem.Query().Where(
		checkinprizeitem.DeletedAtIsNil(),
		checkinprizeitem.IsEnabled(true),
	).Count(ctx)
	if err != nil {
		return PrizeStats{}, fmt.Errorf("count enabled prize items: %w", err)
	}
	totalDraws, err := client.CheckinBlindboxRecord.Query().Count(ctx)
	if err != nil {
		return PrizeStats{}, fmt.Errorf("count blind-box records: %w", err)
	}
	return PrizeStats{TotalItems: totalItems, EnabledItems: enabledItems, TotalDraws: totalDraws}, nil
}

func (r *EntPrizeCatalog) entClient(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.client
}

func prizeFromEnt(item *dbent.CheckinPrizeItem) Prize {
	if item == nil {
		return Prize{}
	}
	return Prize{
		ID: item.ID, Name: item.Name, Rarity: Rarity(item.Rarity), RewardType: RewardType(item.RewardType),
		RewardValue: item.RewardValue, RewardValueMax: item.RewardValueMax, SubscriptionID: item.SubscriptionID,
		SubscriptionDays: item.SubscriptionDays, Weight: item.Weight, Enabled: item.IsEnabled,
		CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func prizesFromEnt(items []*dbent.CheckinPrizeItem) []Prize {
	prizes := make([]Prize, 0, len(items))
	for _, item := range items {
		prizes = append(prizes, prizeFromEnt(item))
	}
	return prizes
}

var _ PrizeCatalog = (*EntPrizeCatalog)(nil)
