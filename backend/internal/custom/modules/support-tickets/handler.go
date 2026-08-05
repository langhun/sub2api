package supporttickets

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

const maxTicketPageSize = 100

type UserHandler struct{ service *Service }

func NewUserHandler(service *Service) *UserHandler { return &UserHandler{service: service} }

type AdminHandler struct{ service *Service }

func NewAdminHandler(service *Service) *AdminHandler { return &AdminHandler{service: service} }

func (h *UserHandler) List(c *gin.Context) {
	if !h.available(c) {
		return
	}
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	page, size := ticketPagination(c)
	items, total, err := h.service.ListUser(c.Request.Context(), userID, page, size)
	h.respondList(c, items, total, page, size, err)
}
func (h *UserHandler) Get(c *gin.Context) {
	if !h.available(c) {
		return
	}
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.DetailUser(c.Request.Context(), userID, c.Param("public_id"))
	respondTicket(c, detail, err)
}

type createTicketRequest struct {
	Category string `json:"category"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
}

func (h *UserHandler) Create(c *gin.Context) {
	if !h.available(c) {
		return
	}
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	var req createTicketRequest
	if !bindTicketJSON(c, &req) {
		return
	}
	ticket, err := h.service.Create(c.Request.Context(), CreateInput{UserID: userID, Category: req.Category, Subject: req.Subject, Body: req.Body})
	if err != nil {
		respondTicketError(c, err)
		return
	}
	response.Created(c, ticket)
}

type replyRequest struct {
	Body string `json:"body"`
}

func (h *UserHandler) Reply(c *gin.Context) {
	if !h.available(c) {
		return
	}
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	var req replyRequest
	if !bindTicketJSON(c, &req) {
		return
	}
	ticket, err := h.service.ReplyUser(c.Request.Context(), ReplyInput{UserID: userID, PublicID: c.Param("public_id"), Body: req.Body})
	respondTicket(c, ticket, err)
}
func (h *UserHandler) Reopen(c *gin.Context) {
	if !h.available(c) {
		return
	}
	userID, ok := currentUser(c)
	if !ok {
		return
	}
	ticket, err := h.service.Reopen(c.Request.Context(), c.Param("public_id"), userID)
	respondTicket(c, ticket, err)
}

func (h *AdminHandler) List(c *gin.Context) {
	if !h.available(c) {
		return
	}
	page, size := ticketPagination(c)
	filter := ListFilter{Page: page, PageSize: size, Status: c.Query("status"), Priority: c.Query("priority")}
	if raw := c.Query("user_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			response.BadRequest(c, "invalid user_id")
			return
		}
		filter.UserID = &id
	}
	items, total, err := h.service.ListAdmin(c.Request.Context(), filter)
	h.respondList(c, items, total, page, size, err)
}
func (h *AdminHandler) Get(c *gin.Context) {
	if !h.available(c) {
		return
	}
	detail, err := h.service.DetailAdmin(c.Request.Context(), c.Param("public_id"))
	respondTicket(c, detail, err)
}

type adminReplyRequest struct {
	Body     string `json:"body"`
	Internal bool   `json:"internal"`
}

type adminCreateTicketRequest struct {
	UserID   int64  `json:"user_id"`
	Category string `json:"category"`
	Subject  string `json:"subject"`
	Body     string `json:"body"`
}

func (h *AdminHandler) Create(c *gin.Context) {
	if !h.available(c) {
		return
	}
	adminID, ok := currentUser(c)
	if !ok {
		return
	}
	var req adminCreateTicketRequest
	if !bindTicketJSON(c, &req) {
		return
	}
	ticket, err := h.service.CreateAdmin(c.Request.Context(), AdminCreateInput{
		AdminID:  adminID,
		UserID:   req.UserID,
		Category: req.Category,
		Subject:  req.Subject,
		Body:     req.Body,
	})
	if err != nil {
		respondTicketError(c, err)
		return
	}
	response.Created(c, ticket)
}

func (h *AdminHandler) Reply(c *gin.Context) {
	if !h.available(c) {
		return
	}
	adminID, ok := currentUser(c)
	if !ok {
		return
	}
	var req adminReplyRequest
	if !bindTicketJSON(c, &req) {
		return
	}
	ticket, err := h.service.ReplyAdmin(c.Request.Context(), ReplyInput{UserID: adminID, PublicID: c.Param("public_id"), Body: req.Body, Internal: req.Internal})
	respondTicket(c, ticket, err)
}

type updateTicketRequest struct {
	Status   string `json:"status"`
	Priority string `json:"priority"`
}

func (h *AdminHandler) Update(c *gin.Context) {
	if !h.available(c) {
		return
	}
	adminID, ok := currentUser(c)
	if !ok {
		return
	}
	var req updateTicketRequest
	if !bindTicketJSON(c, &req) {
		return
	}
	if req.Status != "" && req.Priority != "" {
		response.BadRequest(c, "update one field at a time")
		return
	}
	ticket, err := h.service.Update(c.Request.Context(), UpdateInput{UserID: adminID, PublicID: c.Param("public_id"), Status: req.Status, Priority: req.Priority})
	respondTicket(c, ticket, err)
}

func (h *UserHandler) available(c *gin.Context) bool {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "support tickets are unavailable")
		return false
	}
	return true
}
func (h *AdminHandler) available(c *gin.Context) bool {
	if h == nil || h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "support tickets are unavailable")
		return false
	}
	return true
}
func (h *UserHandler) respondList(c *gin.Context, items []Ticket, total int64, page, size int, err error) {
	if err != nil {
		respondTicketError(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}
func (h *AdminHandler) respondList(c *gin.Context, items []Ticket, total int64, page, size int, err error) {
	if err != nil {
		respondTicketError(c, err)
		return
	}
	response.Paginated(c, items, total, page, size)
}
func currentUser(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return 0, false
	}
	return subject.UserID, true
}

func bindTicketJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.Error(c, http.StatusRequestEntityTooLarge, "ticket request body is too large")
		} else {
			response.BadRequest(c, "invalid request")
		}
		return false
	}
	return true
}
func ticketPagination(c *gin.Context) (int, int) {
	page := 1
	size := 20
	if raw := c.Query("page"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			page = value
		}
	}
	if raw := c.Query("page_size"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 && value <= maxTicketPageSize {
			size = value
		}
	}
	return page, size
}
func respondTicket(c *gin.Context, data any, err error) {
	if err != nil {
		respondTicketError(c, err)
		return
	}
	response.Success(c, data)
}
func respondTicketError(c *gin.Context, err error) {
	if errors.Is(err, ErrNotFound) {
		response.NotFound(c, "support ticket not found")
		return
	}
	if errors.Is(err, ErrTicketInactive) || errors.Is(err, ErrInvalidTransition) || errors.Is(err, ErrTicketMessageLimit) || errors.Is(err, ErrTicketLimit) || errors.Is(err, ErrTargetUserUnavailable) {
		response.BadRequest(c, err.Error())
		return
	}
	message := err.Error()
	if message == "support tickets are disabled" {
		response.Forbidden(c, message)
		return
	}
	if stringsHasPrefix(message, "invalid ") || stringsHasPrefix(message, "subject ") || stringsHasPrefix(message, "message ") || message == "status or priority is required" {
		response.BadRequest(c, message)
		return
	}
	slog.Error("support ticket request failed", "error", err)
	response.Error(c, http.StatusInternalServerError, "support ticket request failed")
}
func stringsHasPrefix(value, prefix string) bool {
	return len(value) >= len(prefix) && value[:len(prefix)] == prefix
}
