package service

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/checkinblindboxrecord"
	"github.com/Wei-Shaw/sub2api/ent/checkinprizeitem"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

var ErrBlindboxPrizeInvalid = infraerrors.BadRequest("BLINDBOX_PRIZE_INVALID", "blind box prize configuration is invalid")

var (
	secureRandomIntN = func(n int) (int, error) {
		if n <= 0 {
			return 0, fmt.Errorf("random upper bound must be positive")
		}
		value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
		if err != nil {
			return 0, err
		}
		return int(value.Int64()), nil
	}
	secureRandomFloat64 = func() (float64, error) {
		const precision = int64(1) << 53
		value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(precision))
		if err != nil {
			return 0, err
		}
		return float64(value.Int64()) / float64(precision), nil
	}
)

const (
	BlindboxRewardBalance        = "balance"
	BlindboxRewardConcurrency    = "concurrency"
	BlindboxRewardSubscription   = "subscription"
	BlindboxRewardInvitationCode = "invitation_code"

	RarityCommon    = "common"
	RarityRare      = "rare"
	RarityEpic      = "epic"
	RarityLegendary = "legendary"
)

type PrizeItem struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	Rarity           string  `json:"rarity"`
	RewardType       string  `json:"reward_type"`
	RewardValue      float64 `json:"reward_value"`
	RewardValueMax   float64 `json:"reward_value_max"`
	SubscriptionID   *int64  `json:"subscription_id,omitempty"`
	SubscriptionDays int     `json:"subscription_days"`
	Weight           int     `json:"weight"`
	IsEnabled        bool    `json:"is_enabled"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type BlindboxResult struct {
	PrizeName        string  `json:"prize_name"`
	Rarity           string  `json:"rarity"`
	RewardType       string  `json:"reward_type"`
	RewardValue      float64 `json:"reward_value"`
	SubscriptionDays int     `json:"subscription_days,omitempty"`
	RewardDetail     string  `json:"reward_detail,omitempty"`
}

type BlindboxRecord struct {
	ID               int64   `json:"id"`
	PrizeName        string  `json:"prize_name"`
	Rarity           string  `json:"rarity"`
	RewardType       string  `json:"reward_type"`
	RewardValue      float64 `json:"reward_value"`
	RewardDetail     string  `json:"reward_detail,omitempty"`
	SubscriptionDays int     `json:"subscription_days,omitempty"`
	StreakDays       int     `json:"streak_days"`
	CreatedAt        string  `json:"created_at"`
}

type BlindboxRecordList struct {
	Items []BlindboxRecord `json:"items"`
	Total int64            `json:"total"`
}

type BlindBoxService struct {
	entClient       *dbent.Client
	db              *sql.DB
	settingSvc      *SettingService
	userRepo        UserRepository
	billingCache    *BillingCacheService
	subscriptionSvc *SubscriptionService
	redeemCodeRepo  RedeemCodeRepository
}

func NewBlindBoxService(
	entClient *dbent.Client,
	db *sql.DB,
	settingSvc *SettingService,
	userRepo UserRepository,
	billingCache *BillingCacheService,
	subscriptionSvc *SubscriptionService,
	redeemCodeRepo RedeemCodeRepository,
) *BlindBoxService {
	return &BlindBoxService{
		entClient:       entClient,
		db:              db,
		settingSvc:      settingSvc,
		userRepo:        userRepo,
		billingCache:    billingCache,
		subscriptionSvc: subscriptionSvc,
		redeemCodeRepo:  redeemCodeRepo,
	}
}

func (s *BlindBoxService) ListPrizeItems(ctx context.Context) ([]PrizeItem, error) {
	items, err := s.entClient.CheckinPrizeItem.Query().
		Where(checkinprizeitem.DeletedAtIsNil()).
		Order(dbent.Desc(checkinprizeitem.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query prize items: %w", err)
	}

	result := make([]PrizeItem, 0, len(items))
	for _, item := range items {
		result = append(result, prizeItemFromEnt(item))
	}
	return result, nil
}

type CreatePrizeItemRequest struct {
	Name             string  `json:"name" binding:"required"`
	Rarity           string  `json:"rarity" binding:"required"`
	RewardType       string  `json:"reward_type" binding:"required"`
	RewardValue      float64 `json:"reward_value"`
	RewardValueMax   float64 `json:"reward_value_max"`
	SubscriptionID   *int64  `json:"subscription_id"`
	SubscriptionDays int     `json:"subscription_days"`
	Weight           int     `json:"weight"`
	IsEnabled        *bool   `json:"is_enabled"`
}

func (s *BlindBoxService) CreatePrizeItem(ctx context.Context, req CreatePrizeItemRequest) (*PrizeItem, error) {
	if req.Weight <= 0 {
		req.Weight = 100
	}
	if err := validatePrizeItem(req.Rarity, req.RewardType, req.RewardValue, req.RewardValueMax, req.SubscriptionID, req.SubscriptionDays, req.Weight); err != nil {
		return nil, err
	}
	builder := s.entClient.CheckinPrizeItem.Create().
		SetName(req.Name).
		SetRarity(req.Rarity).
		SetRewardType(req.RewardType).
		SetRewardValue(req.RewardValue).
		SetRewardValueMax(req.RewardValueMax).
		SetSubscriptionDays(req.SubscriptionDays).
		SetWeight(req.Weight)

	if req.SubscriptionID != nil {
		builder.SetSubscriptionID(*req.SubscriptionID)
	}
	if req.IsEnabled != nil {
		builder.SetIsEnabled(*req.IsEnabled)
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create prize item: %w", err)
	}

	result := prizeItemFromEnt(item)
	return &result, nil
}

type UpdatePrizeItemRequest struct {
	Name             *string  `json:"name"`
	Rarity           *string  `json:"rarity"`
	RewardType       *string  `json:"reward_type"`
	RewardValue      *float64 `json:"reward_value"`
	RewardValueMax   *float64 `json:"reward_value_max"`
	SubscriptionID   **int64  `json:"subscription_id"`
	SubscriptionDays *int     `json:"subscription_days"`
	Weight           *int     `json:"weight"`
	IsEnabled        *bool    `json:"is_enabled"`
}

func (s *BlindBoxService) UpdatePrizeItem(ctx context.Context, id int64, req UpdatePrizeItemRequest) (*PrizeItem, error) {
	existing, err := s.entClient.CheckinPrizeItem.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get prize item: %w", err)
	}
	rarity, rewardType := existing.Rarity, existing.RewardType
	rewardValue, rewardValueMax := existing.RewardValue, existing.RewardValueMax
	subscriptionID, subscriptionDays, weight := existing.SubscriptionID, existing.SubscriptionDays, existing.Weight
	if req.Rarity != nil {
		rarity = *req.Rarity
	}
	if req.RewardType != nil {
		rewardType = *req.RewardType
	}
	if req.RewardValue != nil {
		rewardValue = *req.RewardValue
	}
	if req.RewardValueMax != nil {
		rewardValueMax = *req.RewardValueMax
	}
	if req.SubscriptionID != nil {
		subscriptionID = *req.SubscriptionID
	}
	if req.SubscriptionDays != nil {
		subscriptionDays = *req.SubscriptionDays
	}
	if req.Weight != nil {
		weight = *req.Weight
	}
	if err := validatePrizeItem(rarity, rewardType, rewardValue, rewardValueMax, subscriptionID, subscriptionDays, weight); err != nil {
		return nil, err
	}
	builder := s.entClient.CheckinPrizeItem.UpdateOneID(id)
	if req.Name != nil {
		builder.SetName(*req.Name)
	}
	if req.Rarity != nil {
		builder.SetRarity(*req.Rarity)
	}
	if req.RewardType != nil {
		builder.SetRewardType(*req.RewardType)
	}
	if req.RewardValue != nil {
		builder.SetRewardValue(*req.RewardValue)
	}
	if req.RewardValueMax != nil {
		builder.SetRewardValueMax(*req.RewardValueMax)
	}
	if req.SubscriptionID != nil {
		builder.SetNillableSubscriptionID(*req.SubscriptionID)
	}
	if req.SubscriptionDays != nil {
		builder.SetSubscriptionDays(*req.SubscriptionDays)
	}
	if req.Weight != nil {
		builder.SetWeight(*req.Weight)
	}
	if req.IsEnabled != nil {
		builder.SetIsEnabled(*req.IsEnabled)
	}

	item, err := builder.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update prize item: %w", err)
	}

	result := prizeItemFromEnt(item)
	return &result, nil
}

func validatePrizeItem(rarity, rewardType string, rewardValue, rewardValueMax float64, subscriptionID *int64, subscriptionDays, weight int) error {
	validRarity := rarity == RarityCommon || rarity == RarityRare || rarity == RarityEpic || rarity == RarityLegendary
	validReward := rewardType == BlindboxRewardBalance || rewardType == BlindboxRewardConcurrency || rewardType == BlindboxRewardSubscription || rewardType == BlindboxRewardInvitationCode
	if !validRarity || !validReward || weight <= 0 || math.IsNaN(rewardValue) || math.IsInf(rewardValue, 0) || rewardValue < 0 || math.IsNaN(rewardValueMax) || math.IsInf(rewardValueMax, 0) || rewardValueMax < 0 || (rewardValueMax > 0 && rewardValueMax < rewardValue) {
		return ErrBlindboxPrizeInvalid
	}
	if rewardType == BlindboxRewardSubscription && (subscriptionID == nil || *subscriptionID <= 0 || subscriptionDays <= 0) {
		return ErrBlindboxPrizeInvalid
	}
	if rewardType == BlindboxRewardConcurrency && rewardValue != math.Trunc(rewardValue) {
		return ErrBlindboxPrizeInvalid
	}
	return nil
}

func (s *BlindBoxService) DeletePrizeItem(ctx context.Context, id int64) error {
	return s.entClient.CheckinPrizeItem.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(ctx)
}

func (s *BlindBoxService) ShouldTriggerBlindbox(ctx context.Context, userID int64, streakDays int) bool {
	if (!s.settingSvc.IsCheckinEnabled(ctx) && !s.settingSvc.IsCheckinLuckEnabled(ctx)) || !s.settingSvc.IsCheckinBlindboxEnabled(ctx) {
		return false
	}

	triggerType := s.settingSvc.GetCheckinBlindboxTriggerType(ctx)
	interval := s.settingSvc.GetCheckinBlindboxInterval(ctx)
	if interval <= 0 {
		return false
	}

	if triggerType == "total" {
		var totalCheckins int
		if tx := dbent.TxFromContext(ctx); tx != nil {
			rows, err := tx.Client().QueryContext(ctx, `SELECT COUNT(*) FROM checkins WHERE user_id = $1`, userID)
			if err == nil {
				defer rows.Close()
				if rows.Next() {
					err = rows.Scan(&totalCheckins)
				}
			}
			if err != nil {
				logger.LegacyPrintf("service.blindbox", "failed to count total checkins for user %d: %v", userID, err)
				return false
			}
		} else if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM checkins WHERE user_id = $1`, userID).Scan(&totalCheckins); err != nil {
			logger.LegacyPrintf("service.blindbox", "failed to count total checkins for user %d: %v", userID, err)
			return false
		}
		return totalCheckins > 0 && totalCheckins%interval == 0
	}

	return streakDays > 0 && streakDays%interval == 0
}

func (s *BlindBoxService) Draw(ctx context.Context, userID int64, streakDays int) (*BlindboxResult, error) {
	client := s.entClient
	if tx := dbent.TxFromContext(ctx); tx != nil {
		client = tx.Client()
	}
	items, err := client.CheckinPrizeItem.Query().
		Where(
			checkinprizeitem.IsEnabled(true),
			checkinprizeitem.DeletedAtIsNil(),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query prize items: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	totalWeight := 0
	for _, item := range items {
		totalWeight += item.Weight
	}

	roll, err := secureRandomIntN(totalWeight)
	if err != nil {
		return nil, fmt.Errorf("select blindbox prize: %w", err)
	}
	cumWeight := 0
	var selected *dbent.CheckinPrizeItem
	for _, item := range items {
		cumWeight += item.Weight
		if roll < cumWeight {
			selected = item
			break
		}
	}
	if selected == nil {
		selected = items[0]
	}

	rewardValue := selected.RewardValue
	if selected.RewardType == BlindboxRewardBalance && selected.RewardValueMax > selected.RewardValue {
		randomValue, err := secureRandomFloat64()
		if err != nil {
			return nil, fmt.Errorf("select blindbox reward value: %w", err)
		}
		rewardValue = selected.RewardValue + randomValue*(selected.RewardValueMax-selected.RewardValue)
		rewardValue = math.Round(rewardValue*100) / 100
	}

	var rewardDetail string
	applyAndRecord := func(txCtx context.Context, txClient *dbent.Client) error {
		var applyErr error
		rewardDetail, applyErr = s.applyReward(txCtx, txClient, userID, selected, rewardValue)
		if applyErr != nil {
			return fmt.Errorf("apply reward: %w", applyErr)
		}

		_, saveErr := txClient.CheckinBlindboxRecord.Create().
			SetUserID(userID).
			SetPrizeItemID(selected.ID).
			SetPrizeName(selected.Name).
			SetRarity(selected.Rarity).
			SetRewardType(selected.RewardType).
			SetRewardValue(rewardValue).
			SetRewardDetail(rewardDetail).
			SetStreakDays(streakDays).
			Save(txCtx)
		if saveErr != nil {
			return fmt.Errorf("save blindbox record: %w", saveErr)
		}
		return nil
	}

	if tx := dbent.TxFromContext(ctx); tx != nil {
		if err := applyAndRecord(ctx, tx.Client()); err != nil {
			return nil, err
		}
	} else {
		tx, err := s.entClient.Tx(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin blindbox transaction: %w", err)
		}
		txCtx := dbent.NewTxContext(ctx, tx)
		if err := applyAndRecord(txCtx, tx.Client()); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit blindbox transaction: %w", err)
		}
	}

	return &BlindboxResult{
		PrizeName:        selected.Name,
		Rarity:           selected.Rarity,
		RewardType:       selected.RewardType,
		RewardValue:      rewardValue,
		SubscriptionDays: selected.SubscriptionDays,
		RewardDetail:     rewardDetail,
	}, nil
}

func (s *BlindBoxService) applyReward(ctx context.Context, client *dbent.Client, userID int64, item *dbent.CheckinPrizeItem, value float64) (string, error) {
	switch item.RewardType {
	case BlindboxRewardBalance:
		if value > 0 {
			if err := updateBalanceWithoutRecharge(ctx, s.userRepo, userID, value); err != nil {
				return "", fmt.Errorf("update balance: %w", err)
			}
			if err := s.createAuditRecord(ctx, userID, value, item); err != nil {
				return "", err
			}
		}
	case BlindboxRewardConcurrency:
		_, err := client.User.UpdateOneID(userID).
			AddConcurrency(int(value)).
			Save(ctx)
		if err != nil {
			return "", fmt.Errorf("update concurrency: %w", err)
		}
		if err := s.createAuditRecord(ctx, userID, value, item); err != nil {
			return "", err
		}
	case BlindboxRewardSubscription:
		if item.SubscriptionID != nil && item.SubscriptionDays > 0 {
			_, _, err := s.subscriptionSvc.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
				UserID:       userID,
				GroupID:      *item.SubscriptionID,
				ValidityDays: item.SubscriptionDays,
				Notes:        "check-in blind box reward",
			})
			if err != nil {
				return "", fmt.Errorf("assign subscription: %w", err)
			}
			if err := s.createAuditRecord(ctx, userID, float64(item.SubscriptionDays), item); err != nil {
				return "", err
			}
		}
	case BlindboxRewardInvitationCode:
		format := DefaultCompactRedeemCodeFormat()
		if s.settingSvc != nil {
			format = s.settingSvc.GetCodeFormatSettings(ctx).Invitation
		}
		code, err := format.Generate()
		if err != nil {
			return "", fmt.Errorf("generate invitation code: %w", err)
		}
		redeemCode := &RedeemCode{
			Code:   code,
			Type:   RedeemTypeInvitation,
			Value:  0,
			Status: StatusUnused,
		}
		if createErr := s.redeemCodeRepo.Create(ctx, redeemCode); createErr != nil {
			return "", fmt.Errorf("create invitation redeem code: %w", createErr)
		}
		if err := s.createAuditRecord(ctx, userID, 0, item); err != nil {
			return "", err
		}
		if s.billingCache != nil {
			go func() {
				cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = s.billingCache.InvalidateUserBalance(cacheCtx, userID)
			}()
		}
		return code, nil
	}
	if s.billingCache != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCache.InvalidateUserBalance(cacheCtx, userID)
		}()
	}
	return "", nil
}

func (s *BlindBoxService) createAuditRecord(ctx context.Context, userID int64, value float64, item *dbent.CheckinPrizeItem) error {
	code, err := GenerateRedeemCode()
	if err != nil {
		return err
	}
	now := time.Now()
	record := &RedeemCode{
		Code:   code,
		Type:   AdjustmentTypeCheckinBlindbox,
		Value:  value,
		Status: StatusUsed,
		UsedBy: &userID,
		UsedAt: &now,
		Notes:  fmt.Sprintf("%s · %s · %s", item.Name, readableRarity(item.Rarity), readableRewardType(item.RewardType)),
	}
	if item.SubscriptionID != nil {
		record.GroupID = item.SubscriptionID
		record.ValidityDays = item.SubscriptionDays
	}
	return s.redeemCodeRepo.Create(ctx, record)
}

func readableRarity(rarity string) string {
	switch rarity {
	case RarityCommon:
		return "Common"
	case RarityRare:
		return "Rare"
	case RarityEpic:
		return "Epic"
	case RarityLegendary:
		return "Legendary"
	default:
		return rarity
	}
}

func readableRewardType(rewardType string) string {
	switch rewardType {
	case BlindboxRewardBalance:
		return "Balance"
	case BlindboxRewardConcurrency:
		return "Concurrency"
	case BlindboxRewardSubscription:
		return "Subscription"
	case BlindboxRewardInvitationCode:
		return "Invitation Code"
	default:
		return rewardType
	}
}

func (s *BlindBoxService) GetUserRecords(ctx context.Context, userID int64, page, pageSize int) (*BlindboxRecordList, error) {
	offset := (page - 1) * pageSize

	total, err := s.entClient.CheckinBlindboxRecord.Query().
		Where(checkinblindboxrecord.UserID(userID)).
		Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count records: %w", err)
	}

	records, err := s.entClient.CheckinBlindboxRecord.Query().
		Where(checkinblindboxrecord.UserID(userID)).
		Order(dbent.Desc(checkinblindboxrecord.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("query records: %w", err)
	}

	items := make([]BlindboxRecord, 0, len(records))
	for _, r := range records {
		rec := BlindboxRecord{
			ID:           r.ID,
			PrizeName:    r.PrizeName,
			Rarity:       r.Rarity,
			RewardType:   r.RewardType,
			RewardValue:  r.RewardValue,
			RewardDetail: r.RewardDetail,
			StreakDays:   r.StreakDays,
			CreatedAt:    r.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if r.RewardType == BlindboxRewardSubscription && r.PrizeItemID > 0 {
			prize, err := s.entClient.CheckinPrizeItem.Get(ctx, r.PrizeItemID)
			if err == nil {
				rec.SubscriptionDays = prize.SubscriptionDays
			}
		}
		items = append(items, rec)
	}

	return &BlindboxRecordList{Items: items, Total: int64(total)}, nil
}

func (s *BlindBoxService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	totalItems, err := s.entClient.CheckinPrizeItem.Query().
		Where(checkinprizeitem.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	enabledItems, err := s.entClient.CheckinPrizeItem.Query().
		Where(checkinprizeitem.DeletedAtIsNil(), checkinprizeitem.IsEnabled(true)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	totalDraws, err := s.entClient.CheckinBlindboxRecord.Query().Count(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_items":   totalItems,
		"enabled_items": enabledItems,
		"total_draws":   totalDraws,
	}, nil
}

func prizeItemFromEnt(item *dbent.CheckinPrizeItem) PrizeItem {
	return PrizeItem{
		ID:               item.ID,
		Name:             item.Name,
		Rarity:           item.Rarity,
		RewardType:       item.RewardType,
		RewardValue:      item.RewardValue,
		RewardValueMax:   item.RewardValueMax,
		SubscriptionID:   item.SubscriptionID,
		SubscriptionDays: item.SubscriptionDays,
		Weight:           item.Weight,
		IsEnabled:        item.IsEnabled,
		CreatedAt:        item.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        item.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
