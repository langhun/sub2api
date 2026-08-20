package bepusdt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentSignsAndReturnsCheckoutURL(t *testing.T) {
	var received createRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/order/create-order", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		_, _ = w.Write([]byte(`{"status_code":200,"message":"success","data":{"trade_id":"trade-1","payment_url":"https://checkout.example/order-1"}}`))
	}))
	defer server.Close()

	provider, err := New("12", map[string]string{"apiBase": server.URL, "apiToken": "secret", "notifyUrl": "https://sub.example/api/v1/payment/webhook/bepusdt"})
	require.NoError(t, err)
	result, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{OrderID: "sub2_1", Amount: "12.34", Subject: "Recharge", ReturnURL: "https://sub.example/payment/result"})
	require.NoError(t, err)
	require.Equal(t, "trade-1", result.TradeNo)
	require.Equal(t, "https://checkout.example/order-1", result.PayURL)
	require.Equal(t, "USDT", received.Currencies)
	require.Equal(t, 600, received.Timeout)
	require.Equal(t, signMap(map[string]any{"amount": received.Amount, "currencies": received.Currencies, "fiat": received.Fiat, "name": received.Name, "notify_url": received.NotifyURL, "order_id": received.OrderID, "redirect_url": received.RedirectURL, "reselect": received.Reselect, "timeout": received.Timeout}, "secret"), received.Signature)
}

func TestVerifyNotification(t *testing.T) {
	provider, err := New("12", map[string]string{"apiBase": "https://pay.example", "apiToken": "secret"})
	require.NoError(t, err)
	body := map[string]any{"trade_id": "trade-1", "order_id": "sub2_1", "amount": 12.34, "actual_amount": "1.7", "token": "0xabc", "block_transaction_id": "tx-1", "status": 2}
	body["signature"] = signMap(body, "secret")
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	notification, err := provider.VerifyNotification(context.Background(), string(raw), nil)
	require.NoError(t, err)
	require.Equal(t, payment.NotificationStatusSuccess, notification.Status)
	require.Equal(t, "sub2_1", notification.OrderID)
	require.Equal(t, "tx-1", notification.Metadata["block_transaction_id"])
	require.Equal(t, "1.7", notification.Metadata["actual_amount"])

	body["signature"] = "invalid"
	raw, err = json.Marshal(body)
	require.NoError(t, err)
	_, err = provider.VerifyNotification(context.Background(), string(raw), nil)
	require.Error(t, err)
}

func TestVerifyNotificationRequiresSuccessFieldsAndIgnoresPending(t *testing.T) {
	provider, err := New("12", map[string]string{"apiBase": "https://pay.example", "apiToken": "secret"})
	require.NoError(t, err)

	pending := map[string]any{"trade_id": "trade-1", "order_id": "sub2_1", "amount": 12.34, "actual_amount": "1.7", "token": "0xabc", "block_transaction_id": "tx-1", "status": 1}
	pending["signature"] = signMap(pending, "secret")
	raw, err := json.Marshal(pending)
	require.NoError(t, err)
	notification, err := provider.VerifyNotification(context.Background(), string(raw), nil)
	require.NoError(t, err)
	require.Nil(t, notification)

	missingTradeID := map[string]any{"order_id": "sub2_1", "amount": 12.34, "actual_amount": "1.7", "token": "0xabc", "block_transaction_id": "tx-1", "status": 2}
	missingTradeID["signature"] = signMap(missingTradeID, "secret")
	raw, err = json.Marshal(missingTradeID)
	require.NoError(t, err)
	_, err = provider.VerifyNotification(context.Background(), string(raw), nil)
	require.ErrorContains(t, err, "missing required payment fields")

	zeroAmount := map[string]any{"trade_id": "trade-1", "order_id": "sub2_1", "amount": 0, "actual_amount": "0", "token": "0xabc", "block_transaction_id": "tx-1", "status": 2}
	zeroAmount["signature"] = signMap(zeroAmount, "secret")
	raw, err = json.Marshal(zeroAmount)
	require.NoError(t, err)
	_, err = provider.VerifyNotification(context.Background(), string(raw), nil)
	require.ErrorContains(t, err, "missing required payment fields")
}

func TestNewRejectsNonCNYFiat(t *testing.T) {
	_, err := New("12", map[string]string{"apiBase": "https://pay.example", "apiToken": "secret", "fiat": "USD"})
	require.ErrorContains(t, err, "fiat must be CNY")
}

func TestExtractWebhookOrderID(t *testing.T) {
	require.Equal(t, "sub2_1", extractWebhookOrderID(`{"order_id":" sub2_1 "}`))
	require.Empty(t, extractWebhookOrderID(`{"trade_id":"trade-1"}`))
	require.Empty(t, extractWebhookOrderID(`not-json`))
}
