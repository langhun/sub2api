package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAntigravityOAuthService_GenerateAuthURL_UsesProxyPoolMode(t *testing.T) {
	settingRepo := &settingRepoStubForPool{values: map[string]string{
		SettingKeyAutoFailoverProxyPool: "[1]",
	}}
	settingSvc := NewSettingService(settingRepo, nil)
	proxyRepo := &proxyRepoStubForPool{
		proxies: map[int64]Proxy{
			1: {ID: 1, Name: "pool", Protocol: "http", Host: "pool.example", Port: 8080, Status: StatusActive},
		},
	}
	proxyPool := NewAutoFailoverProxyPoolService(proxyRepo, nil, settingSvc, &proxyLatencyCacheStubForPool{}, nil)
	svc := NewAntigravityOAuthService(proxyRepo)
	svc.SetAutoFailoverProxyPool(proxyPool)

	result, err := svc.GenerateAuthURL(context.Background(), nil, AccountProxyModePool)
	require.NoError(t, err)
	require.NotEmpty(t, result.SessionID)

	session, ok := svc.sessionStore.Get(result.SessionID)
	require.True(t, ok)
	require.Equal(t, "http://pool.example:8080", session.ProxyURL)
}
