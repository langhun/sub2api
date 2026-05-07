package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TempUnschedState 临时不可调度状态
type TempUnschedState struct {
	UntilUnix       int64  `json:"until_unix"`        // 解除时间（Unix 时间戳）
	TriggeredAtUnix int64  `json:"triggered_at_unix"` // 触发时间（Unix 时间戳）
	StatusCode      int    `json:"status_code"`       // 触发的错误码
	ReasonCode      string `json:"reason_code"`       // 结构化原因代码
	MatchedKeyword  string `json:"matched_keyword"`   // 匹配的关键词
	RuleIndex       int    `json:"rule_index"`        // 触发的规则索引
	ErrorMessage    string `json:"error_message"`     // 错误消息
}

var tempUnschedStatusCodePattern = regexp.MustCompile(`\b([1-5][0-9]{2})\b`)

const (
	TempUnschedReasonCodeCustomRule                    = "custom_rule"
	TempUnschedReasonCodeOAuth401RefreshWindow         = "oauth_401_refresh_window"
	TempUnschedReasonCodeOpenAI403Cooldown             = "openai_403_cooldown"
	TempUnschedReasonCodeAntigravityInternal500Penalty = "antigravity_internal_500_penalty"
	TempUnschedReasonCodeTokenRefreshRetryExhausted    = "token_refresh_retry_exhausted"
	TempUnschedReasonCodeStreamTimeout                 = "stream_timeout"
)

// BuildTempUnschedReason serializes a temp-unsched state for storage.
// Falls back to the provided plain-text reason when serialization fails.
func BuildTempUnschedReason(state *TempUnschedState, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if state == nil {
		return fallback
	}

	state = normalizeTempUnschedState(state, 0, fallback)
	raw, err := json.Marshal(state)
	if err != nil {
		return fallback
	}
	return string(raw)
}

// ParseTempUnschedReason converts either structured JSON or legacy plain-text
// reasons into a normalized state for display and recovery logic.
func ParseTempUnschedReason(reason string, fallbackUntilUnix int64) *TempUnschedState {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil
	}

	var state TempUnschedState
	if err := json.Unmarshal([]byte(reason), &state); err == nil {
		return normalizeTempUnschedState(&state, fallbackUntilUnix, "")
	}

	state = TempUnschedState{
		UntilUnix:    fallbackUntilUnix,
		RuleIndex:    -1,
		ErrorMessage: reason,
	}

	if matches := tempUnschedStatusCodePattern.FindStringSubmatch(reason); len(matches) == 2 {
		if code, err := strconv.Atoi(matches[1]); err == nil {
			state.StatusCode = code
		}
	}

	return normalizeTempUnschedState(&state, fallbackUntilUnix, reason)
}

func normalizeTempUnschedState(state *TempUnschedState, fallbackUntilUnix int64, fallbackMessage string) *TempUnschedState {
	if state == nil {
		return nil
	}

	if state.UntilUnix == 0 {
		state.UntilUnix = fallbackUntilUnix
	}
	if state.MatchedKeyword == "" {
		state.MatchedKeyword = inferTempUnschedMatchedKeyword(state.StatusCode, state.ErrorMessage)
	}
	if state.ReasonCode == "" {
		state.ReasonCode = inferTempUnschedReasonCode(state.StatusCode, state.MatchedKeyword, state.ErrorMessage)
	}
	if strings.TrimSpace(state.ErrorMessage) == "" {
		state.ErrorMessage = strings.TrimSpace(fallbackMessage)
	}
	if strings.TrimSpace(state.ErrorMessage) == "" {
		state.ErrorMessage = buildTempUnschedFallbackMessage(state)
	}
	return state
}

func inferTempUnschedMatchedKeyword(statusCode int, message string) string {
	lower := strings.ToLower(strings.TrimSpace(message))
	switch {
	case strings.Contains(lower, "stream_timeout"), strings.Contains(lower, "stream data interval timeout"):
		return "stream_timeout"
	case strings.Contains(lower, "token refresh retry exhausted"):
		return "token_refresh"
	case strings.Contains(lower, "internal 500"):
		return "internal_500"
	case strings.Contains(lower, "oauth 401"):
		return "oauth_401"
	case statusCode == 403 && strings.Contains(lower, "temporary cooldown"):
		return "temporary_cooldown"
	default:
		return ""
	}
}

func inferTempUnschedReasonCode(statusCode int, matchedKeyword, message string) string {
	lowerKeyword := strings.ToLower(strings.TrimSpace(matchedKeyword))
	lowerMessage := strings.ToLower(strings.TrimSpace(message))

	switch {
	case lowerKeyword == "oauth_401" || strings.Contains(lowerMessage, "oauth 401"):
		return TempUnschedReasonCodeOAuth401RefreshWindow
	case lowerKeyword == "temporary_cooldown" || (statusCode == 403 && strings.Contains(lowerMessage, "temporary cooldown")):
		return TempUnschedReasonCodeOpenAI403Cooldown
	case lowerKeyword == "internal_500" || strings.Contains(lowerMessage, "internal 500"):
		return TempUnschedReasonCodeAntigravityInternal500Penalty
	case lowerKeyword == "token_refresh" || strings.Contains(lowerMessage, "token refresh retry exhausted"):
		return TempUnschedReasonCodeTokenRefreshRetryExhausted
	case lowerKeyword == "stream_timeout" || strings.Contains(lowerMessage, "stream data interval timeout"):
		return TempUnschedReasonCodeStreamTimeout
	case lowerKeyword != "":
		return TempUnschedReasonCodeCustomRule
	case statusCode > 0:
		return fmt.Sprintf("http_%d", statusCode)
	default:
		return "legacy_plain_text"
	}
}

func buildTempUnschedFallbackMessage(state *TempUnschedState) string {
	if state == nil {
		return ""
	}

	switch state.ReasonCode {
	case TempUnschedReasonCodeOAuth401RefreshWindow:
		return "OAuth 401 temporary unschedulable"
	case TempUnschedReasonCodeOpenAI403Cooldown:
		return "OpenAI 403 temporary cooldown"
	case TempUnschedReasonCodeAntigravityInternal500Penalty:
		return "Antigravity INTERNAL 500 temporary penalty"
	case TempUnschedReasonCodeTokenRefreshRetryExhausted:
		return "Token refresh retry exhausted"
	case TempUnschedReasonCodeStreamTimeout:
		return "Stream timeout temporary unschedulable"
	case TempUnschedReasonCodeCustomRule:
		if state.StatusCode > 0 && state.MatchedKeyword != "" {
			return fmt.Sprintf("Temporary unschedulable: http %d matched keyword %q", state.StatusCode, state.MatchedKeyword)
		}
	}

	if state.StatusCode > 0 {
		return fmt.Sprintf("Temporary unschedulable: http %d", state.StatusCode)
	}
	if state.MatchedKeyword != "" {
		return "Temporary unschedulable: " + state.MatchedKeyword
	}
	return ""
}

// TempUnschedCache 临时不可调度缓存接口
type TempUnschedCache interface {
	SetTempUnsched(ctx context.Context, accountID int64, state *TempUnschedState) error
	GetTempUnsched(ctx context.Context, accountID int64) (*TempUnschedState, error)
	DeleteTempUnsched(ctx context.Context, accountID int64) error
}

// TimeoutCounterCache 超时计数器缓存接口
type TimeoutCounterCache interface {
	// IncrementTimeoutCount 增加账户的超时计数，返回当前计数值
	// windowMinutes 是计数窗口时间（分钟），超过此时间计数器会自动重置
	IncrementTimeoutCount(ctx context.Context, accountID int64, windowMinutes int) (int64, error)
	// GetTimeoutCount 获取账户当前的超时计数
	GetTimeoutCount(ctx context.Context, accountID int64) (int64, error)
	// ResetTimeoutCount 重置账户的超时计数
	ResetTimeoutCount(ctx context.Context, accountID int64) error
	// GetTimeoutCountTTL 获取计数器剩余过期时间
	GetTimeoutCountTTL(ctx context.Context, accountID int64) (time.Duration, error)
}
