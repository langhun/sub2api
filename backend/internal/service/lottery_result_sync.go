package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

func (s *LotteryService) syncLatestResult(ctx context.Context, lotteryType string) (*LotterySyncResult, error) {
	if strings.TrimSpace(lotteryType) == "" {
		lotteryType = LotteryTypeSSQ
	}
	provider, err := s.providerByType(lotteryType)
	if err != nil {
		return nil, err
	}
	result, err := provider.GetLatestResult(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeLotteryResult(result, lotteryType, provider.Name())
	if err != nil {
		return nil, err
	}
	if s == nil || s.entClient == nil {
		return nil, ErrLotteryStorageUnavailable
	}

	var synced LotteryResultView
	replayed := false
	if err := runSerializableLotteryOperationTx(ctx, s.entClient, func(txClient *dbent.Client) error {
		if err := lockLotteryIssueScope(ctx, txClient, normalized.LotteryType, normalized.IssueNo, 0); err != nil {
			return err
		}
		view, isReplay, err := saveLotteryResultInTx(ctx, txClient, normalized)
		if err != nil {
			return err
		}
		if err := markLotteryIssueOpenedInTx(ctx, txClient, normalized); err != nil {
			return err
		}
		synced = view
		replayed = isReplay
		return nil
	}); err != nil {
		return nil, err
	}

	return &LotterySyncResult{
		LotteryType: synced.LotteryType,
		IssueNo:     synced.IssueNo,
		Result:      synced,
		Replayed:    replayed,
	}, nil
}

func normalizeLotteryResult(result *Result, fallbackLotteryType, fallbackSource string) (*Result, error) {
	if result == nil {
		return nil, ErrLotteryDataInvalid
	}
	lotteryType := strings.TrimSpace(result.LotteryType)
	if lotteryType == "" {
		lotteryType = fallbackLotteryType
	}
	normalizedType, err := normalizeLotteryType(lotteryType)
	if err != nil {
		return nil, err
	}
	issueNo, err := normalizeLotteryIssueNo(result.IssueNo)
	if err != nil {
		return nil, err
	}
	redBalls, err := normalizeLotteryBalls(strings.Join(result.RedBalls, ","), 6, 1, 33)
	if err != nil {
		return nil, err
	}
	blueBall, err := normalizeLotteryBall(result.BlueBall, 1, 16)
	if err != nil {
		return nil, err
	}
	source := strings.TrimSpace(result.Source)
	if source == "" {
		source = strings.TrimSpace(fallbackSource)
	}
	if source == "" || len(source) > 64 {
		return nil, ErrLotteryDataInvalid
	}
	sourcePayload := result.SourcePayload
	if len(sourcePayload) == 0 || !json.Valid(sourcePayload) {
		sourcePayload = json.RawMessage(`{}`)
	}
	openedAt := result.OpenedAt
	if openedAt.IsZero() {
		return nil, ErrLotteryDataInvalid
	}
	return &Result{
		LotteryType:   normalizedType,
		IssueNo:       issueNo,
		RedBalls:      redBalls,
		BlueBall:      blueBall,
		OpenedAt:      openedAt,
		Source:        source,
		SourceRef:     strings.TrimSpace(result.SourceRef),
		SourcePayload: sourcePayload,
	}, nil
}

func normalizeLotteryIssueNo(raw string) (string, error) {
	issueNo := strings.TrimSpace(raw)
	if issueNo == "" || len(issueNo) > 32 {
		return "", ErrLotteryDataInvalid
	}
	for _, r := range issueNo {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return "", ErrLotteryDataInvalid.WithMetadata(map[string]string{"issue_no": issueNo})
	}
	return issueNo, nil
}

func lotteryResultConflictError(existing LotteryResultView, incoming *Result) error {
	return ErrLotteryResultConflict.WithMetadata(map[string]string{
		"lottery_type":    incoming.LotteryType,
		"issue_no":        incoming.IssueNo,
		"existing_red":    strings.Join(existing.RedBalls, ","),
		"existing_blue":   existing.BlueBall,
		"incoming_red":    strings.Join(incoming.RedBalls, ","),
		"incoming_blue":   incoming.BlueBall,
		"existing_id":     fmt.Sprintf("%d", existing.ID),
		"incoming_source": incoming.Source,
	})
}

func lotteryResultOpenTime(result *Result) time.Time {
	if result == nil {
		return time.Time{}
	}
	return result.OpenedAt
}
