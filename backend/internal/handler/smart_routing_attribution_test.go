package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpsSmartRoutingRecoveredLogPersistsFailedGroup(t *testing.T) {
	setupOpsErrorLogTestQueue(t, 2)
	gin.SetMode(gin.TestMode)
	ops := service.NewOpsService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router := gin.New()
	router.Use(OpsErrorLoggerMiddleware(ops))
	router.POST("/v1/responses", func(c *gin.Context) {
		primary, candidate := int64(2), int64(22)
		key := &service.APIKey{ID: 9, GroupID: &primary}
		c.Set(string(middleware.ContextKeyAPIKey), key)
		setOpsSelectedAccount(c, 11, service.PlatformOpenAI)
		c.Set(service.OpsUpstreamErrorsKey, []*service.OpsUpstreamErrorEvent{{AccountID: 11, GroupID: primary, UpstreamStatusCode: 503, Message: "unavailable"}})
		key.GroupID = &candidate
		setOpsSelectedAccount(c, 12, service.PlatformOpenAI)
		c.JSON(http.StatusOK, gin.H{"status": "completed"})
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/responses", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, int64(1), OpsErrorLogQueueLength())
	job := <-opsErrorLogQueue
	require.EqualValues(t, 2, *job.entry.GroupID)
	require.EqualValues(t, 11, *job.entry.AccountID)
	require.Equal(t, http.StatusOK, job.entry.StatusCode)
	events, err := service.ParseOpsUpstreamErrors(*job.entry.UpstreamErrorsJSON)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.EqualValues(t, 2, events[0].GroupID)
}

func TestOpsSmartRoutingFailureKeepsAttemptGroupAfterKeyChanges(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	primary, candidate := int64(2), int64(21)
	key := &service.APIKey{ID: 9, GroupID: &candidate}
	c.Set(string(middleware.ContextKeyAPIKey), key)
	setOpsSelectedAccount(c, 11, service.PlatformOpenAI)
	key.GroupID = &primary
	setOpsSelectedAccount(c, 12, service.PlatformOpenAI)
	failedID := int64(11)
	entry := &service.OpsInsertErrorLogInput{GroupID: key.GroupID, AccountID: &failedID}
	applyOpsAttemptGroup(c, entry)
	require.Equal(t, candidate, *entry.GroupID)
	require.Equal(t, primary, *key.GroupID, "audit must not mutate the final billing group")
	// An account may be admitted to both groups; the event is older than the
	// current snapshot, so its explicit immutable group takes precedence.
	setOpsSelectedAccount(c, 11, service.PlatformOpenAI)
	entry.UpstreamErrors = []*service.OpsUpstreamErrorEvent{{AccountID: 11, GroupID: candidate}}
	applyOpsAttemptGroup(c, entry)
	require.Equal(t, candidate, *entry.GroupID)
}

func TestOpsSmartRoutingStreamCapturesGroupBeforeRestoringPrimary(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	primary, candidate := int64(2), int64(22)
	key := &service.APIKey{ID: 9, GroupID: &candidate}
	c.Set(string(middleware.ContextKeyAPIKey), key)
	setOpsSelectedAccount(c, 11, service.PlatformOpenAI)
	service.MarkOpsStreamFailure(c, "upstream_error", "stream_read_error", "interrupted", 502)
	key.GroupID = &primary
	streamErr, ok := service.GetOpsStreamError(c)
	require.True(t, ok)
	require.Equal(t, candidate, streamErr.GroupID)
	require.EqualValues(t, 11, streamErr.AccountID)
}
