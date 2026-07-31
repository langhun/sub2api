package upstreamcost

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type rateMultiplierWriterStub struct {
	ids     []int64
	updates service.AccountBulkUpdate
	updated int64
	err     error
}

func (s *rateMultiplierWriterStub) BulkUpdate(_ context.Context, ids []int64, updates service.AccountBulkUpdate) (int64, error) {
	s.ids = ids
	s.updates = updates
	return s.updated, s.err
}

func TestRateSyncerUsesUpstreamEffectiveMultiplier(t *testing.T) {
	writer := &rateMultiplierWriterStub{updated: 1}
	syncer := NewRateSyncer(writer)
	err := syncer.OnUpstreamBillingProbeSuccess(context.Background(), &service.Account{ID: 7}, &service.UpstreamBillingProbeSnapshot{
		Status: service.UpstreamBillingProbeStatusOK,
		Data: map[string]any{
			"billing_scope":             "token",
			"effective_rate_multiplier": 0.08,
		},
	})

	require.NoError(t, err)
	require.Equal(t, []int64{7}, writer.ids)
	require.NotNil(t, writer.updates.RateMultiplier)
	require.Equal(t, 0.08, *writer.updates.RateMultiplier)
}

func TestRateSyncerDoesNotOverwriteFromInvalidSnapshot(t *testing.T) {
	writer := &rateMultiplierWriterStub{updated: 1}
	err := NewRateSyncer(writer).OnUpstreamBillingProbeSuccess(context.Background(), &service.Account{ID: 7}, &service.UpstreamBillingProbeSnapshot{
		Status: service.UpstreamBillingProbeStatusOK,
		Data: map[string]any{
			"billing_scope": "token",
		},
	})

	require.NoError(t, err)
	require.Empty(t, writer.ids)
}
