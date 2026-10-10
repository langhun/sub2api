package xznoauthinvoice

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestClientTokenFormAndCache(t *testing.T) {
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth/token":
			tokenCalls.Add(1)
			if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("content type = %q", got)
			}
			body, _ := io.ReadAll(r.Body)
			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatal(err)
			}
			if values.Get("grant_type") != "client_credentials" ||
				values.Get("client_id") != "client" ||
				values.Get("client_secret") != "secret" ||
				values.Get("scope") != "invoice.apply" {
				t.Fatalf("unexpected token form: %s", body)
			}
			_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer","expires_in":900}`))
		case "/api/v1/invoice-orders/validate":
			if got := r.Header.Get("Authorization"); got != "Bearer token" {
				t.Fatalf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"orders":[],"totalAmount":"0.00","currency":"CNY"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(config.XZNOAuthInvoiceConfig{
		Enabled:      true,
		BaseURL:      server.URL,
		ClientID:     "client",
		ClientSecret: "secret",
	})
	if _, err := client.Validate(context.Background(), validateRequest{OrderNos: []string{"ORDER-1"}, CreateTaxPayments: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Validate(context.Background(), validateRequest{OrderNos: []string{"ORDER-2"}, CreateTaxPayments: false}); err != nil {
		t.Fatal(err)
	}
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("token calls = %d, want 1", got)
	}
}

func TestRemoteErrorPreservesSafeCodeAndRequestID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":900}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"code":"IDENTITY_VERIFICATION_REQUIRED","message":"verify owner","requestId":"req-123"}`))
	}))
	defer server.Close()

	client := NewClient(config.XZNOAuthInvoiceConfig{Enabled: true, BaseURL: server.URL, ClientID: "client", ClientSecret: "secret"})
	_, err := client.Validate(context.Background(), validateRequest{OrderNos: []string{"ORDER-1"}})
	remote, ok := err.(*RemoteError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if remote.Code != "IDENTITY_VERIFICATION_REQUIRED" || remote.RequestID != "req-123" {
		t.Fatalf("remote error = %#v", remote)
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token") {
		t.Fatalf("sensitive value leaked: %q", err.Error())
	}
}

func TestClientRefreshesInvalidTokenOnce(t *testing.T) {
	var tokenCalls atomic.Int32
	var validateCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth/token":
			tokenCalls.Add(1)
			if tokenCalls.Load() == 1 {
				_, _ = w.Write([]byte(`{"access_token":"stale","expires_in":900}`))
				return
			}
			_, _ = w.Write([]byte(`{"access_token":"fresh","expires_in":900}`))
		case "/api/v1/invoice-orders/validate":
			if validateCalls.Add(1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"code":"AUTH_INVALID_TOKEN","message":"expired"}`))
				return
			}
			if got := r.Header.Get("Authorization"); got != "Bearer fresh" {
				t.Fatalf("authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"orders":[],"totalAmount":"0.00","currency":"CNY"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(config.XZNOAuthInvoiceConfig{
		Enabled:      true,
		BaseURL:      server.URL,
		ClientID:     "client",
		ClientSecret: "secret",
	})
	if _, err := client.Validate(context.Background(), validateRequest{OrderNos: []string{"ORDER-1"}}); err != nil {
		t.Fatal(err)
	}
	if got := tokenCalls.Load(); got != 2 {
		t.Fatalf("token calls = %d, want 2", got)
	}
}
