package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/shopspring/decimal"
)

const (
	lotteryOrderStatusWin  = "win"
	lotteryOrderStatusLose = "lose"

	lotteryPrizeLevelFirst  = "first"
	lotteryPrizeLevelSecond = "second"
	lotteryPrizeLevelThird  = "third"
	lotteryPrizeLevelFourth = "fourth"
	lotteryPrizeLevelFifth  = "fifth"
	lotteryPrizeLevelSixth  = "sixth"
)

var (
	lotteryPrizeFirst  = decimal.NewFromInt(500000)
	lotteryPrizeSecond = decimal.NewFromInt(100000)
	lotteryPrizeThird  = decimal.NewFromInt(3000)
	lotteryPrizeFourth = decimal.NewFromInt(200)
	lotteryPrizeFifth  = decimal.NewFromInt(10)
	lotteryPrizeSixth  = decimal.NewFromInt(5)
)

type lotteryPrize struct {
	level   string
	reward  decimal.Decimal
	redHits int
	blueHit bool
}

func (s *LotteryService) settleIssue(ctx context.Context, lotteryType, issueNo string) (*LotterySettlementResult, error) {
	payload, err := normalizeLotterySettlementInput(lotteryType, issueNo)
	if err != nil {
		return nil, err
	}
	if s == nil || s.entClient == nil {
		return nil, ErrLotteryStorageUnavailable
	}
	if s.jackpotService == nil {
		return nil, ErrLotteryJackpotUnavailable
	}
	if s.bankApplier == nil {
		return nil, ErrBankClientUnavailable
	}

	result := &LotterySettlementResult{
		LotteryType: payload.lotteryType,
		IssueNo:     payload.issueNo,
	}
	winningUsers := make(map[int64]struct{})
	err = runSerializableLotteryOperationTx(ctx, s.entClient, func(txClient *dbent.Client) error {
		if err := lockLotteryIssueScope(ctx, txClient, payload.lotteryType, payload.issueNo, 0); err != nil {
			return err
		}
		drawResult, err := getLotteryResultByIssueInTx(ctx, txClient, payload.lotteryType, payload.issueNo)
		if err != nil {
			return err
		}
		if drawResult == nil {
			return ErrLotteryResultNotFound.WithMetadata(map[string]string{
				"lottery_type": payload.lotteryType,
				"issue_no":     payload.issueNo,
			})
		}
		status, exists, err := getLotteryIssueStatusForUpdate(ctx, txClient, payload.lotteryType, payload.issueNo)
		if err != nil {
			return err
		}
		if !exists {
			if err := markLotteryIssueOpenedInTx(ctx, txClient, lotteryResultFromView(*drawResult)); err != nil {
				return err
			}
			status = lotteryIssueStatusOpened
		}
		if status == lotteryIssueStatusSettled {
			result.Replayed = true
			return nil
		}

		orders, err := listPendingLotteryOrdersForSettlement(ctx, txClient, payload.lotteryType, payload.issueNo)
		if err != nil {
			return err
		}
		plans := buildLotterySettlementPlans(orders, *drawResult)
		if plans.rewardTotal.GreaterThan(decimal.Zero) {
			if err := s.jackpotService.withdrawInTx(ctx, txClient, payload.lotteryType, plans.rewardTotal); err != nil {
				return err
			}
		}
		for _, plan := range plans.items {
			if err := updateLotteryOrderSettlementInTx(ctx, txClient, plan.order, plan.prize); err != nil {
				return err
			}
			if plan.prize.reward.GreaterThan(decimal.Zero) {
				if _, err := s.bankApplier.ApplyTransferInTx(ctx, txClient, buildLotteryWinBankRequest(plan.order, plan.prize)); err != nil {
					return err
				}
				if err := createLotteryRewardLogInTx(ctx, txClient, plan.order, plan.prize); err != nil {
					return err
				}
				winningUsers[plan.order.UserID] = struct{}{}
			}
		}
		if err := markLotteryIssueSettledInTx(ctx, txClient, payload.lotteryType, payload.issueNo); err != nil {
			return err
		}
		result.TotalOrders = len(plans.items)
		result.WinOrders = plans.winCount
		result.LoseOrders = plans.loseCount
		result.RewardTotal = plans.rewardTotal
		return nil
	})
	if err != nil {
		return nil, err
	}
	for userID := range winningUsers {
		s.invalidateBalanceCaches(ctx, userID)
	}
	return result, nil
}

type lotterySettlementPayload struct {
	lotteryType string
	issueNo     string
}

func normalizeLotterySettlementInput(lotteryType, issueNo string) (lotterySettlementPayload, error) {
	if strings.TrimSpace(lotteryType) == "" {
		lotteryType = LotteryTypeSSQ
	}
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return lotterySettlementPayload{}, err
	}
	normalizedIssue, err := normalizeLotteryIssueNo(issueNo)
	if err != nil {
		return lotterySettlementPayload{}, ErrLotteryIssueInvalid.WithCause(err)
	}
	return lotterySettlementPayload{lotteryType: normalizedType, issueNo: normalizedIssue}, nil
}

func calculateSSQPrize(order lotterySettlementOrder, result LotteryResultView) lotteryPrize {
	redHits := countLotteryRedHits(order.RedBalls, result.RedBalls)
	blueHit := order.BlueBall == result.BlueBall
	prize := lotteryPrize{redHits: redHits, blueHit: blueHit}
	switch {
	case redHits == 6 && blueHit:
		prize.level = lotteryPrizeLevelFirst
		prize.reward = lotteryPrizeFirst
	case redHits == 6:
		prize.level = lotteryPrizeLevelSecond
		prize.reward = lotteryPrizeSecond
	case redHits == 5 && blueHit:
		prize.level = lotteryPrizeLevelThird
		prize.reward = lotteryPrizeThird
	case redHits == 5 || (redHits == 4 && blueHit):
		prize.level = lotteryPrizeLevelFourth
		prize.reward = lotteryPrizeFourth
	case redHits == 4 || (redHits == 3 && blueHit):
		prize.level = lotteryPrizeLevelFifth
		prize.reward = lotteryPrizeFifth
	case blueHit:
		prize.level = lotteryPrizeLevelSixth
		prize.reward = lotteryPrizeSixth
	default:
		prize.reward = decimal.Zero
	}
	return prize
}

func countLotteryRedHits(orderBalls, resultBalls []string) int {
	resultSet := make(map[string]struct{}, len(resultBalls))
	for _, ball := range resultBalls {
		resultSet[ball] = struct{}{}
	}
	hits := 0
	for _, ball := range orderBalls {
		if _, ok := resultSet[ball]; ok {
			hits++
		}
	}
	return hits
}

type lotterySettlementPlan struct {
	order lotterySettlementOrder
	prize lotteryPrize
}

type lotterySettlementPlans struct {
	items       []lotterySettlementPlan
	winCount    int
	loseCount   int
	rewardTotal decimal.Decimal
}

func buildLotterySettlementPlans(orders []lotterySettlementOrder, result LotteryResultView) lotterySettlementPlans {
	plans := lotterySettlementPlans{
		items: make([]lotterySettlementPlan, 0, len(orders)),
	}
	for _, order := range orders {
		prize := calculateSSQPrize(order, result)
		if prize.reward.GreaterThan(decimal.Zero) {
			plans.winCount++
			plans.rewardTotal = plans.rewardTotal.Add(prize.reward)
		} else {
			plans.loseCount++
		}
		plans.items = append(plans.items, lotterySettlementPlan{order: order, prize: prize})
	}
	return plans
}

func buildLotteryWinBankRequest(order lotterySettlementOrder, prize lotteryPrize) TransferFundsRequest {
	orderID := strconv.FormatInt(order.ID, 10)
	key := fmt.Sprintf("lottery:win:%s:%d", order.IssueNo, order.ID)
	return TransferFundsRequest{
		UserID:           order.UserID,
		Amount:           prize.reward,
		Type:             BankTxTypeLotteryWin,
		BusinessModule:   BankBusinessModuleGame,
		Description:      "lottery win",
		IdempotencyScope: fmt.Sprintf("lottery:win:user:%d:issue:%s", order.UserID, order.IssueNo),
		IdempotencyKey:   key,
		ReferenceType:    "lottery_order",
		ReferenceID:      orderID,
		Metadata: map[string]any{
			"lottery_type": order.LotteryType,
			"issue_no":     order.IssueNo,
			"order_id":     orderID,
			"prize_level":  prize.level,
			"red_hits":     prize.redHits,
			"blue_hit":     prize.blueHit,
		},
	}
}

func lotteryResultFromView(view LotteryResultView) *Result {
	return &Result{
		LotteryType: view.LotteryType,
		IssueNo:     view.IssueNo,
		RedBalls:    append([]string(nil), view.RedBalls...),
		BlueBall:    view.BlueBall,
		OpenedAt:    view.OpenedAt,
		Source:      view.Source,
		SourceRef:   view.SourceRef,
	}
}

func scanLotteryIssueStatus(scanner interface{ Scan(dest ...any) error }) (string, error) {
	var status string
	if err := scanner.Scan(&status); err != nil {
		return "", err
	}
	return strings.TrimSpace(status), nil
}
