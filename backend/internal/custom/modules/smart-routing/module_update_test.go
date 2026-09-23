package smartrouting

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type routingUpdateKeyRepo struct {
	service.APIKeyRepository
}

func (*routingUpdateKeyRepo) GetByID(_ context.Context, id int64) (*service.APIKey, error) {
	return &service.APIKey{ID: id, UserID: 42}, nil
}

func TestUpdateClassifiesValidationAndDatabaseErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
		detail string
	}{
		{"invalid weights", `{"price_weight":0.2,"speed_weight":0.6,"success_weight":0.2}`, http.StatusBadRequest, "weights must"},
		{"invalid ttl", `{"sticky_ttl_seconds":86401}`, http.StatusBadRequest, "sticky_ttl_seconds out of range"},
		{"invalid exploration", `{"exploration_rate":0.02}`, http.StatusBadRequest, "exploration_rate must"},
		{"invalid version", `{"version":-1}`, http.StatusBadRequest, "version must"},
		{"invalid mode", `{"mode":"unknown"}`, http.StatusBadRequest, "mode must"},
		{"invalid strategy", `{"strategy":"unknown"}`, http.StatusBadRequest, "invalid strategy"},
		{"database unavailable", `{}`, http.StatusInternalServerError, "database unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = db.Close() }()
			if tc.status == http.StatusInternalServerError {
				mock.ExpectBegin().WillReturnError(errors.New("database unavailable"))
			}
			m := &Module{
				repo:    NewRepository(db),
				apiKeys: service.NewAPIKeyService(&routingUpdateKeyRepo{}, nil, nil, nil, nil, nil, nil),
			}
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
			c.Params = gin.Params{{Key: "id", Value: "1"}}
			c.Request = httptest.NewRequest(http.MethodPut, "/keys/1/routing", strings.NewReader(tc.body))
			c.Request.Header.Set("Content-Type", "application/json")
			m.update(c)
			if response.Code != tc.status || !strings.Contains(response.Body.String(), tc.detail) {
				t.Fatalf("response = %d %s; want %d containing %q", response.Code, response.Body.String(), tc.status, tc.detail)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
