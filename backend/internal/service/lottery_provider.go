package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	LotteryTypeSSQ = "ssq"

	lotteryIssueStatusPending = "pending"
	lotteryIssueStatusOpened  = "opened"
	lotteryIssueStatusSettled = "settled"

	lotteryOpenHour   = 21
	lotteryOpenMinute = 15
	lotteryCutoffLead = 10 * time.Minute
)

var (
	ErrLotteryTypeInvalid         = infraerrors.BadRequest("LOTTERY_TYPE_INVALID", "lottery type is invalid")
	ErrLotteryProviderNotFound    = infraerrors.NotFound("LOTTERY_PROVIDER_NOT_FOUND", "lottery provider not found")
	ErrLotteryProviderUnavailable = infraerrors.ServiceUnavailable("LOTTERY_PROVIDER_UNAVAILABLE", "lottery provider unavailable")
	ErrLotteryJackpotUnavailable  = infraerrors.InternalServer("LOTTERY_JACKPOT_UNAVAILABLE", "lottery jackpot service is unavailable")
	ErrLotteryDataInvalid         = infraerrors.InternalServer("LOTTERY_DATA_INVALID", "lottery provider returned invalid data")
	ErrLotteryIssueClosed         = infraerrors.Forbidden("LOTTERY_ISSUE_CLOSED", "lottery betting is closed for this issue")
	ErrLotteryIssueMismatch       = infraerrors.BadRequest("LOTTERY_ISSUE_MISMATCH", "lottery issue does not match current issue")
	ErrLotteryBetLimitExceeded    = infraerrors.BadRequest("LOTTERY_BET_LIMIT_EXCEEDED", "lottery bet limit exceeded for this issue")
	ErrLotteryBetCountInvalid     = infraerrors.BadRequest("LOTTERY_BET_COUNT_INVALID", "lottery MVP currently supports one bet per request")
	ErrLotteryOrderQueryInvalid   = infraerrors.BadRequest("LOTTERY_ORDER_QUERY_INVALID", "lottery order query is invalid")
	ErrLotteryOrderReplayMissing  = infraerrors.InternalServer("LOTTERY_ORDER_REPLAY_MISSING", "lottery order replay state is inconsistent")
	ErrLotteryNumbersInvalid      = infraerrors.BadRequest("LOTTERY_NUMBERS_INVALID", "lottery numbers are invalid")
)

type Issue struct {
	LotteryType string    `json:"lottery_type"`
	IssueNo     string    `json:"issue_no"`
	OpenTime    time.Time `json:"open_time"`
	CutoffTime  time.Time `json:"cutoff_time"`
	IsClosed    bool      `json:"is_closed"`
	Status      string    `json:"status"`
	Source      string    `json:"source"`
}

type Result struct {
	LotteryType   string          `json:"lottery_type"`
	IssueNo       string          `json:"issue_no"`
	RedBalls      []string        `json:"red_balls"`
	BlueBall      string          `json:"blue_ball"`
	OpenedAt      time.Time       `json:"opened_at"`
	Source        string          `json:"source"`
	SourceRef     string          `json:"source_ref"`
	SourcePayload json.RawMessage `json:"source_payload"`
}

type LotteryProvider interface {
	Name() string
	GetCurrentIssue(ctx context.Context) (*Issue, error)
	GetLatestResult(ctx context.Context) (*Result, error)
}

type lotteryTypedProvider interface {
	LotteryProvider
	LotteryType() string
}

func normalizeLotteryType(raw string) (string, error) {
	lotteryType := strings.ToLower(strings.TrimSpace(raw))
	if lotteryType == "" || len(lotteryType) > 32 {
		return "", ErrLotteryTypeInvalid
	}
	for _, r := range lotteryType {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return "", ErrLotteryTypeInvalid
	}
	return lotteryType, nil
}

func normalizeLotteryBalls(raw string, expectedCount, minValue, maxValue int) ([]string, error) {
	parts := strings.FieldsFunc(strings.TrimSpace(raw), func(r rune) bool {
		return r == ',' || r == ' ' || r == ';' || r == '|'
	})
	if len(parts) != expectedCount {
		return nil, ErrLotteryDataInvalid.WithMetadata(map[string]string{
			"expected": strconv.Itoa(expectedCount),
			"actual":   strconv.Itoa(len(parts)),
		})
	}

	values := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		number, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, ErrLotteryDataInvalid.WithCause(err)
		}
		if number < minValue || number > maxValue {
			return nil, ErrLotteryDataInvalid.WithMetadata(map[string]string{
				"min":   strconv.Itoa(minValue),
				"max":   strconv.Itoa(maxValue),
				"value": strconv.Itoa(number),
			})
		}
		if _, ok := seen[number]; ok {
			return nil, ErrLotteryDataInvalid.WithMetadata(map[string]string{
				"value": strconv.Itoa(number),
			})
		}
		seen[number] = struct{}{}
		values = append(values, number)
	}

	sort.Ints(values)
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, fmt.Sprintf("%02d", value))
	}
	return normalized, nil
}

func normalizeLotteryBall(raw string, minValue, maxValue int) (string, error) {
	normalized, err := normalizeLotteryBalls(raw, 1, minValue, maxValue)
	if err != nil {
		return "", err
	}
	return normalized[0], nil
}

func lotteryDrawTimeFromDate(drawDate time.Time) time.Time {
	loc := timezone.Location()
	localDate := drawDate.In(loc)
	return time.Date(localDate.Year(), localDate.Month(), localDate.Day(), lotteryOpenHour, lotteryOpenMinute, 0, 0, loc)
}

func decorateLotteryIssue(issue *Issue, now time.Time) *Issue {
	if issue == nil {
		return nil
	}
	copied := *issue
	copied.OpenTime = copied.OpenTime.In(timezone.Location())
	copied.CutoffTime = copied.OpenTime.Add(-lotteryCutoffLead)
	copied.IsClosed = !now.Before(copied.CutoffTime)
	switch {
	case now.Before(copied.OpenTime):
		copied.Status = lotteryIssueStatusPending
	default:
		copied.Status = lotteryIssueStatusOpened
	}
	return &copied
}
