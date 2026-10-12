package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsOpenAIAccountQuotaExhaustedError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		msg    string
		body   string
		want   bool
	}{
		{name: "chinese balance message", status: http.StatusForbidden, msg: "用户额度不足, 剩余额度: ¥-0.000868", want: true},
		{name: "structured insufficient quota", status: http.StatusForbidden, body: `{"error":{"code":"insufficient_quota","message":"quota exceeded"}}`, want: true},
		{name: "plain payment required", status: http.StatusPaymentRequired, msg: "payment required: insufficient balance", want: true},
		{name: "temporary forbidden", status: http.StatusForbidden, msg: "forbidden content policy", want: false},
		{name: "temporary rate limit", status: http.StatusTooManyRequests, msg: "rate limit exceeded", want: false},
		{name: "positive remaining balance", status: http.StatusForbidden, msg: "账户余额: ¥10, 剩余额度: ¥8", want: false},
		{name: "unrelated server error", status: http.StatusServiceUnavailable, msg: "upstream unavailable", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isOpenAIAccountQuotaExhaustedError(tt.status, tt.msg, []byte(tt.body)))
		})
	}
}

func TestNewOpenAIUpstreamFailoverErrorQuotaSkipsSameAccountRetry(t *testing.T) {
	err := newOpenAIUpstreamFailoverError(
		http.StatusForbidden,
		nil,
		[]byte(`{"error":{"code":"insufficient_quota","message":"用户额度不足"}}`),
		"用户额度不足",
		true,
	)
	require.False(t, err.RetryableOnSameAccount)
	require.Equal(t, GatewayFailureScopeAccount, err.Scope)
	require.Equal(t, NextAccountRetry, err.NextAccountAction)
	require.True(t, err.ShouldClearStickyAccountBinding())
}

func TestUpstreamFailoverErrorStickyBindingDisposition(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  *UpstreamFailoverError
		want bool
	}{
		{name: "account failure", err: &UpstreamFailoverError{Scope: GatewayFailureScopeAccount, NextAccountAction: NextAccountRetry}, want: true},
		{name: "legacy retry account failure", err: &UpstreamFailoverError{NextAccountAction: NextAccountLegacyRetry}, want: true},
		{name: "request failure", err: &UpstreamFailoverError{Scope: GatewayFailureScopeRequest, NextAccountAction: NextAccountRetry}},
		{name: "provider failure", err: &UpstreamFailoverError{Scope: GatewayFailureScopeProvider, NextAccountAction: NextAccountRetry}},
		{name: "routing sentinel", err: &UpstreamFailoverError{Reason: SmartRoutingAccountUnavailableReason, NextAccountAction: NextAccountRetry}},
		{name: "do not retry", err: &UpstreamFailoverError{Scope: GatewayFailureScopeAccount, NextAccountAction: NextAccountStop}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.err.ShouldClearStickyAccountBinding())
		})
	}
}

func TestNewOpenAIAccountFailoverErrorPoolModeCapsTransientSameAccountRetry(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"pool_mode": true,
		},
	}
	err := newOpenAIUpstreamFailoverError(
		http.StatusServiceUnavailable,
		nil,
		[]byte(`{"error":{"message":"upstream unavailable"}}`),
		"upstream unavailable",
		true,
	)
	// The account-specific wrapper applies the one-retry pool-site cap.
	err = (&OpenAIGatewayService{}).newOpenAIAccountFailoverErrorWithClassificationHeaders(
		account,
		http.StatusServiceUnavailable,
		nil,
		nil,
		[]byte(`{"error":{"message":"upstream unavailable"}}`),
		"upstream unavailable",
		false,
		err.RetryableOnSameAccount,
	)
	require.Equal(t, 1, err.SameAccountRetryMax)
}
