//go:build unit

package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type receiverHandlerSettingRepo struct{ service.SettingRepository }

func (receiverHandlerSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return map[string]string{service.SettingKeyTransferEnabled: "true"}, nil
}

type receiverHandlerUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r *receiverHandlerUserRepo) ResolveActiveTransferReceiver(context.Context, string, *int64) (*service.User, error) {
	return r.user, nil
}

func newReceiverHandler(user *service.User) *BalanceTransferHandler {
	settings := service.NewSettingService(receiverHandlerSettingRepo{}, &config.Config{})
	transfer := service.NewBalanceTransferService(nil, nil, &receiverHandlerUserRepo{user: user}, settings, nil)
	return NewBalanceTransferHandler(transfer)
}

func TestBalanceTransferHandlerResolveReceiver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := newReceiverHandler(&service.User{ID: 9, Status: service.StatusActive, Username: "alice"})

	t.Run("returns only id and masked display", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/user/transfer/receiver?query=alice", nil)
		c.Set("user_id", int64(1))

		h.ResolveReceiver(c)

		require.Equal(t, http.StatusOK, w.Code)
		require.JSONEq(t, `{"receiver_id":9,"receiver_display":"alice"}`, w.Body.String())
	})

	t.Run("rejects short nonnumeric query", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/user/transfer/receiver?query=a", nil)
		c.Set("user_id", int64(1))

		h.ResolveReceiver(c)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "RECEIVER_QUERY_INVALID")
	})
}

func TestGetUserIDAware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("reads the authenticated subject set by JWT middleware", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user", servermiddleware.AuthSubject{UserID: 42})

		require.Equal(t, int64(42), GetUserIDAware(c))
	})

	t.Run("prefers the authenticated subject over the legacy key", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user", servermiddleware.AuthSubject{UserID: 42})
		c.Set("user_id", int64(7))

		require.Equal(t, int64(42), GetUserIDAware(c))
	})

	t.Run("keeps the legacy key fallback", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Set("user_id", int64(7))

		require.Equal(t, int64(7), GetUserIDAware(c))
	})
}
