package docs

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type createRevisionRequest struct {
	Locale         string `json:"locale"`
	Content        string `json:"content"`
	EditSummary    string `json:"edit_summary"`
	BaseRevisionID *int64 `json:"base_revision_id"`
}

type reviewRevisionRequest struct {
	Decision   string `json:"decision"`
	ReviewNote string `json:"review_note"`
}

func (h *Handler) List(c *gin.Context) {
	userID, ok := currentUser(c)
	if !ok || h == nil || h.service == nil {
		return
	}
	role, _ := servermiddleware.GetUserRoleFromContext(c)
	result, err := h.service.List(c.Request.Context(), c.Param("slug"), normalizeLocale(c.Query("locale")), userID, role == "admin")
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *Handler) Create(c *gin.Context) {
	userID, ok := currentUser(c)
	if !ok || h == nil || h.service == nil {
		return
	}
	var req createRevisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid documentation revision")
		return
	}
	revision, err := h.service.Create(c.Request.Context(), CreateInput{
		Slug: c.Param("slug"), Locale: normalizeLocale(req.Locale), Content: req.Content,
		EditSummary: req.EditSummary, AuthorUserID: userID, BaseRevisionID: req.BaseRevisionID,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, revision)
}

func (h *Handler) Revert(c *gin.Context) {
	userID, ok := currentUser(c)
	if !ok || h == nil || h.service == nil {
		return
	}
	revisionID, err := strconv.ParseInt(c.Param("revision_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid revision")
		return
	}
	revision, err := h.service.Revert(c.Request.Context(), c.Param("slug"), normalizeLocale(c.Query("locale")), revisionID, userID)
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, revision)
}

func (h *Handler) Review(c *gin.Context) {
	userID, ok := currentUser(c)
	if !ok || h == nil || h.service == nil {
		return
	}
	var req reviewRevisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid revision review")
		return
	}
	revisionID, err := strconv.ParseInt(c.Param("revision_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid revision")
		return
	}
	revision, err := h.service.Review(c.Request.Context(), ReviewInput{
		Slug: c.Param("slug"), Locale: normalizeLocale(c.Query("locale")), Decision: req.Decision,
		ReviewNote: req.ReviewNote, RevisionID: revisionID, ReviewerUserID: userID,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Success(c, revision)
}

func currentUser(c *gin.Context) (int64, bool) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "unauthorized")
		return 0, false
	}
	return subject.UserID, true
}

func normalizeLocale(value string) string {
	if value == "" {
		return "zh"
	}
	if value == "en" {
		return "en"
	}
	return value
}

func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrConflict):
		response.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrRevisionNotFound):
		response.NotFound(c, err.Error())
	default:
		response.BadRequest(c, err.Error())
	}
}
