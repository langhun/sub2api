package checkinuseraccess

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	activitycontract "github.com/Wei-Shaw/sub2api/internal/custom/modules/activity/contract"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type eligibilityStub struct {
	calls int
	err   error
}

func (s *eligibilityStub) CheckinRewardAllowed(context.Context, int64) error {
	s.calls++
	return s.err
}

func TestEligibilityDeniesDisabledUsersBeforeNextPolicy(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	next := &eligibilityStub{}
	policy := NewModule(db, next).Eligibility

	mock.ExpectQuery(regexp.QuoteMeta("SELECT disabled FROM custom_checkin_user_access WHERE user_id = $1")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"disabled"}).AddRow(true))

	availability, err := policy.CheckinAvailability(context.Background(), 7)
	require.NoError(t, err)
	require.False(t, availability.Allowed)
	require.Equal(t, unavailableReasonAdminDisabled, availability.Reason)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT disabled FROM custom_checkin_user_access WHERE user_id = $1")).
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"disabled"}).AddRow(true))
	err = policy.CheckinRewardAllowed(context.Background(), 7)
	require.Equal(t, "CHECKIN_ACCESS_DISABLED", infraerrors.Reason(err))
	require.Zero(t, next.calls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEligibilityPassesAllowedUsersToExistingPolicy(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	nextErr := errors.New("existing reward policy rejection")
	next := &eligibilityStub{err: nextErr}
	policy := NewModule(db, next).Eligibility

	mock.ExpectQuery(regexp.QuoteMeta("SELECT disabled FROM custom_checkin_user_access WHERE user_id = $1")).
		WithArgs(int64(8)).
		WillReturnRows(sqlmock.NewRows([]string{"disabled"}))
	err = policy.CheckinRewardAllowed(context.Background(), 8)
	require.ErrorIs(t, err, nextErr)
	require.Equal(t, 1, next.calls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestModuleRegistersAdministratorRoutes(t *testing.T) {
	router := gin.New()
	admin := router.Group("/api/v1/custom/admin")
	NewModule(nil, nil).RegisterRoutes(admin)
	routes := map[string]struct{}{}
	for _, route := range router.Routes() {
		routes[route.Method+" "+route.Path] = struct{}{}
	}
	for _, expected := range []string{
		"GET /api/v1/custom/admin/checkin/users/:user_id/access",
		"PUT /api/v1/custom/admin/checkin/users/:user_id/access",
	} {
		_, ok := routes[expected]
		require.Truef(t, ok, "missing %s", expected)
	}
}

func TestUpdateAccessRejectsMissingDisabledFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewModule(nil, nil).RegisterRoutes(router.Group("/api/v1/custom/admin"))
	request := httptest.NewRequest(http.MethodPut, "/api/v1/custom/admin/checkin/users/7/access", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)
	require.Equal(t, http.StatusBadRequest, writer.Code)
}

func TestEligibilityImplementsOptionalStatusPort(t *testing.T) {
	var _ activitycontract.CheckinAvailabilityProvider = NewModule(nil, nil).Eligibility
}
