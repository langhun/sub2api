package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// recordingLoadBalancer captures the order amount the service hands to the
// load balancer when selecting a payment instance.
type recordingLoadBalancer struct {
	payment.LoadBalancer
	gotOrderAmount float64
	calls          int
}

func (lb *recordingLoadBalancer) SelectInstance(_ context.Context, _ string, _ payment.PaymentType, _ payment.Strategy, orderAmount float64) (*payment.InstanceSelection, error) {
	lb.calls++
	lb.gotOrderAmount = orderAmount
	return &payment.InstanceSelection{}, nil
}

// TestSelectCreateOrderInstanceUsesRechargeAmountNotPayAmount pins the limit
// contract: single-transaction min/max must be validated against the recharge
// amount the user entered (before the site's fee surcharge), otherwise an
// exactly-at-limit recharge is rejected because the fee pushes the charged
// amount over the configured cap.
func TestSelectCreateOrderInstanceUsesRechargeAmountNotPayAmount(t *testing.T) {
	lb := &recordingLoadBalancer{}
	s := &PaymentService{loadBalancer: lb}
	req := CreateOrderRequest{PaymentType: payment.TypeWxpay, Amount: 200, OrderType: payment.OrderTypeBalance}
	cfg := &PaymentConfig{RechargeFeeRate: 1.6}

	// CreateOrder passes the 200 recharge amount (fee excluded); the selection
	// path must forward it as-is. If selection ever recomputes the fee-inclusive
	// charge (203.20), an exactly-at-limit 200 recharge gets rejected again.
	if _, err := s.selectCreateOrderInstance(context.Background(), req, cfg, 200); err != nil {
		t.Fatalf("selectCreateOrderInstance returned error: %v", err)
	}
	if lb.calls != 1 {
		t.Fatalf("SelectInstance calls = %d, want 1", lb.calls)
	}
	if lb.gotOrderAmount != 200 {
		t.Fatalf("SelectInstance orderAmount = %v, want the 200 recharge amount (fee excluded)", lb.gotOrderAmount)
	}
}
