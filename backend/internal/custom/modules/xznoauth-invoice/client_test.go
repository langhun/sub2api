package xznoauthinvoice

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestClientTaxStatusUsesMerchantContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/oauth/token":
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":900}`))
		case "/api/v1/invoice-tax-payments/status":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body["taxOrderNo"] != "INV-TAX-1" {
				t.Fatalf("unexpected tax status body: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"paid":true}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(config.XZNOAuthInvoiceConfig{Enabled: true, BaseURL: server.URL, ClientID: "client", ClientSecret: "secret"})
	paid, err := client.TaxStatus(context.Background(), "INV-TAX-1")
	if err != nil || !paid {
		t.Fatalf("tax status = %v, %v", paid, err)
	}
}

func TestClientInvoiceEndpointContracts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":900}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.URL.Path == "/api/v1/invoice-orders/validate":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["needPayTax"] != true || body["createTaxPayments"] != false || len(body["orderNos"].([]any)) != 1 {
				t.Fatalf("unexpected validation request: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"orders":[{"orderNo":"ORDER-1","amount":"10.00","currency":"CNY"}],"totalAmount":"10.00","taxAmount":"0.60","taxPaidAmount":"0.00","taxDueAmount":"0.60","currency":"CNY"}}`))
		case r.URL.Path == "/api/v1/invoices" && r.Method == http.MethodPost:
			if r.Method != http.MethodPost {
				t.Fatalf("create method = %s", r.Method)
			}
			var body BuyerInput
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body.OrderNos) != 1 || body.OrderNos[0] != "ORDER-1" || !body.NeedPayTax || body.TaxOrderNos != nil {
				t.Fatalf("unexpected create request: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"code":0,"data":{"id":42,"status":"awaiting_tax_payment"}}`))
		case r.URL.Path == "/api/v1/invoices/42/tax-payments":
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || len(body) != 0 {
				t.Fatalf("checkout method/body = %s/%q", r.Method, body)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"application":{"id":42,"status":"awaiting_tax_payment"},"taxPaidAmount":"0.00","taxDueAmount":"0.60","taxPayments":{"alipay":{"taxOrderNo":"INV-TAX-42","payUrl":"https://pay.example/42"}}}}`))
		case r.URL.Path == "/api/v1/invoice-tax-payments/status":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body) != 1 || body["taxOrderNo"] != "INV-TAX-42" {
				t.Fatalf("unexpected payment status request: %#v", body)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"paid":true}}`))
		case r.URL.Path == "/api/v1/invoices" && r.Method == http.MethodGet:
			if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("pageSize") != "20" {
				t.Fatalf("list query = %s", r.URL.RawQuery)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"items":[],"total":0,"page":1,"page_size":20,"pages":1}}`))
		case r.URL.Path == "/api/v1/invoices/42/cancel":
			body, _ := io.ReadAll(r.Body)
			if r.Method != http.MethodPost || len(body) != 0 {
				t.Fatalf("cancel method/body = %s/%q", r.Method, body)
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"id":42,"status":"canceled"}}`))
		case r.URL.Path == "/api/v1/invoices/42/pdf":
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7\nfixture"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(config.XZNOAuthInvoiceConfig{Enabled: true, BaseURL: server.URL, ClientID: "client", ClientSecret: "secret"})
	validated, err := client.Validate(context.Background(), validateRequest{OrderNos: []string{"ORDER-1"}, NeedPayTax: true, CreateTaxPayments: false})
	if err != nil || validated.TaxDueAmount != "0.60" {
		t.Fatalf("validate = %#v, %v", validated, err)
	}
	created, err := client.Create(context.Background(), BuyerInput{OrderNos: []string{"ORDER-1"}, NeedPayTax: true, Title: "Example", TaxpayerID: "ID", RecipientEmail: "billing@example.com"})
	if err != nil || created.ID != 42 || created.Status != "awaiting_tax_payment" {
		t.Fatalf("create = %#v, %v", created, err)
	}
	checkout, err := client.Checkout(context.Background(), created.ID)
	if err != nil || checkout.Application.ID != created.ID || checkout.TaxPayments["alipay"].TaxOrderNo != "INV-TAX-42" {
		t.Fatalf("checkout = %#v, %v", checkout, err)
	}
	paid, err := client.TaxStatus(context.Background(), "INV-TAX-42")
	if err != nil || !paid {
		t.Fatalf("tax status = %v, %v", paid, err)
	}
	if _, err := client.List(context.Background(), 1, 20); err != nil {
		t.Fatal(err)
	}
	canceled, err := client.Cancel(context.Background(), created.ID)
	if err != nil || canceled.Status != "canceled" {
		t.Fatalf("cancel = %#v, %v", canceled, err)
	}
	pdf, err := client.PDF(context.Background(), created.ID)
	if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF = %q, %v", pdf, err)
	}
}

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
	var tokenCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/oauth/token" {
			tokenCalls.Add(1)
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
	if got := tokenCalls.Load(); got != 1 {
		t.Fatalf("identity verification error triggered %d token calls, want 1", got)
	}
	if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "token") {
		t.Fatalf("sensitive value leaked: %q", err.Error())
	}
}

func TestClientRejectsNonPDFSuccessResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/oauth/token" {
			_, _ = w.Write([]byte(`{"access_token":"token","expires_in":900}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":"INVOICE_PDF_NOT_READY","message":"not ready"}`))
	}))
	defer server.Close()

	client := NewClient(config.XZNOAuthInvoiceConfig{Enabled: true, BaseURL: server.URL, ClientID: "client", ClientSecret: "secret"})
	_, err := client.PDF(context.Background(), 42)
	remote, ok := err.(*RemoteError)
	if !ok || remote.Code != "INVOICE_PDF_NOT_READY" {
		t.Fatalf("PDF error = %#v", err)
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
