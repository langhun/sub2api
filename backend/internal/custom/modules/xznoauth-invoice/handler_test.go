package xznoauthinvoice

import (
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
