package bepusdt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/payment/provider"
)

const ProviderKey payment.PaymentType = "bepusdt"

// Register installs BEpusdt through the generic Overlay payment extension points.
func Register() {
	provider.RegisterExternalProvider(ProviderKey, New)
	payment.RegisterExternalProviderDefinition(payment.ExternalProviderDefinition{
		ProviderKey:                       ProviderKey,
		SensitiveConfigFields:             []string{"apiToken"},
		PendingOrderProtectedConfigFields: []string{"apiToken", "apiBase", "fiat", "currencies"},
		VisibleMethods:                    []payment.PaymentType{ProviderKey},
		DisplayNameConfigKey:              "displayName",
		DefaultDisplayName:                "USDT",
		CurrencyConfigKey:                 "fiat",
		DefaultCurrency:                   "CNY",
		WebhookPath:                       "bepusdt",
		WebhookOrderID:                    extractWebhookOrderID,
	})
}

type Config struct {
	APIBase    string
	APIToken   string
	Fiat       string
	Currencies string
	Timeout    string
	Reselect   string
}

type Provider struct {
	instanceID string
	config     map[string]string
	client     *http.Client
}

func init() { Register() }

func New(instanceID string, cfg map[string]string) (payment.Provider, error) {
	base := strings.TrimRight(strings.TrimSpace(cfg["apiBase"]), "/")
	if base == "" || strings.TrimSpace(cfg["apiToken"]) == "" {
		return nil, fmt.Errorf("bepusdt apiBase and apiToken are required")
	}
	copyCfg := map[string]string{}
	for k, v := range cfg {
		copyCfg[k] = v
	}
	copyCfg["apiBase"] = base
	if fiat := strings.ToUpper(strings.TrimSpace(copyCfg["fiat"])); fiat != "" && fiat != "CNY" {
		return nil, fmt.Errorf("bepusdt fiat must be CNY")
	}
	copyCfg["fiat"] = "CNY"
	if strings.TrimSpace(copyCfg["currencies"]) == "" {
		copyCfg["currencies"] = "USDT"
	}
	if parseTimeout(copyCfg["timeout"]) == 0 {
		copyCfg["timeout"] = "600"
	}
	return &Provider{instanceID: instanceID, config: copyCfg, client: &http.Client{}}, nil
}

func (p *Provider) Name() string                          { return "BEpusdt" }
func (p *Provider) ProviderKey() string                   { return ProviderKey }
func (p *Provider) SupportedTypes() []payment.PaymentType { return []payment.PaymentType{ProviderKey} }
func (p *Provider) MerchantIdentityMetadata() map[string]string {
	return map[string]string{"fiat": strings.ToUpper(p.config["fiat"]), "currencies": p.config["currencies"]}
}

type createRequest struct {
	OrderID     string  `json:"order_id"`
	Amount      float64 `json:"amount"`
	Fiat        string  `json:"fiat"`
	Currencies  string  `json:"currencies"`
	Name        string  `json:"name"`
	NotifyURL   string  `json:"notify_url"`
	RedirectURL string  `json:"redirect_url"`
	Reselect    bool    `json:"reselect"`
	Timeout     int     `json:"timeout"`
	Signature   string  `json:"signature"`
}

type createResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Data       struct {
		TradeID    string `json:"trade_id"`
		PaymentURL string `json:"payment_url"`
	} `json:"data"`
}

func (p *Provider) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	amount, err := parseAmount(req.Amount)
	if err != nil || amount <= 0 {
		return nil, fmt.Errorf("bepusdt create: invalid amount %q", req.Amount)
	}
	notifyURL := strings.TrimSpace(req.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(p.config["notifyUrl"])
	}
	if notifyURL == "" || strings.TrimSpace(req.ReturnURL) == "" {
		return nil, fmt.Errorf("bepusdt create: notify and return URLs are required")
	}
	r := createRequest{OrderID: req.OrderID, Amount: amount, Fiat: strings.ToUpper(p.config["fiat"]), Currencies: p.config["currencies"], Name: req.Subject, NotifyURL: notifyURL, RedirectURL: req.ReturnURL, Reselect: strings.EqualFold(p.config["reselect"], "true"), Timeout: parseTimeout(p.config["timeout"])}
	r.Signature = signMap(map[string]any{"amount": r.Amount, "currencies": r.Currencies, "fiat": r.Fiat, "name": r.Name, "notify_url": r.NotifyURL, "order_id": r.OrderID, "redirect_url": r.RedirectURL, "reselect": r.Reselect, "timeout": r.Timeout}, p.config["apiToken"])
	body, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config["apiBase"]+"/api/v1/order/create-order", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bepusdt create: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out createResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("bepusdt response: %w", err)
	}
	if resp.StatusCode/100 != 2 || out.StatusCode != 200 || out.Data.PaymentURL == "" {
		return nil, fmt.Errorf("bepusdt create failed: %s", out.Message)
	}
	return &payment.CreatePaymentResponse{TradeNo: out.Data.TradeID, PayURL: out.Data.PaymentURL}, nil
}

func (p *Provider) QueryOrder(context.Context, string) (*payment.QueryOrderResponse, error) {
	return nil, fmt.Errorf("bepusdt order query is callback-only")
}
func (p *Provider) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("bepusdt refunds are unsupported")
}
func (p *Provider) CancelPayment(ctx context.Context, tradeNo string) error {
	payload := map[string]any{"trade_id": tradeNo}
	payload["signature"] = signMap(map[string]any{"trade_id": tradeNo}, p.config["apiToken"])
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config["apiBase"]+"/api/v1/order/cancel-transaction", strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("bepusdt cancel status %s", res.Status)
	}
	return nil
}

type notification struct {
	TradeID      string  `json:"trade_id"`
	OrderID      string  `json:"order_id"`
	Amount       float64 `json:"amount"`
	ActualAmount string  `json:"actual_amount"`
	Token        string  `json:"token"`
	BlockTx      string  `json:"block_transaction_id"`
	Signature    string  `json:"signature"`
	Status       int     `json:"status"`
}

func extractWebhookOrderID(raw string) string {
	var n struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		return ""
	}
	return strings.TrimSpace(n.OrderID)
}

func (p *Provider) VerifyNotification(_ context.Context, raw string, _ map[string]string) (*payment.PaymentNotification, error) {
	var n notification
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		return nil, fmt.Errorf("bepusdt webhook JSON: %w", err)
	}
	if n.Signature == "" || signMap(map[string]any{"actual_amount": n.ActualAmount, "amount": n.Amount, "block_transaction_id": n.BlockTx, "order_id": n.OrderID, "status": n.Status, "token": n.Token, "trade_id": n.TradeID}, p.config["apiToken"]) != strings.ToLower(n.Signature) {
		return nil, fmt.Errorf("bepusdt invalid signature")
	}
	if n.Status != 2 {
		return nil, nil
	}
	if strings.TrimSpace(n.TradeID) == "" || strings.TrimSpace(n.OrderID) == "" || n.Amount <= 0 {
		return nil, fmt.Errorf("bepusdt notification is missing required payment fields")
	}
	return &payment.PaymentNotification{TradeNo: n.TradeID, OrderID: n.OrderID, Amount: n.Amount, Status: payment.NotificationStatusSuccess, RawData: raw, Metadata: map[string]string{"fiat": strings.ToUpper(p.config["fiat"]), "actual_amount": n.ActualAmount, "token": n.Token, "block_transaction_id": n.BlockTx}}, nil
}

func parseAmount(v string) (float64, error) {
	var n float64
	_, err := fmt.Sscanf(v, "%f", &n)
	return n, err
}

func parseTimeout(v string) int {
	var timeout int
	_, _ = fmt.Sscanf(v, "%d", &timeout)
	if timeout < 180 {
		return 600
	}
	return timeout
}
