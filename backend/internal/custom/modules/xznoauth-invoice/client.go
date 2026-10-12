package xznoauthinvoice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type RemoteError struct {
	Status    int
	Code      string
	Message   string
	RequestID string
}

func (e *RemoteError) Error() string {
	if e == nil {
		return "invoice upstream error"
	}
	return e.Message
}

type apiEnvelope struct {
	Code      json.RawMessage `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"requestId"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type Client struct {
	cfg        config.XZNOAuthInvoiceConfig
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	tokenUntil time.Time
}

func NewClient(cfg config.XZNOAuthInvoiceConfig) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://oauth.xzncraft.cn"
	}
	cfg.BaseURL = base
	return &Client{cfg: cfg, httpClient: &http.Client{Timeout: timeout}}
}

func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enabled && c.cfg.ClientID != "" && c.cfg.ClientSecret != ""
}

func (c *Client) tokenFor(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.token != "" && time.Now().Before(c.tokenUntil) {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("scope", "invoice.apply")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/api/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("invoice token request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &RemoteError{Status: resp.StatusCode, Code: "OAUTH_INVALID_CLIENT", Message: "invoice service authentication failed"}
	}
	var token tokenResponse
	if err := json.Unmarshal(body, &token); err != nil || token.AccessToken == "" {
		return "", errors.New("invoice service returned an invalid token")
	}
	ttl := time.Duration(token.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	cacheTTL := ttl - 30*time.Second
	if cacheTTL <= 0 {
		cacheTTL = ttl / 2
		if cacheTTL <= 0 {
			cacheTTL = time.Second
		}
	}
	c.mu.Lock()
	c.token = token.AccessToken
	c.tokenUntil = time.Now().Add(cacheTTL)
	c.mu.Unlock()
	return token.AccessToken, nil
}

func (c *Client) invalidateToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if token == "" || c.token == token {
		c.token = ""
		c.tokenUntil = time.Time{}
	}
}

func (c *Client) doJSON(ctx context.Context, method, path string, input any, output any) error {
	if !c.Enabled() {
		return errors.New("invoice service is not configured")
	}
	var body []byte
	if input != nil {
		data, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = data
	}
	token, err := c.tokenFor(ctx)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 2; attempt++ {
		var requestBody io.Reader
		if body != nil {
			requestBody = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, c.cfg.BaseURL+path, requestBody)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("invoice upstream request: %w", err)
		}
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		_ = resp.Body.Close()
		var envelope apiEnvelope
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return fmt.Errorf("invoice upstream response: %w", err)
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && string(envelope.Code) == "0" {
			if output == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
				return nil
			}
			return json.Unmarshal(envelope.Data, output)
		}

		code := strings.Trim(string(envelope.Code), `"`)
		if code == "" {
			code = strconv.Itoa(resp.StatusCode)
		}
		if attempt == 0 && shouldRefreshToken(resp.StatusCode, code) {
			c.invalidateToken(token)
			token, err = c.tokenFor(ctx)
			if err != nil {
				return err
			}
			continue
		}
		message := envelope.Message
		if message == "" {
			message = "invoice request failed"
		}
		return &RemoteError{Status: resp.StatusCode, Code: code, Message: message, RequestID: envelope.RequestID}
	}
	return errors.New("invoice upstream request failed")
}

func shouldRefreshToken(status int, code string) bool {
	if status == http.StatusUnauthorized {
		return true
	}
	return code == "AUTH_INVALID_TOKEN"
}

func (c *Client) Validate(ctx context.Context, req validateRequest) (Validation, error) {
	var out Validation
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/invoice-orders/validate", req, &out)
	return out, err
}

func (c *Client) Create(ctx context.Context, input BuyerInput) (InvoiceApplication, error) {
	var out InvoiceApplication
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/invoices", input, &out)
	return out, err
}

func (c *Client) Checkout(ctx context.Context, id int64) (TaxCheckout, error) {
	var out TaxCheckout
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/invoices/"+strconv.FormatInt(id, 10)+"/tax-payments", nil, &out)
	return out, err
}

func (c *Client) TaxStatus(ctx context.Context, taxOrderNo string) (bool, error) {
	var out struct {
		Paid bool `json:"paid"`
	}
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/invoice-tax-payments/status", map[string]string{"taxOrderNo": taxOrderNo}, &out)
	return out.Paid, err
}

func (c *Client) List(ctx context.Context, page, pageSize int) (InvoicePage, error) {
	var out InvoicePage
	path := "/api/v1/invoices?page=" + strconv.Itoa(page) + "&pageSize=" + strconv.Itoa(pageSize)
	err := c.doJSON(ctx, http.MethodGet, path, nil, &out)
	return out, err
}

func (c *Client) Cancel(ctx context.Context, id int64) (InvoiceApplication, error) {
	var out InvoiceApplication
	err := c.doJSON(ctx, http.MethodPost, "/api/v1/invoices/"+strconv.FormatInt(id, 10)+"/cancel", nil, &out)
	return out, err
}

func (c *Client) PDF(ctx context.Context, id int64) ([]byte, error) {
	if !c.Enabled() {
		return nil, errors.New("invoice service is not configured")
	}
	token, err := c.tokenFor(ctx)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/api/v1/invoices/"+strconv.FormatInt(id, 10)+"/pdf", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/pdf")
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && bytes.HasPrefix(data, []byte("%PDF-")) {
			return data, nil
		}
		var envelope apiEnvelope
		_ = json.Unmarshal(data, &envelope)
		code := strings.Trim(string(envelope.Code), `"`)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && code == "" {
			code = "INVOICE_INVALID_PDF_RESPONSE"
		}
		if code == "" {
			code = strconv.Itoa(resp.StatusCode)
		}
		if attempt == 0 && shouldRefreshToken(resp.StatusCode, code) {
			c.invalidateToken(token)
			token, err = c.tokenFor(ctx)
			if err != nil {
				return nil, err
			}
			continue
		}
		return nil, &RemoteError{Status: resp.StatusCode, Code: code, Message: envelope.Message, RequestID: envelope.RequestID}
	}
	return nil, errors.New("invoice PDF request failed")
}
