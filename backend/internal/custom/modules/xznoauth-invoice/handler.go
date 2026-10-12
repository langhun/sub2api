package xznoauthinvoice

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	client *Client
	store  *Repository
}

func NewHandler(client *Client, store *Repository) *Handler {
	return &Handler{client: client, store: store}
}

func (h *Handler) currentUser(c *gin.Context) (int64, bool) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return 0, false
	}
	return subject.UserID, true
}

func (h *Handler) unavailable(c *gin.Context) bool {
	if h == nil || h.client == nil || h.store == nil || !h.client.Enabled() {
		response.Error(c, http.StatusServiceUnavailable, "invoice service is not configured")
		return true
	}
	return false
}

func (h *Handler) Validate(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	var req validateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "at least one order number is required")
		return
	}
	req.CreateTaxPayments = false
	orderNos, err := validateOrderNos(req.OrderNos)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.OrderNos = orderNos
	out, err := h.client.Validate(c.Request.Context(), req)
	if err != nil {
		h.remoteError(c, err)
		return
	}
	response.Success(c, out)
}

func (h *Handler) Create(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	userID, ok := h.currentUser(c)
	if !ok {
		return
	}
	var req BuyerInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid invoice application")
		return
	}
	orderNos, err := validateOrderNos(req.OrderNos)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.OrderNos = orderNos
	req.TaxOrderNos = nil
	req.Title = strings.TrimSpace(req.Title)
	req.TaxpayerID = strings.TrimSpace(req.TaxpayerID)
	req.RecipientEmail = strings.TrimSpace(req.RecipientEmail)
	req.BuyerAddress = strings.TrimSpace(req.BuyerAddress)
	req.BuyerPhone = strings.TrimSpace(req.BuyerPhone)
	req.BuyerBank = strings.TrimSpace(req.BuyerBank)
	req.BuyerBankAccount = normalizeBankAccount(req.BuyerBankAccount)
	if req.BuyerType != "individual" && req.BuyerType != "company" {
		response.BadRequest(c, "buyer type must be individual or company")
		return
	}
	if req.Title == "" || req.TaxpayerID == "" || req.RecipientEmail == "" {
		response.BadRequest(c, "order numbers, title, taxpayer ID and recipient email are required")
		return
	}
	if err := validateBuyerDetails(req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	out, err := h.client.Create(c.Request.Context(), req)
	if err != nil {
		h.remoteError(c, err)
		return
	}
	if out.ID <= 0 || (out.Status != "pending" && out.Status != "awaiting_tax_payment") {
		response.Error(c, http.StatusBadGateway, "invoice service returned an invalid application")
		return
	}
	if err := h.store.SaveOwnership(c.Request.Context(), userID, out.ID, out.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to save invoice ownership")
		return
	}
	response.Created(c, out)
}

func (h *Handler) List(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	userID, ok := h.currentUser(c)
	if !ok {
		return
	}
	page, size := response.ParsePagination(c)
	remoteIDs, localTotal, err := h.store.OwnedPage(c.Request.Context(), userID, page, size)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to load invoice ownership")
		return
	}
	itemsByID := make(map[int64]InvoiceApplication, len(remoteIDs))
	for remotePage := 1; len(itemsByID) < len(remoteIDs); remotePage++ {
		out, err := h.client.List(c.Request.Context(), remotePage, 100)
		if err != nil {
			h.remoteError(c, err)
			return
		}
		for _, item := range out.Items {
			for _, id := range remoteIDs {
				if item.ID == id {
					itemsByID[id] = item
					break
				}
			}
		}
		if len(out.Items) == 0 || int64(remotePage*100) >= out.Total {
			break
		}
	}
	items := make([]InvoiceApplication, 0, len(remoteIDs))
	for _, id := range remoteIDs {
		if item, ok := itemsByID[id]; ok {
			items = append(items, item)
		}
	}
	response.Paginated(c, items, localTotal, page, size)
}

func (h *Handler) Checkout(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	userID, ok := h.currentUser(c)
	if !ok {
		return
	}
	id, ok := h.applicationID(c)
	if !ok {
		return
	}
	if !h.requireOwned(c, userID, id) {
		return
	}
	out, err := h.client.Checkout(c.Request.Context(), id)
	if err != nil {
		h.remoteError(c, err)
		return
	}
	if out.Application.ID != id {
		response.Error(c, http.StatusBadGateway, "invoice service returned a mismatched application")
		return
	}
	if err := h.store.SaveOwnership(c.Request.Context(), userID, out.Application.ID, out.Application.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update invoice ownership")
		return
	}
	if err := h.store.SaveTaxOrders(c.Request.Context(), userID, id, out.TaxPayments); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to save invoice tax payment ownership")
		return
	}
	response.Success(c, out)
}

func (h *Handler) TaxStatus(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	userID, ok := h.currentUser(c)
	if !ok {
		return
	}
	var req struct {
		ApplicationID int64  `json:"applicationId,omitempty"` // backward-compatible facade field
		TaxOrderNo    string `json:"taxOrderNo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.TaxOrderNo) == "" {
		response.BadRequest(c, "tax order number is required")
		return
	}
	taxOrderNo := strings.TrimSpace(req.TaxOrderNo)
	applicationID := req.ApplicationID
	var err error
	var ownsTaxOrder bool
	if applicationID > 0 {
		if !h.requireOwned(c, userID, applicationID) {
			return
		}
		ownsTaxOrder, err = h.store.OwnsTaxOrder(c.Request.Context(), userID, applicationID, taxOrderNo)
	} else {
		applicationID, ownsTaxOrder, err = h.store.OwnedTaxOrderApplication(c.Request.Context(), userID, taxOrderNo)
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to verify invoice tax payment ownership")
		return
	}
	if !ownsTaxOrder {
		response.NotFound(c, "invoice tax payment not found")
		return
	}
	paid, err := h.client.TaxStatus(c.Request.Context(), taxOrderNo)
	if err != nil {
		h.remoteError(c, err)
		return
	}
	response.Success(c, gin.H{"paid": paid})
}

func (h *Handler) Cancel(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	userID, ok := h.currentUser(c)
	if !ok {
		return
	}
	id, ok := h.applicationID(c)
	if !ok || !h.requireOwned(c, userID, id) {
		return
	}
	out, err := h.client.Cancel(c.Request.Context(), id)
	if err != nil {
		h.remoteError(c, err)
		return
	}
	if err := h.store.UpdateStatus(c.Request.Context(), userID, id, out.Status); err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to update invoice ownership")
		return
	}
	response.Success(c, out)
}

func (h *Handler) PDF(c *gin.Context) {
	if h.unavailable(c) {
		return
	}
	userID, ok := h.currentUser(c)
	if !ok {
		return
	}
	id, ok := h.applicationID(c)
	if !ok || !h.requireOwned(c, userID, id) {
		return
	}
	data, err := h.client.PDF(c.Request.Context(), id)
	if err != nil {
		h.remoteError(c, err)
		return
	}
	c.Header("Content-Disposition", "inline; filename=invoice-"+strconv.FormatInt(id, 10)+".pdf")
	c.Data(http.StatusOK, "application/pdf", data)
}

func (h *Handler) applicationID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.NotFound(c, "invoice not found")
		return 0, false
	}
	return id, true
}

func (h *Handler) requireOwned(c *gin.Context, userID, remoteID int64) bool {
	owned, err := h.store.Owned(c.Request.Context(), userID, remoteID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to verify invoice ownership")
		return false
	}
	if !owned {
		response.NotFound(c, "invoice not found")
		return false
	}
	return true
}

func (h *Handler) remoteError(c *gin.Context, err error) {
	var remote *RemoteError
	if errors.As(err, &remote) {
		log.Printf("[xznoauth-invoice] upstream rejected request status=%d code=%q request_id=%q", remote.Status, remote.Code, remote.RequestID)
		meta := map[string]string{}
		if remote.RequestID != "" {
			meta["request_id"] = remote.RequestID
		}
		status := remote.Status
		if status < 400 {
			status = http.StatusBadGateway
		}
		response.ErrorWithDetails(c, status, safeRemoteMessage(remote.Code), remote.Code, meta)
		return
	}
	response.Error(c, http.StatusBadGateway, "invoice service request failed")
}

func safeRemoteMessage(code string) string {
	switch code {
	case "IDENTITY_VERIFICATION_REQUIRED":
		return "The invoice application operator must complete real-name verification"
	case "INVOICE_ORDER_NOT_ELIGIBLE", "INVOICE_ORDER_ALREADY_CLAIMED", "INVOICE_DUPLICATE_ORDER", "INVOICE_INVALID_ORDERS":
		return "One or more orders cannot be used for this invoice application"
	case "INVOICE_TAX_PAYMENT_NOT_ELIGIBLE", "INVOICE_TAX_PAYMENT_EXCEEDS_REQUIRED", "INVOICE_TAX_ORDER_ALREADY_CLAIMED":
		return "The invoice tax payment cannot be applied to this invoice"
	case "INVOICE_CANNOT_CANCEL":
		return "This invoice application can no longer be canceled"
	case "NOT_FOUND":
		return "Invoice application not found"
	case "OAUTH_INVALID_CLIENT", "AUTH_INVALID_TOKEN", "AUTH_INSUFFICIENT_SCOPE", "AUTH_FORBIDDEN":
		return "Invoice service credentials or permissions are not configured correctly"
	default:
		return "Invoice service rejected the request"
	}
}

func validateOrderNos(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, errors.New("at least one order number is required")
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, errors.New("order numbers cannot be empty")
		}
		if utf8.RuneCountInString(value) > 128 {
			return nil, errors.New("order numbers cannot exceed 128 characters")
		}
		if _, exists := seen[value]; exists {
			return nil, errors.New("duplicate order numbers are not allowed")
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out, nil
}

func normalizeBankAccount(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
}

func validateBuyerDetails(req BuyerInput) error {
	if _, err := mail.ParseAddress(req.RecipientEmail); err != nil || strings.ContainsAny(req.RecipientEmail, "<> \t\r\n") {
		return errors.New("recipient email is invalid")
	}
	if req.BuyerPhone != "" {
		digits := 0
		for _, r := range req.BuyerPhone {
			if r >= '0' && r <= '9' {
				digits++
				continue
			}
			if r != '+' && r != '(' && r != ')' && r != '-' && !unicode.IsSpace(r) {
				return errors.New("buyer phone is invalid")
			}
		}
		if digits < 6 || utf8.RuneCountInString(req.BuyerPhone) > 32 {
			return errors.New("buyer phone is invalid")
		}
	}
	if account := req.BuyerBankAccount; account != "" {
		if len(account) < 8 || len(account) > 32 {
			return errors.New("buyer bank account must contain 8 to 32 digits")
		}
		for _, r := range account {
			if r < '0' || r > '9' {
				return errors.New("buyer bank account must contain 8 to 32 digits")
			}
		}
	}
	return nil
}
