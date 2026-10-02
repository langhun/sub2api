package registrationprotection

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func riskTestRouter(db *sql.DB) *gin.Engine {
	router := gin.New()
	NewModule(db, nil).RegisterRoutes(router.Group("/admin"))
	return router
}

func TestRiskListIncludesLowScoresAndRetainsHistoricalEvidence(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer func() { _ = db.Close() }()
	expired := time.Now().UTC().Add(-48 * time.Hour)
	// The predicate must retain low-score and expired evidence, even when the
	// scoring action is none. Limits use a lookahead row for pagination.
	mock.ExpectQuery(regexp.QuoteMeta("WHERE p.risk_action <> 'none' OR p.risk_score > 0 OR p.risk_signals <> '[]'::jsonb ORDER BY p.updated_at DESC, p.user_id DESC LIMIT $1 OFFSET $2")).
		WithArgs(3, 0).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "risk_score", "risk_action", "risk_signals", "risk_expires_at", "payment_unlocked", "related_accounts"}).
			AddRow(986, "test4@example.com", 2, "none", `["exact_fingerprint_repeat"]`, expired, false, `[{"user_id":741,"email":"other@example.com"}]`).
			AddRow(985, "test3@example.com", 2, "none", `["fast_checkin","fast_checkin"]`, nil, false, `[]`).
			AddRow(741, "other@example.com", 3, "medium", `[]`, expired, false, `[]`))

	rec := httptest.NewRecorder()
	riskTestRouter(db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/registration-risk?limit=2", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	var result struct {
		Items   []riskListItem `json:"items"`
		HasMore bool           `json:"has_more"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.Items, 2)
	require.True(t, result.HasMore)
	require.Equal(t, "observe", result.Items[0].Action)
	require.Equal(t, 2, result.Items[0].Score)
	require.Equal(t, "fingerprint_repeat", result.Items[0].Evidence[0].Detail)
	require.Contains(t, rec.Body.String(), "test4@example.com")
	require.Equal(t, []riskRelatedAccount{{UserID: 741, Email: "other@example.com"}}, result.Items[0].RelatedAccounts)
	require.Equal(t, "observe", result.Items[1].Action)
	require.Equal(t, []string{"fast_checkin"}, result.Items[1].Signals)
	require.Len(t, result.Items[1].Evidence, 1)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRiskListNextPageAndEmptyState(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT p.user_id").WithArgs(101, 100).
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "email", "risk_score", "risk_action", "risk_signals", "risk_expires_at", "payment_unlocked", "related_accounts"}))
	rec := httptest.NewRecorder()
	riskTestRouter(db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/registration-risk?offset=100", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"items":[],"has_more":false}`, rec.Body.String())
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRiskListRejectsInvalidOffset(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer func() { _ = db.Close() }()
	for _, offset := range []string{"-1", "invalid", "99999999999999999999999999"} {
		rec := httptest.NewRecorder()
		riskTestRouter(db).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/registration-risk?offset="+offset, nil))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRiskOverrideRequiresDurableAudit(t *testing.T) {
	for _, tc := range []struct {
		name, path, action string
		rows               int64
		auditErr           bool
		status             int
	}{
		{"lock", "lock", RiskActionManualLock, 1, false, http.StatusOK},
		{"unlock", "unlock", RiskActionManualUnlock, 1, false, http.StatusOK},
		{"missing profile", "lock", RiskActionManualLock, 0, false, http.StatusNotFound},
		{"audit unavailable", "lock", RiskActionManualLock, 1, true, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock := newFingerprintSQLMock(t)
			defer func() { _ = db.Close() }()
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE custom_registration_risk_profiles").WithArgs(int64(986), tc.action, "reviewed duplicate").WillReturnResult(sqlmock.NewResult(0, tc.rows))
			if tc.rows > 0 {
				audit := mock.ExpectExec("INSERT INTO custom_registration_risk_audit").WithArgs(int64(986), tc.action, "reviewed duplicate")
				if tc.auditErr {
					audit.WillReturnError(errors.New("unavailable"))
				} else {
					audit.WillReturnResult(sqlmock.NewResult(1, 1))
				}
			}
			if tc.status == http.StatusOK {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			rec := httptest.NewRecorder()
			riskTestRouter(db).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/registration-risk/986/"+tc.path, strings.NewReader(`{"reason":" reviewed duplicate "}`)))
			require.Equal(t, tc.status, rec.Code)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestRiskOverrideRejectsInvalidReasonsWithoutWrites(t *testing.T) {
	db, mock := newFingerprintSQLMock(t)
	defer func() { _ = db.Close() }()
	for _, body := range []string{`{`, `{"reason":"  "}`, `{"reason":"` + strings.Repeat("a", 501) + `"}`} {
		rec := httptest.NewRecorder()
		riskTestRouter(db).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/admin/registration-risk/986/lock", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
	require.NoError(t, mock.ExpectationsWereMet())
}
