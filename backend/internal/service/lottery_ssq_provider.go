package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httpclient"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const (
	ssqProviderName    = "fucai"
	ssqProviderTimeout = 10 * time.Second
	ssqDrawNoticeURL   = "https://www.cwl.gov.cn/cwl_admin/front/cwlkj/search/kjxx/findDrawNotice?name=ssq&issueCount=1&pageNo=1&pageSize=1&systemType=PC"
	ssqDateLayout      = "2006-01-02"
)

type lotteryHTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type ssqProviderNoticeResponse struct {
	State   int                       `json:"state"`
	Message string                    `json:"message"`
	Result  []ssqProviderNoticeResult `json:"result"`
}

type ssqProviderNoticeResult struct {
	Code        string `json:"code"`
	Red         string `json:"red"`
	Blue        string `json:"blue"`
	Date        string `json:"date"`
	DetailsLink string `json:"detailsLink"`
}

type SSQProvider struct {
	httpClient lotteryHTTPDoer
	endpoint   string
}

func NewSSQProvider() (*SSQProvider, error) {
	client, err := httpclient.GetClient(httpclient.Options{
		Timeout:               ssqProviderTimeout,
		ResponseHeaderTimeout: ssqProviderTimeout,
	})
	if err != nil {
		return nil, ErrLotteryProviderUnavailable.WithCause(err)
	}
	return NewSSQProviderWithClient(client, ssqDrawNoticeURL), nil
}

func NewSSQProviderWithClient(httpClient lotteryHTTPDoer, endpoint string) *SSQProvider {
	if strings.TrimSpace(endpoint) == "" {
		endpoint = ssqDrawNoticeURL
	}
	return &SSQProvider{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
}

func (p *SSQProvider) Name() string {
	return ssqProviderName
}

func (p *SSQProvider) LotteryType() string {
	return LotteryTypeSSQ
}

func (p *SSQProvider) GetCurrentIssue(ctx context.Context) (*Issue, error) {
	notice, _, err := p.fetchLatestNotice(ctx)
	if err != nil {
		return nil, err
	}
	openTime, err := parseSSQOpenTime(notice.Date)
	if err != nil {
		return nil, err
	}
	return &Issue{
		LotteryType: LotteryTypeSSQ,
		IssueNo:     strings.TrimSpace(notice.Code),
		OpenTime:    openTime,
		Status:      ssqIssueStatus(openTime, timezone.Now()),
		Source:      p.Name(),
	}, nil
}

func (p *SSQProvider) GetLatestResult(ctx context.Context) (*Result, error) {
	notice, payload, err := p.fetchLatestNotice(ctx)
	if err != nil {
		return nil, err
	}
	openTime, err := parseSSQOpenTime(notice.Date)
	if err != nil {
		return nil, err
	}
	redBalls, err := normalizeLotteryBalls(notice.Red, 6, 1, 33)
	if err != nil {
		return nil, err
	}
	blueBall, err := normalizeLotteryBall(notice.Blue, 1, 16)
	if err != nil {
		return nil, err
	}
	return &Result{
		LotteryType:   LotteryTypeSSQ,
		IssueNo:       strings.TrimSpace(notice.Code),
		RedBalls:      redBalls,
		BlueBall:      blueBall,
		OpenedAt:      openTime,
		Source:        p.Name(),
		SourceRef:     buildSSQSourceRef(notice.DetailsLink),
		SourcePayload: payload,
	}, nil
}

func (p *SSQProvider) fetchLatestNotice(ctx context.Context) (*ssqProviderNoticeResult, json.RawMessage, error) {
	if p == nil || p.httpClient == nil {
		return nil, nil, ErrLotteryProviderUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint, nil)
	if err != nil {
		return nil, nil, ErrLotteryProviderUnavailable.WithCause(err)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, nil, ErrLotteryProviderUnavailable.WithCause(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, ErrLotteryProviderUnavailable.WithMetadata(map[string]string{
			"status_code": fmt.Sprintf("%d", resp.StatusCode),
		})
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, nil, ErrLotteryProviderUnavailable.WithCause(err)
	}

	var payload ssqProviderNoticeResponse
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil, ErrLotteryProviderUnavailable.WithCause(err)
	}
	if payload.State != 0 || len(payload.Result) == 0 {
		return nil, nil, ErrLotteryDataInvalid.WithMetadata(map[string]string{
			"state":   fmt.Sprintf("%d", payload.State),
			"message": payload.Message,
		})
	}

	notice := payload.Result[0]
	if strings.TrimSpace(notice.Code) == "" {
		return nil, nil, ErrLotteryDataInvalid
	}
	return &notice, raw, nil
}

func parseSSQOpenTime(raw string) (time.Time, error) {
	datePart := strings.TrimSpace(raw)
	if len(datePart) >= len(ssqDateLayout) {
		datePart = datePart[:len(ssqDateLayout)]
	}
	parsed, err := time.ParseInLocation(ssqDateLayout, datePart, timezone.Location())
	if err != nil {
		return time.Time{}, ErrLotteryDataInvalid.WithCause(err)
	}
	return lotteryDrawTimeFromDate(parsed), nil
}

func buildSSQSourceRef(detailsLink string) string {
	detailsLink = strings.TrimSpace(detailsLink)
	if detailsLink == "" {
		return ""
	}
	if strings.HasPrefix(detailsLink, "http://") || strings.HasPrefix(detailsLink, "https://") {
		return detailsLink
	}
	return "https://www.cwl.gov.cn" + detailsLink
}

func ssqIssueStatus(openTime, now time.Time) string {
	if now.Before(openTime) {
		return lotteryIssueStatusPending
	}
	return lotteryIssueStatusOpened
}
