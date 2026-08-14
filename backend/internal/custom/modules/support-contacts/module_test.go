package supportcontacts

import (
	"bytes"
	"database/sql/driver"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name string
		req  contactRequest
		ok   bool
	}{
		{name: "wechat", req: contactRequest{Channel: "WeChat", Contact: "support", Link: "https://example.test/contact", QRImageURL: "https://example.test/qr.png"}, ok: true},
		{name: "mailto", req: contactRequest{Channel: "email", Contact: "help@example.test", Link: "mailto:help@example.test"}, ok: true},
		{name: "bad channel", req: contactRequest{Channel: "discord", Contact: "support"}},
		{name: "unsafe link", req: contactRequest{Channel: "link", Contact: "support", Link: "javascript:alert(1)"}},
		{name: "unsafe qr", req: contactRequest{Channel: "wechat", Contact: "support", QRImageURL: "http://example.test/qr.png"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequest(&tt.req)
			if tt.ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestSupportedImageMIME(t *testing.T) {
	pngImage := encodedImage(t, "png")
	jpegImage := encodedImage(t, "jpeg")
	webpImage, err := base64.StdEncoding.DecodeString("UklGRh4AAABXRUJQVlA4TBEAAAAvAAAAAAfQ//73v/+BiOh/AAA=")
	require.NoError(t, err)
	tests := []struct {
		name, want string
		data       []byte
	}{
		{name: "png", want: "image/png", data: pngImage},
		{name: "jpeg", want: "image/jpeg", data: jpegImage},
		{name: "webp", want: "image/webp", data: webpImage},
		{name: "png header only", data: []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}},
		{name: "text", data: []byte("not an image")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := supportedImageMIME(tt.data)
			require.Equal(t, tt.want != "", ok)
			require.Equal(t, tt.want, got)
		})
	}
}

func encodedImage(t *testing.T, format string) []byte {
	t.Helper()
	imageData := image.NewRGBA(image.Rect(0, 0, 1, 1))
	imageData.Set(0, 0, color.RGBA{R: 16, G: 32, B: 48, A: 255})
	var buffer bytes.Buffer
	var err error
	if format == "png" {
		err = png.Encode(&buffer, imageData)
	} else {
		err = jpeg.Encode(&buffer, imageData, nil)
	}
	require.NoError(t, err)
	return buffer.Bytes()
}

func TestPublicListReturnsOnlyEnabledContacts(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	id := uuid.NewString()
	mock.ExpectQuery("SELECT id, channel").
		WillReturnRows(sqlmock.NewRows([]string{"id", "channel", "display_name", "contact_value", "contact_link", "qr_image_url", "has_qr", "enabled", "sort_order"}).
			AddRow(id, "wechat", "Support", "help-id", "", "", false, true, 0))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(db)
	module.RegisterRoutes(router.Group("/api/v1/custom/admin"), router.Group("/api/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/custom/support-contacts", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "help-id")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServeQRAddsSafeHeaders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	id := uuid.NewString()
	png := encodedImage(t, "png")
	mock.ExpectQuery("SELECT qr_image_data").WithArgs(id).WillReturnRows(sqlmock.NewRows([]string{"qr_image_data", "qr_image_mime"}).AddRow(driver.Value(png), "image/png"))

	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(db)
	module.RegisterRoutes(router.Group("/api/v1/custom/admin"), router.Group("/api/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/custom/support-contacts/"+id+"/qr", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "image/png", recorder.Header().Get("Content-Type"))
	require.Equal(t, "nosniff", recorder.Header().Get("X-Content-Type-Options"))
	require.Equal(t, png, recorder.Body.Bytes())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUploadRejectsNonImageBeforeWrite(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	module := NewModule(db)
	module.RegisterRoutes(router.Group("/api/v1/custom/admin"), router.Group("/api/v1"))
	id := uuid.NewString()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("image", "qr.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("not an image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/custom/admin/support-contacts/"+id+"/qr", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}
