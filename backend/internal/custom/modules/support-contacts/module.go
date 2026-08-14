// Package supportcontacts owns customer-service contact cards without coupling
// the feature to upstream settings or user-facing page implementations.
package supportcontacts

import (
	"bytes"
	"database/sql"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

const (
	maxContacts           = 12
	maxImageBytes   int64 = 2 * 1024 * 1024
	maxRequestBytes       = maxImageBytes + 64*1024
	maxImagePixels        = 16 * 1024 * 1024
)

var validChannels = map[string]struct{}{
	"wechat": {}, "qq": {}, "telegram": {}, "email": {}, "link": {},
}

type Module struct{ db *sql.DB }

func NewModule(db *sql.DB) *Module { return &Module{db: db} }

type Contact struct {
	ID          string `json:"id"`
	Channel     string `json:"channel"`
	DisplayName string `json:"display_name"`
	Contact     string `json:"contact"`
	Link        string `json:"link"`
	QRImageURL  string `json:"qr_image_url"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
	HasQRImage  bool   `json:"has_qr_image"`
}

type contactRequest struct {
	Channel     string `json:"channel"`
	DisplayName string `json:"display_name"`
	Contact     string `json:"contact"`
	Link        string `json:"link"`
	QRImageURL  string `json:"qr_image_url"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
}

func (m *Module) RegisterRoutes(admin, public *gin.RouterGroup) {
	if m == nil || admin == nil || public == nil {
		return
	}
	public.GET("/custom/support-contacts", m.listPublic)
	public.GET("/custom/support-contacts/:id/qr", m.serveQR)
	contacts := admin.Group("/support-contacts")
	contacts.Use(middleware.RequestBodyLimit(maxRequestBytes))
	contacts.GET("", m.listAdmin)
	contacts.POST("", m.create)
	contacts.PUT("/:id", m.update)
	contacts.DELETE("/:id", m.delete)
	contacts.POST("/:id/qr", m.uploadQR)
	contacts.DELETE("/:id/qr", m.deleteQR)
}

func (m *Module) listPublic(c *gin.Context) { m.list(c, false) }
func (m *Module) listAdmin(c *gin.Context)  { m.list(c, true) }

func (m *Module) list(c *gin.Context, includeDisabled bool) {
	if m == nil || m.db == nil {
		response.InternalError(c, "Customer support contacts are unavailable")
		return
	}
	query := `SELECT id, channel, display_name, contact_value, contact_link, qr_image_url,
       qr_image_data IS NOT NULL, enabled, sort_order
FROM custom_support_contacts`
	if !includeDisabled {
		query += " WHERE enabled = TRUE"
	}
	query += " ORDER BY sort_order ASC, created_at ASC"
	rows, err := m.db.QueryContext(c.Request.Context(), query)
	if err != nil {
		response.InternalError(c, "Failed to load customer support contacts")
		return
	}
	defer func() { _ = rows.Close() }()
	items := make([]Contact, 0)
	for rows.Next() {
		contact, err := scanContact(rows)
		if err != nil {
			response.InternalError(c, "Failed to load customer support contacts")
			return
		}
		items = append(items, contact)
	}
	if err := rows.Err(); err != nil {
		response.InternalError(c, "Failed to load customer support contacts")
		return
	}
	response.Success(c, items)
}

func (m *Module) create(c *gin.Context) {
	var req contactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid customer support contact")
		return
	}
	if err := validateRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if m == nil || m.db == nil {
		response.InternalError(c, "Customer support contacts are unavailable")
		return
	}
	tx, err := m.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(c.Request.Context(), "LOCK TABLE custom_support_contacts IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	var count int
	if err := tx.QueryRowContext(c.Request.Context(), "SELECT COUNT(*) FROM custom_support_contacts").Scan(&count); err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	if count >= maxContacts {
		response.BadRequest(c, "At most 12 customer support contacts are allowed")
		return
	}
	contact := normalizedContact(uuid.NewString(), req, false)
	_, err = tx.ExecContext(c.Request.Context(), `INSERT INTO custom_support_contacts
    (id, channel, display_name, contact_value, contact_link, qr_image_url, enabled, sort_order, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())`,
		contact.ID, contact.Channel, contact.DisplayName, contact.Contact, contact.Link, contact.QRImageURL, contact.Enabled, contact.SortOrder)
	if err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	if err := tx.Commit(); err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	response.Created(c, contact)
}

func (m *Module) update(c *gin.Context) {
	id, ok := contactID(c)
	if !ok {
		return
	}
	var req contactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid customer support contact")
		return
	}
	if err := validateRequest(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if m == nil || m.db == nil {
		response.InternalError(c, "Customer support contacts are unavailable")
		return
	}
	contact := normalizedContact(id, req, req.QRImageURL == "")
	result, err := m.db.ExecContext(c.Request.Context(), `UPDATE custom_support_contacts
SET channel = $2, display_name = $3, contact_value = $4, contact_link = $5, qr_image_url = $6,
    qr_image_data = CASE WHEN $6 <> '' THEN NULL ELSE qr_image_data END,
    qr_image_mime = CASE WHEN $6 <> '' THEN '' ELSE qr_image_mime END,
    enabled = $7, sort_order = $8, updated_at = NOW()
WHERE id = $1`, contact.ID, contact.Channel, contact.DisplayName, contact.Contact, contact.Link, contact.QRImageURL, contact.Enabled, contact.SortOrder)
	if err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		response.InternalError(c, "Failed to save customer support contact")
		return
	}
	if changed == 0 {
		response.NotFound(c, "Customer support contact not found")
		return
	}
	if contact.QRImageURL == "" {
		contact.HasQRImage, contact.QRImageURL = m.hasUploadedQR(c, id)
	}
	response.Success(c, contact)
}

func (m *Module) delete(c *gin.Context) {
	id, ok := contactID(c)
	if !ok {
		return
	}
	if m == nil || m.db == nil {
		response.InternalError(c, "Customer support contacts are unavailable")
		return
	}
	result, err := m.db.ExecContext(c.Request.Context(), "DELETE FROM custom_support_contacts WHERE id = $1", id)
	if err != nil {
		response.InternalError(c, "Failed to delete customer support contact")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		response.InternalError(c, "Failed to delete customer support contact")
		return
	}
	if changed == 0 {
		response.NotFound(c, "Customer support contact not found")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (m *Module) uploadQR(c *gin.Context) {
	id, ok := contactID(c)
	if !ok {
		return
	}
	if m == nil || m.db == nil {
		response.InternalError(c, "Customer support contacts are unavailable")
		return
	}
	file, err := c.FormFile("image")
	if err != nil {
		response.BadRequest(c, "QR image is required")
		return
	}
	if file.Size <= 0 || file.Size > maxImageBytes {
		response.BadRequest(c, "QR image must be smaller than 2 MiB")
		return
	}
	reader, err := file.Open()
	if err != nil {
		response.BadRequest(c, "Unable to read QR image")
		return
	}
	defer func() { _ = reader.Close() }()
	data, err := io.ReadAll(io.LimitReader(reader, maxImageBytes+1))
	if err != nil || int64(len(data)) > maxImageBytes {
		response.BadRequest(c, "QR image must be smaller than 2 MiB")
		return
	}
	mime, ok := supportedImageMIME(data)
	if !ok {
		response.BadRequest(c, "QR image must be PNG, JPEG, or WebP")
		return
	}
	result, err := m.db.ExecContext(c.Request.Context(), `UPDATE custom_support_contacts
SET qr_image_data = $2, qr_image_mime = $3, qr_image_url = '', updated_at = NOW() WHERE id = $1`, id, data, mime)
	if err != nil {
		response.InternalError(c, "Failed to upload QR image")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		response.InternalError(c, "Failed to upload QR image")
		return
	}
	if changed == 0 {
		response.NotFound(c, "Customer support contact not found")
		return
	}
	response.Success(c, gin.H{"qr_image_url": qrURL(id), "has_qr_image": true})
}

func (m *Module) deleteQR(c *gin.Context) {
	id, ok := contactID(c)
	if !ok {
		return
	}
	if m == nil || m.db == nil {
		response.InternalError(c, "Customer support contacts are unavailable")
		return
	}
	result, err := m.db.ExecContext(c.Request.Context(), `UPDATE custom_support_contacts
SET qr_image_data = NULL, qr_image_mime = '', qr_image_url = '', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		response.InternalError(c, "Failed to delete QR image")
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		response.InternalError(c, "Failed to delete QR image")
		return
	}
	if changed == 0 {
		response.NotFound(c, "Customer support contact not found")
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (m *Module) serveQR(c *gin.Context) {
	id, ok := contactID(c)
	if !ok {
		return
	}
	if m == nil || m.db == nil {
		response.NotFound(c, "QR image not found")
		return
	}
	var data []byte
	var mime string
	err := m.db.QueryRowContext(c.Request.Context(), `SELECT qr_image_data, qr_image_mime
FROM custom_support_contacts WHERE id = $1 AND enabled = TRUE AND qr_image_data IS NOT NULL`, id).Scan(&data, &mime)
	if errors.Is(err, sql.ErrNoRows) {
		response.NotFound(c, "QR image not found")
		return
	}
	if err != nil {
		response.InternalError(c, "Unable to load QR image")
		return
	}
	actualMIME, ok := supportedImageMIME(data)
	if !ok || mime == "" || actualMIME != mime {
		response.NotFound(c, "QR image not found")
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline; filename=customer-support-qr")
	c.Data(http.StatusOK, mime, data)
}

func scanContact(scanner interface{ Scan(...any) error }) (Contact, error) {
	var contact Contact
	var remoteQR string
	if err := scanner.Scan(&contact.ID, &contact.Channel, &contact.DisplayName, &contact.Contact, &contact.Link, &remoteQR, &contact.HasQRImage, &contact.Enabled, &contact.SortOrder); err != nil {
		return Contact{}, err
	}
	if contact.HasQRImage {
		contact.QRImageURL = qrURL(contact.ID)
	} else {
		contact.QRImageURL = remoteQR
	}
	return contact, nil
}

func (m *Module) hasUploadedQR(c *gin.Context, id string) (bool, string) {
	var exists bool
	if err := m.db.QueryRowContext(c.Request.Context(), "SELECT qr_image_data IS NOT NULL FROM custom_support_contacts WHERE id = $1", id).Scan(&exists); err != nil || !exists {
		return false, ""
	}
	return true, qrURL(id)
}

func normalizedContact(id string, req contactRequest, retainUpload bool) Contact {
	contact := Contact{ID: id, Channel: strings.ToLower(strings.TrimSpace(req.Channel)), DisplayName: strings.TrimSpace(req.DisplayName), Contact: strings.TrimSpace(req.Contact), Link: strings.TrimSpace(req.Link), QRImageURL: strings.TrimSpace(req.QRImageURL), Enabled: req.Enabled, SortOrder: req.SortOrder}
	if retainUpload {
		contact.QRImageURL = ""
	}
	return contact
}

func validateRequest(req *contactRequest) error {
	if req == nil {
		return errors.New("customer support contact is required")
	}
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	if _, ok := validChannels[req.Channel]; !ok {
		return errors.New("unsupported customer support channel")
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Contact = strings.TrimSpace(req.Contact)
	req.Link = strings.TrimSpace(req.Link)
	req.QRImageURL = strings.TrimSpace(req.QRImageURL)
	if req.Contact == "" || len(req.Contact) > 200 {
		return errors.New("contact information must be between 1 and 200 characters")
	}
	if len(req.DisplayName) > 80 {
		return errors.New("display name must not exceed 80 characters")
	}
	if len(req.Link) > 500 || (req.Link != "" && !safeContactLink(req.Link)) {
		return errors.New("contact link must use HTTPS, mailto, or tg")
	}
	if len(req.QRImageURL) > 1000 || (req.QRImageURL != "" && !safeHTTPSURL(req.QRImageURL)) {
		return errors.New("QR image URL must use HTTPS")
	}
	if req.SortOrder < -10000 || req.SortOrder > 10000 {
		return errors.New("sort order is out of range")
	}
	return nil
}

func safeHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func safeContactLink(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	if parsed.Scheme == "https" && parsed.Host != "" {
		return true
	}
	if parsed.Scheme == "mailto" && parsed.Opaque != "" {
		return true
	}
	return parsed.Scheme == "tg" && parsed.Opaque != ""
}

func supportedImageMIME(data []byte) (string, bool) {
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > maxImagePixels/config.Height {
		return "", false
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return "", false
	}
	switch format {
	case "png":
		return "image/png", true
	case "jpeg":
		return "image/jpeg", true
	case "webp":
		return "image/webp", true
	default:
		return "", false
	}
}

func contactID(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("id"))
	if _, err := uuid.Parse(id); err != nil {
		response.NotFound(c, "Customer support contact not found")
		return "", false
	}
	return id, true
}

func qrURL(id string) string { return "/api/v1/custom/support-contacts/" + id + "/qr" }
