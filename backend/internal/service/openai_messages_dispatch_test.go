package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/stretchr/testify/require"
)

func TestSelectedMessagesDispatchModelUsesCurrentRoutedGroup(t *testing.T) {
	t.Parallel()

	primary := &Group{
		Platform: PlatformOpenAI,
		MessagesDispatchModelConfig: OpenAIMessagesDispatchModelConfig{
			SonnetMappedModel: "gpt-primary",
		},
	}
	candidate := &Group{
		Platform: PlatformOpenAI,
		MessagesDispatchModelConfig: OpenAIMessagesDispatchModelConfig{
			SonnetMappedModel: "gpt-candidate",
		},
	}
	key := &APIKey{Group: primary}
	svc := &OpenAIGatewayService{}
	ctx := WithAPIKeyContext(
		WithOpenAIMessagesDispatch(context.Background(), true),
		key,
	)

	require.Equal(t, "gpt-primary", svc.selectedMessagesDispatchModel(ctx, "claude-sonnet-4-5"))
	key.Group = candidate
	require.Equal(t, "gpt-candidate", svc.selectedMessagesDispatchModel(ctx, "claude-sonnet-4-5"))

	// Normal Chat/Responses scheduling must not inherit Messages-only aliases.
	require.Equal(t, "claude-sonnet-4-5", svc.selectedMessagesDispatchModel(context.Background(), "claude-sonnet-4-5"))
}

func TestNormalizeOpenAIMessagesDispatchModelConfig(t *testing.T) {
	t.Parallel()

	cfg := normalizeOpenAIMessagesDispatchModelConfig(OpenAIMessagesDispatchModelConfig{
		OpusMappedModel:   " gpt-5.4-high ",
		SonnetMappedModel: "gpt-5.3-codex",
		HaikuMappedModel:  " gpt-5.4-mini-medium ",
		ExactModelMappings: map[string]string{
			" claude-sonnet-4-5-20250929 ": " gpt-5.2-high ",
			"":                             "gpt-5.4",
			"claude-opus-4-6":              " ",
		},
	})

	require.Equal(t, "gpt-5.4", cfg.OpusMappedModel)
	require.Equal(t, "gpt-5.3-codex", cfg.SonnetMappedModel)
	require.Equal(t, "gpt-5.4-mini", cfg.HaikuMappedModel)
	require.Equal(t, map[string]string{
		"claude-sonnet-4-5-20250929": "gpt-5.2",
	}, cfg.ExactModelMappings)
}

func TestNormalizeOpenAIMessagesDispatchModelConfigRejectsForeignModelFamilies(t *testing.T) {
	t.Parallel()

	cfg := normalizeOpenAIMessagesDispatchModelConfig(OpenAIMessagesDispatchModelConfig{
		OpusMappedModel:   "gemini-3.1-pro",
		SonnetMappedModel: "claude-sonnet-4-6",
		HaikuMappedModel:  "us.anthropic.claude-haiku-4-5-v1:0",
		ExactModelMappings: map[string]string{
			"claude-opus-4-6":   "gemini-3.1-pro",
			"claude-sonnet-4-6": "gpt-5.4",
		},
	})

	require.Empty(t, cfg.OpusMappedModel)
	require.Empty(t, cfg.SonnetMappedModel)
	require.Empty(t, cfg.HaikuMappedModel)
	require.Equal(t, map[string]string{"claude-sonnet-4-6": "gpt-5.4"}, cfg.ExactModelMappings)
}

func TestSmartRoutingAccountModelCompatibleRejectsForeignMappingOnlyForRoutedAdmission(t *testing.T) {
	t.Parallel()

	foreign := []string{"gemini-3.1-pro", "google/gemini-2.5-pro", "models/gemini-2.5-flash", "claude3-opus", "anthropic/claude-sonnet-4-6", "us.anthropic.claude-3"}
	for _, mapped := range foreign {
		account := &Account{Platform: PlatformOpenAI, Credentials: map[string]any{
			"model_mapping": map[string]any{"gpt-5": mapped},
		}}
		if SmartRoutingAccountModelCompatible(account, "gpt-5") {
			t.Fatalf("foreign mapping %q was accepted", mapped)
		}
	}

	custom := &Account{Platform: PlatformOpenAI, Credentials: map[string]any{
		"model_mapping": map[string]any{"gpt-5": "my-openai-compatible-model"},
	}}
	if !SmartRoutingAccountModelCompatible(custom, "gpt-5") {
		t.Fatal("unknown OpenAI-compatible mapping should remain allowed")
	}
	anthropic := &Account{Platform: PlatformAnthropic, Credentials: map[string]any{
		"model_mapping": map[string]any{"claude-sonnet-4-6": "claude-sonnet-4-6"},
	}}
	if !SmartRoutingAccountModelCompatible(anthropic, "claude-sonnet-4-6") {
		t.Fatal("helper must not reject native Anthropic mappings")
	}
}

func TestGroupResolveMessagesDispatchModelCannotDispatchToForeignPlatformModel(t *testing.T) {
	t.Parallel()

	group := &Group{
		Platform: PlatformOpenAI,
		MessagesDispatchModelConfig: OpenAIMessagesDispatchModelConfig{
			OpusMappedModel: "gemini-3.1-pro",
			ExactModelMappings: map[string]string{
				"claude-special": "claude-opus-4-6",
			},
		},
	}

	// A rejected family-specific override falls back to the built-in OpenAI
	// target. A rejected exact mapping has no safe fallback and is ignored.
	require.Equal(t, defaultOpenAIMessagesDispatchOpusMappedModel, group.ResolveMessagesDispatchModel("claude-opus-4-6"))
	require.Empty(t, group.ResolveMessagesDispatchModel("claude-special"))
}

func TestGroupResolveMessagesDispatchModel_GrokRequiresCrossClientMapping(t *testing.T) {
	original := xai.RuntimeModelMappingOptions()
	t.Cleanup(func() { xai.SetRuntimeModelMappingOptions(original) })
	group := &Group{Platform: PlatformGrok}

	xai.SetRuntimeModelMappingOptions(xai.ModelMappingOptions{})
	require.Empty(t, group.ResolveMessagesDispatchModel("claude-sonnet-4-5"))

	xai.SetRuntimeModelMappingOptions(xai.ModelMappingOptions{
		DefaultText:          "grok-build-0.1",
		EnableCrossClientMap: true,
	})
	require.Equal(t, "grok-build-0.1", group.ResolveMessagesDispatchModel("claude-sonnet-4-5"))
	require.Equal(t, "grok-build-0.1", group.ResolveMessagesDispatchModel("claude-opus-4-6"))
	require.Equal(t, "grok-build-0.1", group.ResolveMessagesDispatchModel("claude-haiku-4-5"))
	require.Empty(t, group.ResolveMessagesDispatchModel("grok"))
	require.Empty(t, group.ResolveMessagesDispatchModel("gpt-5.3-codex"))
}

func TestSanitizeGroupMessagesDispatchFields_ClearsNonOpenAIPlatform(t *testing.T) {
	t.Parallel()

	group := &Group{
		Platform:              PlatformAnthropic,
		AllowMessagesDispatch: true,
		DefaultMappedModel:    "gpt-5.6-sol",
		MessagesDispatchModelConfig: OpenAIMessagesDispatchModelConfig{
			SonnetMappedModel: "gpt-5.3-codex",
			ExactModelMappings: map[string]string{
				"claude-fable-5": "gpt-5.6-sol",
			},
		},
	}

	sanitizeGroupMessagesDispatchFields(group)

	require.False(t, group.AllowMessagesDispatch)
	require.Empty(t, group.DefaultMappedModel)
	require.Equal(t, OpenAIMessagesDispatchModelConfig{}, group.MessagesDispatchModelConfig)
}

func TestSanitizeGroupMessagesDispatchFields_PreservesCompositeDispatchToggle(t *testing.T) {
	t.Parallel()

	group := &Group{
		Platform:              PlatformComposite,
		AllowMessagesDispatch: true,
		DefaultMappedModel:    "gpt-5.6-sol",
		MessagesDispatchModelConfig: OpenAIMessagesDispatchModelConfig{
			SonnetMappedModel: "gpt-5.3-codex",
			ExactModelMappings: map[string]string{
				"claude-fable-5": "gpt-5.6-sol",
			},
		},
	}

	sanitizeGroupMessagesDispatchFields(group)

	require.True(t, group.AllowMessagesDispatch)
	require.Empty(t, group.DefaultMappedModel)
	require.Equal(t, OpenAIMessagesDispatchModelConfig{}, group.MessagesDispatchModelConfig)
}
