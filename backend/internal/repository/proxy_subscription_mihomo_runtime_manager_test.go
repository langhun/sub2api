package repository

import (
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
