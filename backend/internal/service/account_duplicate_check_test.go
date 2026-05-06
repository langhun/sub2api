package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildAccountDuplicateCheckResultStrongIdentifier(t *testing.T) {
	accounts := []duplicateCheckAccount{
		{
			ID: 1, Name: "openai-1", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Credentials: map[string]any{"chatgpt_account_id": "acct_same"},
		},
		{
			ID: 2, Name: "openai-2", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Status: StatusActive, Credentials: map[string]any{"chatgpt_account_id": "acct_same"},
		},
		{
			ID: 3, Name: "openai-3", Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Status: StatusActive, Credentials: map[string]any{"chatgpt_account_id": "acct_same"},
		},
	}

	result := buildAccountDuplicateCheckResult(accounts)

	require.Equal(t, 3, result.TotalAccounts)
	require.Equal(t, 1, result.DuplicateGroupCount)
	require.Equal(t, 2, result.DuplicateAccountCount)
	require.Equal(t, 1, result.StrongGroupCount)
	require.Equal(t, "chatgpt_account_id", result.Groups[0].KeyType)
	require.Equal(t, AccountDuplicateSeverityStrong, result.Groups[0].Severity)
	require.Equal(t, AccountTypeOAuth, result.Groups[0].Type)
}

func TestBuildAccountDuplicateCheckResultMasksSensitiveTokens(t *testing.T) {
	rawToken := "refresh-token-very-secret-value"
	accounts := []duplicateCheckAccount{
		{
			ID: 1, Name: "a1", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Credentials: map[string]any{"refresh_token": rawToken},
		},
		{
			ID: 2, Name: "a2", Platform: PlatformOpenAI, Type: AccountTypeOAuth,
			Credentials: map[string]any{"refresh_token": rawToken},
		},
	}

	result := buildAccountDuplicateCheckResult(accounts)
	payload, err := json.Marshal(result)

	require.NoError(t, err)
	require.NotContains(t, string(payload), rawToken)
	require.NotEmpty(t, result.Groups[0].MaskedValue)
	require.NotEmpty(t, result.Groups[0].ValueHash)
}

func TestBuildAccountDuplicateCheckResultWeakEmailAndName(t *testing.T) {
	accounts := []duplicateCheckAccount{
		{ID: 1, Name: "Shared Name", Platform: PlatformGemini, Type: AccountTypeOAuth, Extra: map[string]any{"email_address": "USER@example.com"}},
		{ID: 2, Name: " shared   name ", Platform: PlatformGemini, Type: AccountTypeAPIKey, Extra: map[string]any{"email_address": "user@example.com"}},
		{ID: 3, Name: "Shared Name", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Extra: map[string]any{"email_address": "user@example.com"}},
	}

	result := buildAccountDuplicateCheckResult(accounts)

	require.Equal(t, 2, result.WeakGroupCount)
	require.Equal(t, 0, result.StrongGroupCount)
	require.Equal(t, 2, result.DuplicateGroupCount)
	for _, group := range result.Groups {
		require.Equal(t, AccountDuplicateSeverityWeak, group.Severity)
		require.Equal(t, PlatformGemini, group.Platform)
	}
}

func TestBuildAccountDuplicateCheckResultNoDuplicate(t *testing.T) {
	accounts := []duplicateCheckAccount{
		{ID: 1, Name: "a1", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"account_uuid": "u1"}},
		{ID: 2, Name: "a2", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"account_uuid": "u2"}},
	}

	result := buildAccountDuplicateCheckResult(accounts)

	require.Equal(t, 2, result.TotalAccounts)
	require.Zero(t, result.DuplicateGroupCount)
	require.Zero(t, result.DuplicateAccountCount)
	require.Empty(t, result.Groups)
}
