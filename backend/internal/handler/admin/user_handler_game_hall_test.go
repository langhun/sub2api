package admin

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type captureGameHallUserAdminService struct {
	*stubAdminService
	input *service.UpdateUserInput
}

func (s *captureGameHallUserAdminService) UpdateUser(_ context.Context, id int64, input *service.UpdateUserInput) (*service.User, error) {
	s.input = input
	return &service.User{ID: id, Email: "user@example.com", GameHallDisabled: *input.GameHallDisabled}, nil
}

func TestUserHandlerUpdatePreservesExplicitGameHallEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminService := &captureGameHallUserAdminService{stubAdminService: newStubAdminService()}
	handler := NewUserHandler(adminService, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.PUT("/admin/users/:id", handler.Update)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/admin/users/42", bytes.NewBufferString(`{"game_hall_disabled":false}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotNil(t, adminService.input)
	require.NotNil(t, adminService.input.GameHallDisabled)
	require.False(t, *adminService.input.GameHallDisabled)
}
