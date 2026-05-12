package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupBatchPrivacyRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/batch-set-privacy", handler.BatchSetPrivacy)
	router.POST("/api/v1/admin/accounts/batch-clear-privacy", handler.BatchClearPrivacy)
	return router
}

func TestBatchSetPrivacyCountsSuccessFailedAndSkipped(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 1, Name: "openai-oauth", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive},
		{ID: 2, Name: "anthropic-oauth", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth, Status: service.StatusActive},
	}
	adminSvc.forceOpenAIPrivacyMode = service.PrivacyModeTrainingOff

	router := setupBatchPrivacyRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"account_ids": []int64{1, 2, 3},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-set-privacy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	data := resp["data"].(map[string]any)
	require.Equal(t, float64(3), data["total"])
	require.Equal(t, float64(1), data["success"])
	require.Equal(t, float64(1), data["failed"])
	require.Equal(t, float64(1), data["skipped"])

	errorsPayload := data["errors"].([]any)
	require.Len(t, errorsPayload, 1)
	require.Equal(t, float64(3), errorsPayload[0].(map[string]any)["account_id"])
	require.Equal(t, []int64{1}, adminSvc.forcedPrivacyIDs)
}

func TestBatchSetPrivacyTreatsKnownFailureModesAsFailed(t *testing.T) {
	for _, mode := range []string{service.PrivacyModeFailed, service.PrivacyModeCFBlocked} {
		t.Run(mode, func(t *testing.T) {
			adminSvc := newStubAdminService()
			adminSvc.accounts = []service.Account{
				{ID: 21, Name: "openai-oauth", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive},
			}
			adminSvc.forceOpenAIPrivacyMode = mode

			router := setupBatchPrivacyRouter(adminSvc)

			body, err := json.Marshal(map[string]any{
				"account_ids": []int64{21},
			})
			require.NoError(t, err)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-set-privacy", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)

			var resp map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

			data := resp["data"].(map[string]any)
			require.Equal(t, float64(1), data["total"])
			require.Equal(t, float64(0), data["success"])
			require.Equal(t, float64(1), data["failed"])
			require.Equal(t, float64(0), data["skipped"])

			errorsPayload := data["errors"].([]any)
			require.Len(t, errorsPayload, 1)
			require.Equal(t, float64(21), errorsPayload[0].(map[string]any)["account_id"])
		})
	}
}

func TestBatchClearPrivacyCountsSuccessFailedAndSkipped(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{ID: 11, Name: "openai-oauth", Platform: service.PlatformOpenAI, Type: service.AccountTypeOAuth, Status: service.StatusActive},
		{ID: 12, Name: "anthropic-oauth", Platform: service.PlatformAnthropic, Type: service.AccountTypeOAuth, Status: service.StatusActive},
	}

	router := setupBatchPrivacyRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"account_ids": []int64{11, 12, 13},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/batch-clear-privacy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))

	data := resp["data"].(map[string]any)
	require.Equal(t, float64(3), data["total"])
	require.Equal(t, float64(1), data["success"])
	require.Equal(t, float64(1), data["failed"])
	require.Equal(t, float64(1), data["skipped"])

	errorsPayload := data["errors"].([]any)
	require.Len(t, errorsPayload, 1)
	require.Equal(t, float64(13), errorsPayload[0].(map[string]any)["account_id"])
	require.Equal(t, []int64{11}, adminSvc.clearedPrivacyIDs)
}
