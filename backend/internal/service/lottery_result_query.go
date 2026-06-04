package service

import (
	"context"
	"strings"
)

const (
	lotteryResultsDefaultLimit = 100
	lotteryResultsMaxLimit     = 100
)

func (s *LotteryService) getResults(ctx context.Context, query LotteryResultQuery) ([]LotteryResultView, error) {
	if s == nil || s.entClient == nil {
		return nil, ErrLotteryStorageUnavailable
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
	if query.IssueNo != "" {
		if _, err := normalizeLotteryIssueNo(query.IssueNo); err != nil {
			return nil, ErrLotteryIssueInvalid.WithCause(err)
		}
	}
	if query.Limit <= 0 {
		query.Limit = lotteryResultsDefaultLimit
	}
	if query.Limit > lotteryResultsMaxLimit {
		query.Limit = lotteryResultsMaxLimit
	}
	return listLotteryResults(ctx, s.entClient, query)
}

func (s *LotteryService) getResult(ctx context.Context, lotteryType, issueNo string) (*LotteryResultView, error) {
	results, err := s.getResults(ctx, LotteryResultQuery{
		LotteryType: lotteryType,
		IssueNo:     issueNo,
		Limit:       1,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		normalizedType := strings.TrimSpace(lotteryType)
		if normalizedType == "" {
			normalizedType = LotteryTypeSSQ
		}
		return nil, ErrLotteryResultNotFound.WithMetadata(map[string]string{
			"lottery_type": normalizedType,
			"issue_no":     strings.TrimSpace(issueNo),
		})
	}
	return &results[0], nil
}
