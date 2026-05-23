package service

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

const (
	OpsRetryModeClient        = "client"
	OpsRetryModeUpstream      = "upstream"
	OpsRetryModeUpstreamEvent = "upstream_event"
)

const (
	opsRetryStatusUnavailable  = "unavailable"
	opsRetryCaptureBytesLimit  = 64 * 1024
	opsRetryResponsePreviewMax = 8 * 1024
)

var opsRetryRequestHeaderAllowlist = map[string]bool{
	"anthropic-beta":    true,
	"anthropic-version": true,
}

// OpsRetryResult keeps the retry response contract buildable even though the
// underlying retry execution flow has been removed.
type OpsRetryResult struct {
	AttemptID         int64     `json:"attempt_id"`
	Mode              string    `json:"mode"`
	Status            string    `json:"status"`
	PinnedAccountID   *int64    `json:"pinned_account_id,omitempty"`
	UsedAccountID     *int64    `json:"used_account_id,omitempty"`
	HTTPStatusCode    int       `json:"http_status_code"`
	UpstreamRequestID string    `json:"upstream_request_id,omitempty"`
	ResponsePreview   string    `json:"response_preview,omitempty"`
	ResponseTruncated bool      `json:"response_truncated"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	StartedAt         time.Time `json:"started_at"`
	FinishedAt        time.Time `json:"finished_at"`
	DurationMs        int64     `json:"duration_ms"`
}

type limitedResponseWriter struct {
	header      http.Header
	wroteHeader bool

	limit        int
	totalWritten int64
	buf          bytes.Buffer
}

func newLimitedResponseWriter(limit int) *limitedResponseWriter {
	if limit <= 0 {
		limit = 1
	}
	return &limitedResponseWriter{
		header: make(http.Header),
		limit:  limit,
	}
}

func (w *limitedResponseWriter) Header() http.Header {
	return w.header
}

func (w *limitedResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
}

func (w *limitedResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	w.totalWritten += int64(len(p))

	if w.buf.Len() < w.limit {
		remaining := w.limit - w.buf.Len()
		if len(p) > remaining {
			_, _ = w.buf.Write(p[:remaining])
		} else {
			_, _ = w.buf.Write(p)
		}
	}

	// Pretend we wrote everything to avoid upstream/client code treating it as an error.
	return len(p), nil
}

func (w *limitedResponseWriter) Flush() {}

func (w *limitedResponseWriter) bodyBytes() []byte {
	return w.buf.Bytes()
}

func (w *limitedResponseWriter) truncated() bool {
	return w.totalWritten > int64(w.limit)
}

func (s *OpsService) RetryError(ctx context.Context, requestedByUserID int64, errorID int64, mode string, pinnedAccountID *int64) (*OpsRetryResult, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	return nil, opsRetryUnavailableError()
}

func (s *OpsService) RetryUpstreamEvent(ctx context.Context, requestedByUserID int64, errorID int64, idx int) (*OpsRetryResult, error) {
	if err := s.RequireMonitoringEnabled(ctx); err != nil {
		return nil, err
	}
	return nil, opsRetryUnavailableError()
}

func opsRetryUnavailableError() error {
	return infraerrors.ServiceUnavailable(
		"OPS_RETRY_UNAVAILABLE",
		"ops retry feature has been removed and is currently unavailable",
	)
}

func newOpsRetryContext(ctx context.Context, errorLog *OpsErrorLogDetail) (*gin.Context, *limitedResponseWriter) {
	w := newLimitedResponseWriter(opsRetryCaptureBytesLimit)
	c, _ := gin.CreateTestContext(w)

	path := "/"
	if errorLog != nil && strings.TrimSpace(errorLog.RequestPath) != "" {
		path = errorLog.RequestPath
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost"+path, bytes.NewReader(nil))
	req.Header.Set("content-type", "application/json")
	if errorLog != nil && strings.TrimSpace(errorLog.UserAgent) != "" {
		req.Header.Set("user-agent", errorLog.UserAgent)
	}

	if rawHeaders := extractOpsRetryRequestHeaders(errorLog); rawHeaders != "" {
		var stored map[string]string
		if err := json.Unmarshal([]byte(rawHeaders), &stored); err == nil {
			for k, v := range stored {
				key := strings.TrimSpace(k)
				if key == "" || !opsRetryRequestHeaderAllowlist[strings.ToLower(key)] {
					continue
				}
				val := strings.TrimSpace(v)
				if val == "" {
					continue
				}
				req.Header.Set(key, val)
			}
		}
	}

	c.Request = req
	SetOpenAIClientTransport(c, OpenAIClientTransportHTTP)
	return c, w
}

func extractOpsRetryRequestHeaders(errorLog *OpsErrorLogDetail) string {
	if errorLog == nil {
		return ""
	}
	raw, ok := lookupStringField(errorLog, "RequestHeaders")
	if !ok {
		return ""
	}
	return strings.TrimSpace(raw)
}

func lookupStringField(src any, fieldName string) (string, bool) {
	if src == nil {
		return "", false
	}

	value := reflect.ValueOf(src)
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return "", false
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return "", false
	}

	field := value.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String {
		return "", false
	}

	return field.String(), true
}

func extractUpstreamRequestID(c *gin.Context) string {
	if c == nil || c.Writer == nil {
		return ""
	}
	h := c.Writer.Header()
	if h == nil {
		return ""
	}
	for _, key := range []string{"x-request-id", "X-Request-Id", "X-Request-ID"} {
		if v := strings.TrimSpace(h.Get(key)); v != "" {
			return v
		}
	}
	return ""
}

func extractResponsePreview(w *limitedResponseWriter) (preview string, truncated bool) {
	if w == nil {
		return "", false
	}
	b := bytes.TrimSpace(w.bodyBytes())
	if len(b) == 0 {
		return "", w.truncated()
	}
	if len(b) > opsRetryResponsePreviewMax {
		return string(b[:opsRetryResponsePreviewMax]), true
	}
	return string(b), w.truncated()
}
