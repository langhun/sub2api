//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemRepoStubForStats struct {
	redeemRepoStub

	pages []([]RedeemCode)
	total int64
	err   error
	calls []pagination.PaginationParams
}

func (s *redeemRepoStubForStats) ListWithFilters(_ context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
	s.calls = append(s.calls, params)
	if s.err != nil {
		return nil, nil, s.err
	}
	if params.Page <= 0 || params.Page > len(s.pages) {
		return []RedeemCode{}, &pagination.PaginationResult{
			Total:    s.total,
			Page:     params.Page,
			PageSize: params.PageSize,
		}, nil
	}
	return s.pages[params.Page-1], &pagination.PaginationResult{
		Total:    s.total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

func TestAdminService_GetRedeemCodeStats_ComputesExpectedValues(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	repo := &redeemRepoStubForStats{
		total: 7,
		pages: []([]RedeemCode){
			{
				{ID: 1, Type: RedeemTypeBalance, Status: StatusUnused},
				{ID: 2, Type: RedeemTypeConcurrency, Status: StatusUsed, Value: 3},
				{ID: 3, Type: RedeemTypeInvitation, Status: StatusExpired},
				{ID: 4, Type: AdjustmentTypeAdminBalance, Status: StatusUsed, Value: 5},
				{ID: 5, Type: AdjustmentTypeCheckin, Status: StatusUnused, ExpiresAt: &past},
				{ID: 6, Type: AdjustmentTypeAdminConcurrency, Status: StatusUnused, ExpiresAt: &future},
				{ID: 7, Type: RedeemTypeSubscription, Status: StatusUnused, ExpiresAt: &future},
			},
		},
	}
	svc := &adminServiceImpl{redeemCodeRepo: repo}

	stats, err := svc.GetRedeemCodeStats(context.Background())
	require.NoError(t, err)
	require.NotNil(t, stats)
	require.Equal(t, int64(7), stats.TotalCodes)
	require.Equal(t, int64(3), stats.ActiveCodes)
	require.Equal(t, int64(2), stats.UsedCodes)
	require.Equal(t, int64(2), stats.ExpiredCodes)
	require.Equal(t, 8.0, stats.TotalValueDistributed)
	require.Equal(t, int64(3), stats.ByType.Balance)
	require.Equal(t, int64(2), stats.ByType.Concurrency)
	require.Equal(t, int64(1), stats.ByType.Trial)

	require.Len(t, repo.calls, 1)
	require.Equal(t, 1, repo.calls[0].Page)
	require.Equal(t, 1000, repo.calls[0].PageSize)
}

func TestAdminService_GetRedeemCodeStats_PropagatesRepositoryError(t *testing.T) {
	repoErr := errors.New("list failed")
	repo := &redeemRepoStubForStats{err: repoErr}
	svc := &adminServiceImpl{redeemCodeRepo: repo}

	stats, err := svc.GetRedeemCodeStats(context.Background())
	require.ErrorIs(t, err, repoErr)
	require.Nil(t, stats)
}
