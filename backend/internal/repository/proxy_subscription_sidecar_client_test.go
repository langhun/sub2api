package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func TestProxySubscriptionSidecarClient_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewProxySubscriptionSidecarClient(&config.Config{
		ProxySubscriptionSidecar: config.ProxySubscriptionSidecarConfig{
			Enabled: true,
			BaseURL: server.URL,
		},
	})
	if client == nil {
		t.Fatal("expected client")
	}
	if err := client.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck error = %v", err)
	}
}

func TestProxySubscriptionSidecarClient_UpsertRuntime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/runtimes/upsert" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"runtime_id":"demo","listener_host":"127.0.0.1","listener_port":21080,"protocol":"socks5h"}`))
	}))
	defer server.Close()

	client := NewProxySubscriptionSidecarClient(&config.Config{
		ProxySubscriptionSidecar: config.ProxySubscriptionSidecarConfig{
			Enabled: true,
			BaseURL: server.URL,
		},
	})
	resp, err := client.UpsertRuntime(context.Background(), service.ProxySubscriptionSidecarUpsertRequest{
		RuntimeID:   "demo",
		NodeType:    "vmess",
		DisplayName: "demo",
		Server:      "example.com",
		Port:        443,
		Config:      map[string]any{"uuid": "11111111-1111-1111-1111-111111111111"},
	})
	if err != nil {
		t.Fatalf("UpsertRuntime error = %v", err)
	}
	if resp == nil || resp.ListenerPort != 21080 {
		t.Fatalf("unexpected response = %#v", resp)
	}
}
