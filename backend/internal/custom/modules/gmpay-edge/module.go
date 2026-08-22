package gmpayedge

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/payment/provider"
)

const ProviderKey payment.PaymentType = "gmpay_edge"

const (
	defaultCurrency = "CNY"
	tradeSuccess    = "TRADE_SUCCESS"
	maxBodySize     = 1 << 20
)

// Register installs GMPay Edge through generic Overlay payment extension points.
func Register() {
	provider.RegisterExternalProvider(ProviderKey, New)
	payment.RegisterExternalProviderDefinition(payment.ExternalProviderDefinition{
		ProviderKey:                       ProviderKey,
		SensitiveConfigFields:             []string{"apiSecret"},
		PendingOrderProtectedConfigFields: []string{"apiSecret", "apiBase", "pid"},
		VisibleMethods:                    []payment.PaymentType{ProviderKey},
		DisplayNameConfigKey:              "displayName",
		DefaultDisplayName:                "USDT / USDC",
		CurrencyConfigKey:                 "currency",
		DefaultCurrency:                   defaultCurrency,
		WebhookPath:                       "gmpay_edge",
		WebhookOrderID:                    extractWebhookOrderID,
		WebhookAck:                        "ok",
	})
}

type Provider struct {
	instanceID string
	config     map[string]string
	client     *http.Client
}

func init() { Register() }

func New(instanceID string, cfg map[string]string) (payment.Provider, error) {
	for _, key := range []string{"pid", "apiSecret", "apiBase", "notifyUrl", "returnUrl"} {
		if strings.TrimSpace(cfg[key]) == "" {
			return nil, fmt.Errorf("gmpay edge config missing required key: %s", key)
		}
	}
	copyCfg := make(map[string]string, len(cfg)+1)
	for key, value := range cfg {
		copyCfg[key] = value
	}
	copyCfg["apiBase"] = normalizeAPIBase(copyCfg["apiBase"])
	if copyCfg["apiBase"] == "" {
		return nil, fmt.Errorf("gmpay edge apiBase is invalid")
	}
	copyCfg["currency"] = defaultCurrency
	return &Provider{
		instanceID: instanceID,
		config:     copyCfg,
		client:     &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (p *Provider) Name() string        { return "GMPay Edge" }
func (p *Provider) ProviderKey() string { return ProviderKey }
func (p *Provider) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{ProviderKey}
}

func (p *Provider) MerchantIdentityMetadata() map[string]string {
	return map[string]string{"pid": strings.TrimSpace(p.config["pid"]), "currency": defaultCurrency}
}

type createResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"msg"`
	TradeNo string          `json:"trade_no"`
	PayURL  string          `json:"payurl"`
	QRCode  string          `json:"qrcode"`
	Data    json.RawMessage `json:"data"`
}

func (p *Provider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	amount := strings.TrimSpace(req.Amount)
	if !isPositiveDecimal(amount) {
		return nil, fmt.Errorf("gmpay edge create: invalid amount %q", req.Amount)
	}
	notifyURL := strings.TrimSpace(req.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(p.config["notifyUrl"])
	}
	returnURL := strings.TrimSpace(req.ReturnURL)
	if returnURL == "" {
		returnURL = strings.TrimSpace(p.config["returnUrl"])
	}
	params := map[string]string{
		"pid":          p.config["pid"],
		"money":        amount,
		"out_trade_no": req.OrderID,
		"notify_url":   notifyURL,
		"return_url":   returnURL,
		"name":         req.Subject,
	}
	params["sign"] = sign(params, p.config["apiSecret"])
	params["sign_type"] = "MD5"

	body, err := p.postForm(ctx, "/mapi.php", params)
	if err != nil {
		return nil, fmt.Errorf("gmpay edge create: %w", err)
	}
	var response createResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("gmpay edge parse create response: %w", err)
	}
	if response.Code != 1 {
		message := strings.TrimSpace(response.Message)
		if message == "" {
			message = strings.TrimSpace(string(response.Data))
		}
		return nil, fmt.Errorf("gmpay edge create failed: %s", message)
	}
	payURL := strings.TrimSpace(response.PayURL)
	if payURL == "" {
		payURL = strings.TrimSpace(response.QRCode)
	}
	if payURL == "" {
		return nil, fmt.Errorf("gmpay edge create response missing payment URL")
	}
	return &payment.CreatePaymentResponse{TradeNo: response.TradeNo, PayURL: payURL, QRCode: response.QRCode}, nil
}

func (p *Provider) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	params := map[string]string{"act": "order", "pid": p.config["pid"], "out_trade_no": tradeNo}
	params["sign"] = sign(params, p.config["apiSecret"])
	body, err := p.get(ctx, "/api.php", params)
	if err != nil {
		return nil, fmt.Errorf("gmpay edge query: %w", err)
	}
	var response struct {
		Code        int    `json:"code"`
		Message     string `json:"msg"`
		TradeNo     string `json:"trade_no"`
		TradeStatus string `json:"trade_status"`
		Money       string `json:"money"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("gmpay edge parse query response: %w", err)
	}
	if response.Code != 1 {
		return nil, fmt.Errorf("gmpay edge query failed: %s", response.Message)
	}
	status := payment.ProviderStatusPending
	if response.TradeStatus == tradeSuccess {
		status = payment.ProviderStatusPaid
	}
	amount, _ := strconv.ParseFloat(response.Money, 64)
	if response.TradeNo == "" {
		response.TradeNo = tradeNo
	}
	return &payment.QueryOrderResponse{TradeNo: response.TradeNo, Status: status, Amount: amount, Metadata: p.MerchantIdentityMetadata()}, nil
}

func (p *Provider) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return nil, fmt.Errorf("gmpay edge parse notification: %w", err)
	}
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	received := strings.TrimSpace(params["sign"])
	if received == "" || !hmac.Equal([]byte(sign(params, p.config["apiSecret"])), []byte(received)) {
		return nil, fmt.Errorf("gmpay edge invalid signature")
	}
	if params["trade_status"] != tradeSuccess {
		return nil, nil
	}
	orderID := strings.TrimSpace(params["out_trade_no"])
	tradeNo := strings.TrimSpace(params["trade_no"])
	amount, err := strconv.ParseFloat(strings.TrimSpace(params["money"]), 64)
	if orderID == "" || tradeNo == "" || err != nil || amount <= 0 {
		return nil, fmt.Errorf("gmpay edge notification is missing required payment fields")
	}
	return &payment.PaymentNotification{
		TradeNo: tradeNo,
		OrderID: orderID,
		Amount:  amount,
		Status:  payment.NotificationStatusSuccess,
		RawData: rawBody,
		Metadata: map[string]string{
			"pid":      params["pid"],
			"currency": defaultCurrency,
			"type":     params["type"],
		},
	}, nil
}

func (p *Provider) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("gmpay edge refunds are unsupported")
}

func normalizeAPIBase(value string) string {
	base := strings.TrimRight(strings.TrimSpace(value), "/")
	for _, suffix := range []string{"/mapi.php", "/submit.php", "/api.php"} {
		if strings.HasSuffix(strings.ToLower(base), suffix) {
			return strings.TrimRight(base[:len(base)-len(suffix)], "/")
		}
	}
	return base
}

func (p *Provider) postForm(ctx context.Context, suffix string, params map[string]string) ([]byte, error) {
	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config["apiBase"]+suffix, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (p *Provider) get(ctx context.Context, suffix string, params map[string]string) ([]byte, error) {
	query := url.Values{}
	for key, value := range params {
		query.Set(key, value)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.config["apiBase"]+suffix+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key == "sign" || key == "sign_type" || value == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for index, key := range keys {
		if index > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(params[key])
	}
	builder.WriteString(secret)
	digest := md5.Sum([]byte(builder.String()))
	return hex.EncodeToString(digest[:])
}

func extractWebhookOrderID(rawBody string) string {
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(values.Get("out_trade_no"))
}

func isPositiveDecimal(value string) bool {
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed > 0 && !strings.ContainsAny(value, "eE")
}
