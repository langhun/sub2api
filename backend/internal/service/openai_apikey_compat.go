package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var openAIAPIKeyCompatResponsesFields = []string{
	"max_output_tokens",
	"max_completion_tokens",
}

// shouldPreserveOpenAIAPIKeyResponsesFields reports whether the upstream is the
// official OpenAI Responses API, where Codex-specific output token knobs should
// be passed through unchanged.
func shouldPreserveOpenAIAPIKeyResponsesFields(account *Account) bool {
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeAPIKey {
		return false
	}

	baseURL := strings.TrimSpace(account.GetOpenAIBaseURL())
	if baseURL == "" {
		return true
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(parsed.Hostname()), "api.openai.com")
}

func normalizeOpenAIAPIKeyPassthroughCompatBody(body []byte, account *Account) ([]byte, bool, error) {
	if len(body) == 0 || shouldPreserveOpenAIAPIKeyResponsesFields(account) {
		return body, false, nil
	}

	normalized := body
	changed := false

	for _, field := range openAIAPIKeyCompatResponsesFields {
		if value := gjson.GetBytes(normalized, field); !value.Exists() {
			continue
		}
		next, err := sjson.DeleteBytes(normalized, field)
		if err != nil {
			return body, false, fmt.Errorf("normalize apikey passthrough body delete %s: %w", field, err)
		}
		normalized = next
		changed = true
	}

	return normalized, changed, nil
}
