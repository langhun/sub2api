package xznoauthinvoice

import "time"

type Order struct {
	PlatformOrderID int64  `json:"platformOrderId,omitempty"`
	OrderNo         string `json:"orderNo"`
	ExternalNo      string `json:"externalNo,omitempty"`
	ProductName     string `json:"productName,omitempty"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	PaidAt          string `json:"paidAt,omitempty"`
	VerifiedAt      string `json:"verifiedAt,omitempty"`
	TransactionID   string `json:"transactionId,omitempty"`
}

type Validation struct {
	Orders        []Order `json:"orders"`
	TotalAmount   string  `json:"totalAmount"`
	Currency      string  `json:"currency"`
	TaxAmount     string  `json:"taxAmount,omitempty"`
	TaxPaidAmount string  `json:"taxPaidAmount,omitempty"`
	TaxDueAmount  string  `json:"taxDueAmount,omitempty"`
}

type BuyerInput struct {
	OrderNos         []string `json:"orderNos"`
	NeedPayTax       bool     `json:"needPayTax"`
	TaxOrderNos      []string `json:"taxOrderNos,omitempty"`
	BuyerType        string   `json:"buyerType"`
	Title            string   `json:"title"`
	TaxpayerID       string   `json:"taxpayerId"`
	BuyerAddress     string   `json:"buyerAddress,omitempty"`
	BuyerPhone       string   `json:"buyerPhone,omitempty"`
	BuyerBank        string   `json:"buyerBank,omitempty"`
	BuyerBankAccount string   `json:"buyerBankAccount,omitempty"`
	RecipientEmail   string   `json:"recipientEmail"`
}

type InvoiceApplication struct {
	ID             int64     `json:"id"`
	Status         string    `json:"status"`
	OrderNos       []string  `json:"orderNos,omitempty"`
	BuyerType      string    `json:"buyerType,omitempty"`
	Title          string    `json:"title,omitempty"`
	TaxpayerID     string    `json:"taxpayerId,omitempty"`
	RecipientEmail string    `json:"recipientEmail,omitempty"`
	TotalAmount    string    `json:"totalAmount,omitempty"`
	Currency       string    `json:"currency,omitempty"`
	TaxAmount      string    `json:"taxAmount,omitempty"`
	TaxPaidAmount  string    `json:"taxPaidAmount,omitempty"`
	TaxDueAmount   string    `json:"taxDueAmount,omitempty"`
	ReviewNote     string    `json:"reviewNote,omitempty"`
	HasPDF         *bool     `json:"hasPdf,omitempty"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
	UpdatedAt      time.Time `json:"updatedAt,omitempty"`
}

type TaxPayment struct {
	TaxOrderNo string `json:"taxOrderNo"`
	PayURL     string `json:"payUrl"`
}

type TaxCheckout struct {
	Application   InvoiceApplication    `json:"application"`
	TaxPaidAmount string                `json:"taxPaidAmount"`
	TaxDueAmount  string                `json:"taxDueAmount"`
	TaxPayments   map[string]TaxPayment `json:"taxPayments,omitempty"`
}

type InvoicePage struct {
	Items    []InvoiceApplication `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

type validateRequest struct {
	OrderNos          []string `json:"orderNos"`
	NeedPayTax        bool     `json:"needPayTax"`
	CreateTaxPayments bool     `json:"createTaxPayments"`
}
