package gmpayedge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentUsesGMPayEPayMAPI(t *testing.T) {
	var received url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payments/epay/v1/order/create-transaction/mapi.php", r.URL.Path)
		require.NoError(t, r.ParseForm())
		received = r.Form
		_, _ = w.Write([]byte(`{"code":1,"msg":"success","trade_no":"trade-1","payurl":"https://gmpay.example/checkout/1","qrcode":"https://gmpay.example/checkout/1"}`))
	}))
	defer server.Close()

	provider, err := New("1", map[string]string{
		"pid": "1001", "apiSecret": "secret", "apiBase": server.URL + "/payments/epay/v1/order/create-transaction",
		"notifyUrl": "https://merchant.example/api/v1/payment/webhook/gmpay_edge", "returnUrl": "https://merchant.example/payment/result",
	})
	require.NoError(t, err)
	result, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{OrderID: "sub2_1", Amount: "12.50", Subject: "Recharge", ReturnURL: "https://merchant.example/payment/result"})
	require.NoError(t, err)
	require.Equal(t, "trade-1", result.TradeNo)
	require.Equal(t, "https://gmpay.example/checkout/1", result.PayURL)
	require.Equal(t, "1001", received.Get("pid"))
	require.Empty(t, received.Get("type"))
	require.Equal(t, sign(valuesToStringMap(received), "secret"), received.Get("sign"))
}

func TestQueryOrderUsesSignedEpayQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payments/epay/v1/order/create-transaction/api.php", r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "order", r.URL.Query().Get("act"))
		require.Equal(t, sign(map[string]string{"act": "order", "out_trade_no": "sub2_2", "pid": "1001"}, "secret"), r.URL.Query().Get("sign"))
		_, _ = w.Write([]byte(`{"code":1,"msg":"success","trade_no":"trade-2","out_trade_no":"sub2_2","money":"20.00","status":1,"trade_status":"TRADE_SUCCESS"}`))
	}))
	defer server.Close()
	provider, err := New("1", map[string]string{"pid": "1001", "apiSecret": "secret", "apiBase": server.URL + "/payments/epay/v1/order/create-transaction", "notifyUrl": "https://merchant/notify", "returnUrl": "https://merchant/return"})
	require.NoError(t, err)
	result, err := provider.QueryOrder(context.Background(), "sub2_2")
	require.NoError(t, err)
	require.Equal(t, payment.ProviderStatusPaid, result.Status)
	require.Equal(t, 20.0, result.Amount)
}

func TestVerifyNotificationRequiresGMPaySignatureAndSuccess(t *testing.T) {
	provider, err := New("1", map[string]string{"pid": "1001", "apiSecret": "secret", "apiBase": "https://gmpay.example/payments/epay/v1/order/create-transaction", "notifyUrl": "https://merchant/notify", "returnUrl": "https://merchant/return"})
	require.NoError(t, err)
	params := map[string]string{"pid": "1001", "trade_no": "trade-3", "out_trade_no": "sub2_3", "money": "8.00", "trade_status": "TRADE_SUCCESS", "type": "USDT.TRON"}
	params["sign"] = sign(params, "secret")
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	notification, err := provider.VerifyNotification(context.Background(), values.Encode(), nil)
	require.NoError(t, err)
	require.Equal(t, payment.NotificationStatusSuccess, notification.Status)
	require.Equal(t, "sub2_3", notification.OrderID)

	params["trade_status"] = "WAIT_BUYER_PAY"
	params["sign"] = sign(params, "secret")
	values = url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	notification, err = provider.VerifyNotification(context.Background(), values.Encode(), nil)
	require.NoError(t, err)
	require.Nil(t, notification)
}

func TestExtractWebhookOrderID(t *testing.T) {
	require.Equal(t, "sub2_4", extractWebhookOrderID("trade_no=t&out_trade_no=sub2_4"))
}

func valuesToStringMap(values url.Values) map[string]string {
	result := make(map[string]string, len(values))
	for key := range values {
		result[key] = values.Get(key)
	}
	return result
}
