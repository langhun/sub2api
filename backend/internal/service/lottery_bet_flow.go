package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/shopspring/decimal"
)

const (
	lotteryOrderStatusPending = "pending"
	lotteryMaxOrdersPerIssue  = 100
)

var (
	lotterySingleBetCost = decimal.NewFromInt(100)
	lotteryJackpotShare  = decimal.NewFromInt(70)
	lotteryBurnShare     = decimal.NewFromInt(20)
	lotteryPlatformShare = decimal.NewFromInt(10)
)

type lotteryBetPayload struct {
	userID      int64
	lotteryType string
	issueNo     string
	redBalls    []string
	blueBall    string
}

type lotteryOrderRecord struct {
	ID          int64
	LotteryType string
	IssueNo     string
	RedBalls    []string
	BlueBall    string
	Cost        decimal.Decimal
	Reward      decimal.Decimal
	PrizeLevel  string
	RedHits     int
	BlueHit     bool
	Status      string
	CreatedAt   sql.NullTime
}

func (s *LotteryService) createBet(ctx context.Context, input LotteryBetInput) (*LotteryBetResult, error) {
	payload, err := normalizeLotteryBetPayload(input)
	if err != nil {
		return nil, err
	}
	if s == nil || s.entClient == nil || s.jackpotService == nil {
		return nil, ErrLotteryJackpotUnavailable
	}
	currentIssue, err := s.GetCurrentIssue(ctx, payload.lotteryType)
	if err != nil {
		return nil, err
	}
	if payload.issueNo == "" {
		payload.issueNo = currentIssue.IssueNo
	}
	currentIssue = decorateLotteryIssue(currentIssue, timezone.Now())

	result, err := runSerializableLotteryTx(ctx, s.entClient, func(txClient *dbent.Client) (*LotteryBetResult, error) {
		if err := lockLotteryIssueScope(ctx, txClient, payload.lotteryType, payload.issueNo, payload.userID); err != nil {
			return nil, err
		}

		existingOrder, err := findLotteryOrderByNumbers(ctx, txClient, payload)
		if err != nil {
			return nil, err
		}
		if existingOrder != nil {
			return buildLotteryReplayResult(existingOrder), nil
		}

		if payload.issueNo != currentIssue.IssueNo {
			return nil, ErrLotteryIssueMismatch.WithMetadata(map[string]string{
				"requested_issue": payload.issueNo,
				"current_issue":   currentIssue.IssueNo,
			})
		}
		if currentIssue.IsClosed {
			return nil, ErrLotteryIssueClosed.WithMetadata(map[string]string{
				"issue_no": payload.issueNo,
			})
		}
		if err := ensureLotteryIssueInTx(ctx, txClient, currentIssue); err != nil {
			return nil, err
		}

		orderCount, err := countLotteryOrdersByUserIssue(ctx, txClient, payload.userID, payload.lotteryType, payload.issueNo)
		if err != nil {
			return nil, err
		}
		if orderCount >= lotteryMaxOrdersPerIssue {
			return nil, ErrLotteryBetLimitExceeded.WithMetadata(map[string]string{
				"issue_no": payload.issueNo,
				"limit":    fmt.Sprintf("%d", lotteryMaxOrdersPerIssue),
			})
		}

		bankRequest := buildLotteryBetBankRequest(payload)
		bankRequest.RequestID = strings.TrimSpace(input.RequestID)
		bankResult, err := NewBankService(s.entClient).ApplyTransferInTx(ctx, txClient, bankRequest)
		if err != nil {
			return nil, err
		}
		if bankResult.Replayed {
			replayedOrder, replayErr := findLotteryOrderByNumbers(ctx, txClient, payload)
			if replayErr != nil {
				return nil, replayErr
			}
			if replayedOrder == nil {
				return nil, ErrLotteryOrderReplayMissing
			}
			return buildLotteryReplayResult(replayedOrder), nil
		}

		orderID, err := createLotteryOrderInTx(ctx, txClient, payload)
		if err != nil {
			return nil, err
		}
		if err := s.jackpotService.depositInTx(ctx, txClient, payload.lotteryType, lotteryJackpotShare); err != nil {
			return nil, err
		}

		return &LotteryBetResult{
			LotteryType: payload.lotteryType,
			IssueNo:     payload.issueNo,
			OrderIDs:    []int64{orderID},
			Cost:        lotterySingleBetCost,
			Status:      lotteryOrderStatusPending,
			Replayed:    false,
		}, nil
	})
	if err != nil {
		return nil, err
	}
	if result != nil && !result.Replayed {
		s.invalidateBalanceCaches(ctx, payload.userID)
	}
	return result, nil
}

func normalizeLotteryBetPayload(input LotteryBetInput) (lotteryBetPayload, error) {
	lotteryType := strings.TrimSpace(input.LotteryType)
	if lotteryType == "" {
		lotteryType = LotteryTypeSSQ
	}
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return lotteryBetPayload{}, err
	}
	if input.UserID <= 0 {
		return lotteryBetPayload{}, ErrBankInvalidUser
	}
	betCount := input.BetCount
	if betCount == 0 {
		betCount = 1
	}
	if betCount != 1 {
		return lotteryBetPayload{}, ErrLotteryBetCountInvalid
	}
	redBalls, err := normalizeLotteryBetRedBalls(input.RedBalls)
	if err != nil {
		return lotteryBetPayload{}, err
	}
	blueBall, err := normalizeLotteryBetBlueBall(input.BlueBall)
	if err != nil {
		return lotteryBetPayload{}, err
	}
	return lotteryBetPayload{
		userID:      input.UserID,
		lotteryType: normalizedType,
		issueNo:     strings.TrimSpace(input.IssueNo),
		redBalls:    redBalls,
		blueBall:    blueBall,
	}, nil
}

func normalizeLotteryBetRedBalls(values []string) ([]string, error) {
	joined := strings.Join(values, ",")
	balls, err := normalizeLotteryBalls(joined, 6, 1, 33)
	if err != nil {
		return nil, mapLotteryNumbersError(err)
	}
	return balls, nil
}

func buildLotteryBetBankRequest(payload lotteryBetPayload) TransferFundsRequest {
	betKey := lotteryBetKey(payload)
	return TransferFundsRequest{
		UserID:           payload.userID,
		Amount:           lotterySingleBetCost,
		Type:             BankTxTypeLotteryBet,
		BusinessModule:   BankBusinessModuleGame,
		Description:      "lottery bet",
		IdempotencyScope: fmt.Sprintf("lottery:bet:user:%d:issue:%s", payload.userID, payload.issueNo),
		IdempotencyKey:   betKey,
		ReferenceType:    "lottery_order",
		ReferenceID:      betKey,
		Metadata: map[string]any{
			"lottery_type":    payload.lotteryType,
			"issue_no":        payload.issueNo,
			"red_balls":       strings.Join(payload.redBalls, ","),
			"blue_ball":       payload.blueBall,
			"jackpot_amount":  lotteryJackpotShare.String(),
			"burn_amount":     lotteryBurnShare.String(),
			"platform_amount": lotteryPlatformShare.String(),
		},
	}
}

func lotteryBetKey(payload lotteryBetPayload) string {
	return fmt.Sprintf("%s:%s:%d:%s:%s", payload.lotteryType, payload.issueNo, payload.userID, strings.Join(payload.redBalls, ","), payload.blueBall)
}

func buildLotteryReplayResult(record *lotteryOrderRecord) *LotteryBetResult {
	return &LotteryBetResult{
		LotteryType: record.LotteryType,
		IssueNo:     record.IssueNo,
		OrderIDs:    []int64{record.ID},
		Cost:        record.Cost,
		Status:      record.Status,
		Replayed:    true,
	}
}

func (s *LotteryService) invalidateBalanceCaches(ctx context.Context, userID int64) {
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateUserBalance(ctx, userID)
	}
}

func (s *LotteryService) getMyOrders(ctx context.Context, query LotteryOrderQuery) ([]LotteryOrderView, error) {
	if query.UserID <= 0 {
		return nil, ErrLotteryOrderQueryInvalid
	}
	if s == nil || s.entClient == nil {
		return nil, ErrLotteryJackpotUnavailable
	}
	if strings.TrimSpace(query.LotteryType) == "" {
		query.LotteryType = LotteryTypeSSQ
	}
	normalizedType, err := normalizeLotteryType(query.LotteryType)
	if err != nil {
		return nil, err
	}
	query.LotteryType = normalizedType
	query.IssueNo = strings.TrimSpace(query.IssueNo)
	return listLotteryOrders(ctx, s.entClient, query)
}

func normalizeLotteryBetBlueBall(raw string) (string, error) {
	ball, err := normalizeLotteryBall(raw, 1, 16)
	if err != nil {
		return "", mapLotteryNumbersError(err)
	}
	return ball, nil
}

func mapLotteryNumbersError(err error) error {
	appErr := infraerrors.FromError(err)
	if appErr == nil {
		return ErrLotteryNumbersInvalid.WithCause(err)
	}
	withMeta := ErrLotteryNumbersInvalid
	if len(appErr.Metadata) > 0 {
		withMeta = withMeta.WithMetadata(appErr.Metadata)
	}
	return withMeta.WithCause(err)
}
