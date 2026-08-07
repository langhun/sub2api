package service

import (
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestPaymentOrderExternalNumberDoesNotExposeInternalID(t *testing.T) {
	order := &dbent.PaymentOrder{ID: 42, OutTradeNo: " sub2_20260807abc123 "}
	require.Equal(t, "sub2_20260807abc123", paymentOrderExternalNumber(order))

	order.OutTradeNo = ""
	require.Empty(t, paymentOrderExternalNumber(order))
}
