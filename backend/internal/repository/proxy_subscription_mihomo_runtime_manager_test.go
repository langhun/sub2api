package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	appconfig "github.com/Wei-Shaw/sub2api/internal/config"
	mihomoConstant "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/hub/executor"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuildEmbeddedMihomoConfigParses(t *testing.T) {
	tmpDir := t.TempDir()
	mihomoConstant.SetHomeDir(tmpDir)
	providerPath := filepath.Join(tmpDir, "src-1-subscription-1.provider.yaml")
	require.NoError(t, os.WriteFile(providerPath, []byte("proxies:\n  - name: test-node\n    type: http\n    server: 127.0.0.1\n    port: 8080\n"), 0o644))

	manager := &proxySubscriptionMihomoRuntimeManager{
		cfg:         appconfig.ProxySubscriptionMihomoConfig{ListenerPortRange: "21080-21180"},
		dataDir:     tmpDir,
		listenerDir: tmpDir,
		runtimes: map[string]*mihomoRuntimeState{
			"src-1-subscription-1": {
				RuntimeID:    "src-1-subscription-1",
				ProviderPath: providerPath,
				ListenerHost: "127.0.0.1",
				ListenerPort: 21080,
			},
		},
	}

	raw, err := manager.buildEmbeddedConfigLocked()
	require.NoError(t, err)

	var cfg mihomoConfigFile
	require.NoError(t, yaml.Unmarshal(raw, &cfg))
	require.Empty(t, cfg.SocksPort)
	require.Len(t, cfg.Listeners, 1)
	require.Equal(t, "socks", cfg.Listeners[0].Type)
	require.Equal(t, "sub2api-src-1-subscription-1-group", cfg.Listeners[0].Proxy)
	require.Contains(t, cfg.ProxyProviders, "sub2api-src-1-subscription-1-provider")

	_, err = executor.ParseWithBytes(raw)
	require.NoError(t, err)
}

func TestRehydrateReassignsDuplicateListenerPorts(t *testing.T) {
	tmpDir := t.TempDir()
	for _, runtimeID := range []string{"src-1-subscription-1", "src-1-subscription-2"} {
		providerPath := filepath.Join(tmpDir, runtimeID+".provider.yaml")
		require.NoError(t, os.WriteFile(providerPath, []byte("proxies:\n  - name: test-node\n    type: http\n    server: 127.0.0.1\n    port: 8080\n"), 0o644))

		raw, err := buildMihomoConfig(runtimeID, providerPath, "127.0.0.1", 21080)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, runtimeID+".yaml"), raw, 0o644))
	}

	manager := &proxySubscriptionMihomoRuntimeManager{
		cfg:         appconfig.ProxySubscriptionMihomoConfig{ListenerPortRange: "21080-21082"},
		dataDir:     tmpDir,
		listenerDir: tmpDir,
		runtimes:    map[string]*mihomoRuntimeState{},
	}

	require.NoError(t, manager.rehydrateExistingRuntimesLocked(context.Background()))
	require.Equal(t, 21080, manager.runtimes["src-1-subscription-1"].ListenerPort)
	require.Equal(t, 21081, manager.runtimes["src-1-subscription-2"].ListenerPort)

	var rehydrated mihomoConfigFile
	raw, err := os.ReadFile(filepath.Join(tmpDir, "src-1-subscription-2.yaml"))
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(raw, &rehydrated))
	require.Equal(t, 21081, rehydrated.SocksPort)

	embedded, err := manager.buildEmbeddedConfigLocked()
	require.NoError(t, err)
	var cfg mihomoConfigFile
	require.NoError(t, yaml.Unmarshal(embedded, &cfg))
	require.Len(t, cfg.Listeners, 2)
	require.NotEqual(t, cfg.Listeners[0].Port, cfg.Listeners[1].Port)
}
