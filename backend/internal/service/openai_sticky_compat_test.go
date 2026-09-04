package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type stickyDeleteCall struct {
	groupID    int64
	sessionKey string
}

type recordingStickyDeleteCache struct {
	GatewayCache
	calls []stickyDeleteCall
}

func (c *recordingStickyDeleteCache) DeleteSessionAccountID(_ context.Context, groupID int64, sessionKey string) error {
	c.calls = append(c.calls, stickyDeleteCall{groupID: groupID, sessionKey: sessionKey})
	return nil
}

func TestGetStickySessionAccountID_FallbackToLegacyKey(t *testing.T) {
	beforeFallbackTotal, beforeFallbackHit, _ := openAIStickyCompatStats()

	cache := &stubGatewayCache{
		sessionBindings: map[string]int64{
			"openai:legacy-hash": 42,
		},
	}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashReadOldFallback: true,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	accountID, err := svc.getStickySessionAccountID(ctx, nil, "new-hash")
	require.NoError(t, err)
	require.Equal(t, int64(42), accountID)

	afterFallbackTotal, afterFallbackHit, _ := openAIStickyCompatStats()
	require.Equal(t, beforeFallbackTotal+1, afterFallbackTotal)
	require.Equal(t, beforeFallbackHit+1, afterFallbackHit)
}

func TestSetStickySessionAccountID_DualWriteOldEnabled(t *testing.T) {
	_, _, beforeDualWriteTotal := openAIStickyCompatStats()

	cache := &stubGatewayCache{sessionBindings: map[string]int64{}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashDualWriteOld: true,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	err := svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL)
	require.NoError(t, err)
	require.Equal(t, int64(9), cache.sessionBindings["openai:new-hash"])
	require.Equal(t, int64(9), cache.sessionBindings["openai:legacy-hash"])

	_, _, afterDualWriteTotal := openAIStickyCompatStats()
	require.Equal(t, beforeDualWriteTotal+1, afterDualWriteTotal)
}

func TestSetStickySessionAccountID_DualWriteOldDisabled(t *testing.T) {
	cache := &stubGatewayCache{sessionBindings: map[string]int64{}}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashDualWriteOld: false,
				},
			},
		},
	}

	ctx := withOpenAILegacySessionHash(context.Background(), "legacy-hash")
	err := svc.setStickySessionAccountID(ctx, nil, "new-hash", 9, openaiStickySessionTTL)
	require.NoError(t, err)
	require.Equal(t, int64(9), cache.sessionBindings["openai:new-hash"])
	_, exists := cache.sessionBindings["openai:legacy-hash"]
	require.False(t, exists)
}

func TestClearSmartRoutingStickySessionDeletesCurrentGroupAndLegacyHash(t *testing.T) {
	groupID := int64(21)
	cache := &recordingStickyDeleteCache{}
	svc := &OpenAIGatewayService{
		cache: cache,
		cfg: &config.Config{
			Gateway: config.GatewayConfig{
				OpenAIWS: config.GatewayOpenAIWSConfig{
					SessionHashReadOldFallback: true,
				},
			},
		},
	}
	ctx := WithSmartRoutingState(context.Background(), &SmartRoutingRequestState{SmartRoutingActive: true})
	ctx = withOpenAILegacySessionHash(ctx, "legacy-hash")

	svc.ClearSmartRoutingStickySession(ctx, &groupID, "new-hash")

	require.Equal(t, []stickyDeleteCall{
		{groupID: groupID, sessionKey: "openai:new-hash"},
		{groupID: groupID, sessionKey: "openai:legacy-hash"},
	}, cache.calls)
}

func TestClearSmartRoutingStickySessionPreservesLegacySingleGroupBinding(t *testing.T) {
	cache := &recordingStickyDeleteCache{}
	svc := &OpenAIGatewayService{cache: cache}

	svc.ClearSmartRoutingStickySession(context.Background(), nil, "session-hash")

	require.Empty(t, cache.calls)
}

func TestClearSmartRoutingStickySessionIgnoresEmptyHash(t *testing.T) {
	cache := &recordingStickyDeleteCache{}
	svc := &OpenAIGatewayService{cache: cache}
	ctx := WithSmartRoutingState(context.Background(), &SmartRoutingRequestState{SmartRoutingActive: true})

	svc.ClearSmartRoutingStickySession(ctx, nil, "   ")

	require.Empty(t, cache.calls)
}

func TestSnapshotOpenAICompatibilityFallbackMetrics(t *testing.T) {
	before := SnapshotOpenAICompatibilityFallbackMetrics()

	ctx := context.WithValue(context.Background(), ctxkey.ThinkingEnabled, true)
	_, _ = ThinkingEnabledFromContext(ctx)

	after := SnapshotOpenAICompatibilityFallbackMetrics()
	require.GreaterOrEqual(t, after.MetadataLegacyFallbackTotal, before.MetadataLegacyFallbackTotal+1)
	require.GreaterOrEqual(t, after.MetadataLegacyFallbackThinkingEnabledTotal, before.MetadataLegacyFallbackThinkingEnabledTotal+1)
}
