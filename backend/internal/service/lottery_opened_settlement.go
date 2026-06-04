package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const lotteryOpenedSettlementDefaultLimit = 20

func (s *LotteryService) settleOpenedIssues(ctx context.Context, lotteryType string, limit int) (*LotteryOpenedSettlementResult, error) {
	if strings.TrimSpace(lotteryType) == "" {
		lotteryType = LotteryTypeSSQ
	}
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = lotteryOpenedSettlementDefaultLimit
	}
	if limit > lotteryOpenedSettlementDefaultLimit {
		limit = lotteryOpenedSettlementDefaultLimit
	}
	if s == nil || s.entClient == nil {
		return nil, ErrLotteryStorageUnavailable
	}

	issues, err := listOpenedLotteryIssues(ctx, s.entClient, normalizedType, limit)
	if err != nil {
		return nil, err
	}
	result := &LotteryOpenedSettlementResult{
		LotteryType: normalizedType,
		TotalIssues: len(issues),
	}
	var joined error
	for _, issue := range issues {
		settled, settleErr := s.SettleIssue(ctx, normalizedType, issue.IssueNo)
		if settleErr != nil {
			result.FailedIssues = append(result.FailedIssues, issue.IssueNo)
			joined = errors.Join(joined, fmt.Errorf("settle lottery issue %s: %w", issue.IssueNo, settleErr))
			continue
		}
		if settled != nil {
			result.SettledIssues = append(result.SettledIssues, *settled)
		}
	}
	return result, joined
}
