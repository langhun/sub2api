package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestSSQProviderGetLatestResult(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte(`{
			"state": 0,
			"message": "ok",
			"result": [{
				"code": "2026062",
				"detailsLink": "/c/2026/06/02/656270.shtml",
				"date": "2026-06-02(Tue)",
				"red": "29,02,14,07,28,04",
				"blue": "09"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewSSQProviderWithClient(server.Client(), server.URL)
	result, err := provider.GetLatestResult(context.Background())
	require.NoError(t, err)
	require.Equal(t, ssqProviderName, provider.Name())
	require.Equal(t, LotteryTypeSSQ, provider.LotteryType())
	require.Equal(t, "2026062", result.IssueNo)
	require.Equal(t, []string{"02", "04", "07", "14", "28", "29"}, result.RedBalls)
	require.Equal(t, "09", result.BlueBall)
	require.Equal(t, "https://www.cwl.gov.cn/c/2026/06/02/656270.shtml", result.SourceRef)
	require.Equal(t, LotteryTypeSSQ, result.LotteryType)
	require.Equal(t, ssqProviderName, result.Source)
	require.Equal(t, 2026, result.OpenedAt.Year())
	require.Equal(t, 6, int(result.OpenedAt.Month()))
	require.Equal(t, 2, result.OpenedAt.Day())
	require.Equal(t, lotteryOpenHour, result.OpenedAt.Hour())
	require.Equal(t, lotteryOpenMinute, result.OpenedAt.Minute())
	require.NotEmpty(t, result.SourcePayload)
}

func TestSSQProviderGetCurrentIssue(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"state": 0,
			"message": "ok",
			"result": [{
				"code": "2099001",
				"detailsLink": "/c/2099/06/09/demo.shtml",
				"date": "2099-06-09(Tue)",
				"red": "01,02,03,04,05,06",
				"blue": "07"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewSSQProviderWithClient(server.Client(), server.URL)
	issue, err := provider.GetCurrentIssue(context.Background())
	require.NoError(t, err)
	require.Equal(t, LotteryTypeSSQ, issue.LotteryType)
	require.Equal(t, "2099002", issue.IssueNo)
	require.Equal(t, lotteryIssueStatusPending, issue.Status)
	require.Equal(t, ssqProviderName, issue.Source)
	require.Equal(t, lotteryOpenHour, issue.OpenTime.Hour())
	require.Equal(t, lotteryOpenMinute, issue.OpenTime.Minute())
}

func TestSSQProviderRejectsInvalidPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"state": 0,
			"message": "ok",
			"result": [{
				"code": "2026062",
				"date": "2026-06-02(Tue)",
				"red": "02,04,07,14,28,28",
				"blue": "09"
			}]
		}`))
	}))
	defer server.Close()

	provider := NewSSQProviderWithClient(server.Client(), server.URL)
	_, err := provider.GetLatestResult(context.Background())
	require.Error(t, err)
	require.Equal(t, "LOTTERY_DATA_INVALID", infraerrors.Reason(err))
}
