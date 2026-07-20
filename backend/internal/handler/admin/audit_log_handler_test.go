package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type auditLogFilterCaptureRepo struct {
	filter *service.AuditLogFilter
}

func (r *auditLogFilterCaptureRepo) BatchInsert(context.Context, []*service.AuditLog) (int64, error) {
	return 0, nil
}

func (r *auditLogFilterCaptureRepo) Insert(context.Context, *service.AuditLog) error { return nil }

func (r *auditLogFilterCaptureRepo) List(_ context.Context, filter *service.AuditLogFilter) (*service.AuditLogList, error) {
	copy := *filter
	r.filter = &copy
	return &service.AuditLogList{Logs: []*service.AuditLog{}, Page: filter.Page, PageSize: filter.PageSize}, nil
}

func (r *auditLogFilterCaptureRepo) GetByID(context.Context, int64) (*service.AuditLog, error) {
	return nil, service.ErrAuditLogNotFound
}

func (r *auditLogFilterCaptureRepo) Count(context.Context) (int64, error) { return 0, nil }

func (r *auditLogFilterCaptureRepo) TruncateAll(context.Context) error { return nil }

func (r *auditLogFilterCaptureRepo) DeleteBefore(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func TestAuditLogHandlerListAcceptsActorAndLegacyEmailFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "actor", url: "/admin/audit-logs?actor=alice", want: "alice"},
		{name: "legacy actor email", url: "/admin/audit-logs?actor_email=legacy%40example.com", want: "legacy@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &auditLogFilterCaptureRepo{}
			handler := NewAuditLogHandler(service.NewAuditLogService(repo, nil), nil)
			router := gin.New()
			router.GET("/admin/audit-logs", handler.List)

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.url, nil))

			require.Equal(t, http.StatusOK, recorder.Code)
			require.NotNil(t, repo.filter)
			require.Equal(t, tt.want, repo.filter.Actor)
		})
	}
}
