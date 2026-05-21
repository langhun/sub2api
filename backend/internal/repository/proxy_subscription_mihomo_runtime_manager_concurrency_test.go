package repository

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	appconfig "github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestConcurrentRuntimeAccess(t *testing.T) {
	tmpDir := t.TempDir()
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

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.CheckRuntime(ctx, "src-1-subscription-1")
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.mu.Lock()
			_, _ = manager.buildEmbeddedConfigLocked()
			manager.mu.Unlock()
		}()
	}

	wg.Wait()
}

func TestConcurrentCheckRuntimeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := &proxySubscriptionMihomoRuntimeManager{
		cfg:         appconfig.ProxySubscriptionMihomoConfig{ListenerPortRange: "21080-21180"},
		dataDir:     tmpDir,
		listenerDir: tmpDir,
		runtimes:    map[string]*mihomoRuntimeState{},
	}

	var wg sync.WaitGroup
	ctx := context.Background()
	runtimeID := "non-existent-runtime"

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := manager.CheckRuntime(ctx, runtimeID)
			require.Error(t, err)
			require.Contains(t, err.Error(), "runtime not found")
		}()
	}

	wg.Wait()
}

func TestConcurrentDeleteAndCheck(t *testing.T) {
	tmpDir := t.TempDir()
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

	var wg sync.WaitGroup
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.CheckRuntime(ctx, "src-1-subscription-1")
		}()
	}

	wg.Wait()
}
