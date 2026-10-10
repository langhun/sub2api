package xznoauthinvoice

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	if err := c.ShouldBindJSON(&req); err != nil || len(req.OrderNos) == 0 {
		response.BadRequest(c, "at least one order number is required")
		return
	}
	req.CreateTaxPayments = false
	if hasDuplicateStrings(req.OrderNos) {
		response.BadRequest(c, "duplicate order numbers are not allowed")
		return
	}
	req.OrderNos = normalizeStrings(req.OrderNos)
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
	if hasDuplicateStrings(req.OrderNos) {
		response.BadRequest(c, "duplicate order numbers are not allowed")
		return
	}
	req.OrderNos = normalizeStrings(req.OrderNos)
	req.TaxOrderNos = nil
	if len(req.OrderNos) == 0 || strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.TaxpayerID) == "" || strings.TrimSpace(req.RecipientEmail) == "" {
		response.BadRequest(c, "order numbers, title, taxpayer ID and recipient email are required")
		return
	}
	out, err := h.client.Create(c.Request.Context(), req)
	if err != nil {
		h.remoteError(c, err)
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
		ApplicationID int64  `json:"applicationId"`
		TaxOrderNo    string `json:"taxOrderNo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ApplicationID <= 0 || strings.TrimSpace(req.TaxOrderNo) == "" {
		response.BadRequest(c, "application ID and tax order number are required")
		return
	}
	if !h.requireOwned(c, userID, req.ApplicationID) {
		return
	}
	ownsTaxOrder, err := h.store.OwnsTaxOrder(c.Request.Context(), userID, req.ApplicationID, strings.TrimSpace(req.TaxOrderNo))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to verify invoice tax payment ownership")
		return
	}
	if !ownsTaxOrder {
		response.NotFound(c, "invoice tax payment not found")
		return
	}
	paid, err := h.client.TaxStatus(c.Request.Context(), strings.TrimSpace(req.TaxOrderNo))
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
	_ = h.store.UpdateStatus(c.Request.Context(), userID, id, out.Status)
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

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func hasDuplicateStrings(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}
