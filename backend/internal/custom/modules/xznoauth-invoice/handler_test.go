package xznoauthinvoice

import (
	"errors"
	"strconv"
	"strings"
	"testing"
)

func TestValidateOrderNos(t *testing.T) {
	orders := make([]string, 1500)
	for i := range orders {
		orders[i] = " ORDER-" + strings.Repeat("x", 1) + strconv.Itoa(i) + " "
	}
	got, err := validateOrderNos(orders)
	if err != nil || len(got) != len(orders) || got[0] == "" || strings.HasPrefix(got[0], " ") {
		t.Fatalf("validated order count = %d, err = %v", len(got), err)
	}

	for _, input := range [][]string{{}, {""}, {"ORDER-1", " ORDER-1 "}, {strings.Repeat("x", 129)}} {
		if _, err := validateOrderNos(input); err == nil {
			t.Fatalf("validateOrderNos(%q) should fail", input)
		}
	}
}

func TestInvoiceEligibilityHelpers(t *testing.T) {
	for _, code := range []string{
		"INVOICE_ORDER_NOT_ELIGIBLE",
		"INVOICE_ORDER_ALREADY_CLAIMED",
		"INVOICE_DUPLICATE_ORDER",
		"INVOICE_INVALID_ORDERS",
	} {
		if !isInvoiceEligibilityMiss(&RemoteError{Code: code}) {
			t.Errorf("%s should be treated as an ineligible order", code)
		}
	}
	for _, err := range []error{
		&RemoteError{Code: "INVOICE_ORDER_MERCHANT_MISMATCH"},
		&RemoteError{Code: "IDENTITY_VERIFICATION_REQUIRED"},
		errors.New("network failure"),
	} {
		if isInvoiceEligibilityMiss(err) {
			t.Errorf("%v should not be treated as an ineligible order", err)
		}
	}

	order := Order{OrderNo: "ORDER-1", ExternalNo: "EXTERNAL-1", TransactionID: "TX-1"}
	for _, identifier := range []string{"ORDER-1", "EXTERNAL-1", "TX-1"} {
		if !invoiceOrderMatches(order, identifier) {
			t.Errorf("order identifier %q should match", identifier)
		}
	}
	if invoiceOrderMatches(order, "OTHER") {
		t.Fatal("unrelated order identifier unexpectedly matched")
	}
}

func TestBuyerDetailsValidation(t *testing.T) {
	req := BuyerInput{
		RecipientEmail:   "billing@example.com",
		BuyerPhone:       "+86 (10) 1234-5678",
		BuyerBankAccount: normalizeBankAccount(" 1234-5678 9012 "),
	}
	if req.BuyerBankAccount != "123456789012" {
		t.Fatalf("normalized bank account = %q", req.BuyerBankAccount)
	}
	if err := validateBuyerDetails(req); err != nil {
		t.Fatal(err)
	}

	for name, bad := range map[string]BuyerInput{
		"email":        {RecipientEmail: "not-an-email"},
		"phone":        {RecipientEmail: "billing@example.com", BuyerPhone: "555-ABC"},
		"bank account": {RecipientEmail: "billing@example.com", BuyerBankAccount: "123-456"},
	} {
		t.Run(name, func(t *testing.T) {
			if err := validateBuyerDetails(bad); err == nil {
				t.Fatal("expected buyer detail validation error")
			}
		})
	}
}
