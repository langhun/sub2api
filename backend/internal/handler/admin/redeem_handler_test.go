package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newCreateAndRedeemHandler creates a RedeemHandler with a non-nil (but minimal)
// RedeemService so that CreateAndRedeem's nil guard passes and we can test the
// parameter-validation layer that runs before any service call.
func newCreateAndRedeemHandler() *RedeemHandler {
	return &RedeemHandler{
		adminService:  newStubAdminService(),
		redeemService: &service.RedeemService{}, // non-nil to pass nil guard
	}
}

// postCreateAndRedeemValidation calls CreateAndRedeem and returns the response
// status code. For cases that pass validation and proceed into the service layer,
// a panic may occur (because RedeemService internals are nil); this is expected
// and treated as "validation passed" (returns 0 to indicate panic).
func postCreateAndRedeemValidation(t *testing.T, handler *RedeemHandler, body any) (code int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/admin/redeem-codes/create-and-redeem", bytes.NewReader(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	defer func() {
		if r := recover(); r != nil {
			// Panic means we passed validation and entered service layer (expected for minimal stub).
			code = 0
		}
	}()
	handler.CreateAndRedeem(c)
	return w.Code
}

func TestCreateAndRedeem_TypeDefaultsToBalance(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-default",
		"value":   10.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"omitting type should default to balance and pass validation")
}

func TestCreateAndRedeem_SubscriptionRequiresGroupID(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-no-group",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"validity_days": 30,
	})

	assert.Equal(t, http.StatusBadRequest, code)
}

func TestCreateAndRedeem_SubscriptionRequiresNonZeroValidityDays(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()

	t.Run("zero", func(t *testing.T) {
		code := postCreateAndRedeemValidation(t, h, map[string]any{
			"code":          "test-sub-bad-days-zero",
			"type":          "subscription",
			"value":         29.9,
			"user_id":       1,
			"group_id":      groupID,
			"validity_days": 0,
		})

		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("negative_passes_validation", func(t *testing.T) {
		code := postCreateAndRedeemValidation(t, h, map[string]any{
			"code":          "test-sub-negative-days",
			"type":          "subscription",
			"value":         29.9,
			"user_id":       1,
			"group_id":      groupID,
			"validity_days": -7,
		})

		assert.NotEqual(t, http.StatusBadRequest, code,
			"negative validity_days should pass validation for refund")
	})
}

func TestCreateAndRedeem_SubscriptionValidParamsPassValidation(t *testing.T) {
	groupID := int64(5)
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":          "test-sub-valid",
		"type":          "subscription",
		"value":         29.9,
		"user_id":       1,
		"group_id":      groupID,
		"validity_days": 31,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"valid subscription params should pass validation")
}

func TestCreateAndRedeem_BalanceIgnoresSubscriptionFields(t *testing.T) {
	h := newCreateAndRedeemHandler()
	code := postCreateAndRedeemValidation(t, h, map[string]any{
		"code":    "test-balance-no-extras",
		"type":    "balance",
		"value":   50.0,
		"user_id": 1,
	})

	assert.NotEqual(t, http.StatusBadRequest, code,
		"balance type should not require group_id or validity_days")
}

func TestCreateAndRedeem_InvitationRequiresNewFormat(t *testing.T) {
	h := newCreateAndRedeemHandler()

	t.Run("legacy_format_rejected", func(t *testing.T) {
		code := postCreateAndRedeemValidation(t, h, map[string]any{
			"code":    "INVITE123",
			"type":    "invitation",
			"value":   1,
			"user_id": 1,
		})

		assert.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("new_format_passes_validation", func(t *testing.T) {
		code := postCreateAndRedeemValidation(t, h, map[string]any{
			"code":    "dg-abc123",
			"type":    "invitation",
			"value":   1,
			"user_id": 1,
		})

		assert.NotEqual(t, http.StatusBadRequest, code)
	})
}

func TestResolveRedeemCodeExpiresAt_FromDays(t *testing.T) {
	days := 3
	expiresAt, err := resolveRedeemCodeExpiresAt(nil, &days)
	require.NoError(t, err)
	require.NotNil(t, expiresAt)
	require.WithinDuration(t, time.Now().UTC().AddDate(0, 0, days), *expiresAt, 2*time.Second)
}

func TestResolveRedeemCodeExpiresAt_RejectsPastAbsoluteTime(t *testing.T) {
	past := time.Now().UTC().Add(-time.Minute)
	expiresAt, err := resolveRedeemCodeExpiresAt(&past, nil)
	require.Error(t, err)
	require.Nil(t, expiresAt)
}

func TestResolveRedeemCodeExpiresAt_RejectsNonPositiveDays(t *testing.T) {
	days := 0
	expiresAt, err := resolveRedeemCodeExpiresAt(nil, &days)
	require.Error(t, err)
	require.Nil(t, expiresAt)
}

func TestResolveRedeemCodeExpiresAt_RejectsConflictingInputs(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour)
	days := 3
	expiresAt, err := resolveRedeemCodeExpiresAt(&future, &days)
	require.Error(t, err)
	require.Nil(t, expiresAt)
}

func TestRedeemHandlerGetStats_ReturnsServiceStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.redeemStats = service.RedeemCodeStats{
		TotalCodes:            12,
		ActiveCodes:           4,
		UsedCodes:             5,
		ExpiredCodes:          3,
		TotalValueDistributed: 19.5,
		ByType: service.RedeemCodeStatsByType{
			Balance:     7,
			Concurrency: 2,
			Trial:       1,
		},
	}
	handler := NewRedeemHandler(adminSvc, nil)

	router := gin.New()
	router.GET("/api/v1/admin/redeem-codes/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(12), data["total_codes"])
	require.Equal(t, float64(4), data["active_codes"])
	require.Equal(t, float64(5), data["used_codes"])
	require.Equal(t, float64(3), data["expired_codes"])
	require.Equal(t, 19.5, data["total_value_distributed"])

	byType, ok := data["by_type"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, float64(7), byType["balance"])
	require.Equal(t, float64(2), byType["concurrency"])
	require.Equal(t, float64(1), byType["trial"])
}

func TestRedeemHandlerGetStats_PropagatesServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.redeemStatsErr = errors.New("stats failed")
	handler := NewRedeemHandler(adminSvc, nil)

	router := gin.New()
	router.GET("/api/v1/admin/redeem-codes/stats", handler.GetStats)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/redeem-codes/stats", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
