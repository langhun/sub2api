package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemStatsRepoStub struct {
	listWithFiltersFn func(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error)
}

func (s *redeemStatsRepoStub) Create(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Create call")
}

func (s *redeemStatsRepoStub) CreateBatch(ctx context.Context, codes []RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (s *redeemStatsRepoStub) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	panic("unexpected GetByID call")
}

func (s *redeemStatsRepoStub) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	panic("unexpected GetByCode call")
}

func (s *redeemStatsRepoStub) Update(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Update call")
}

func (s *redeemStatsRepoStub) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (s *redeemStatsRepoStub) Use(ctx context.Context, id, userID int64) error {
	panic("unexpected Use call")
}

func (s *redeemStatsRepoStub) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (s *redeemStatsRepoStub) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
	if s.listWithFiltersFn == nil {
		panic("unexpected ListWithFilters call")
	}
	return s.listWithFiltersFn(ctx, params, codeType, status, search)
}

func (s *redeemStatsRepoStub) ListByUser(ctx context.Context, userID int64, limit int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (s *redeemStatsRepoStub) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (s *redeemStatsRepoStub) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

func TestRedeemService_GetStats_AggregatesFromRepository(t *testing.T) {
	now := time.Now()
	future := now.Add(2 * time.Hour)
	past := now.Add(-2 * time.Hour)
	repo := &redeemStatsRepoStub{
		listWithFiltersFn: func(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
			require.Equal(t, 1, params.Page)
			require.Equal(t, 1000, params.PageSize)
			require.Equal(t, "id", params.SortBy)
			require.Equal(t, pagination.SortOrderAsc, params.SortOrder)
			require.Empty(t, codeType)
			require.Empty(t, status)
			require.Empty(t, search)

			return []RedeemCode{
					{ID: 1, Status: StatusUnused, ExpiresAt: nil},
					{ID: 2, Status: StatusUnused, ExpiresAt: &future},
					{ID: 3, Status: StatusUnused, ExpiresAt: &past},
					{ID: 4, Status: StatusUsed, Value: 10},
					{ID: 5, Status: StatusUsed, Value: 2.5},
				}, &pagination.PaginationResult{
					Page:     1,
					PageSize: 1000,
					Total:    5,
					Pages:    1,
				}, nil
		},
	}

	svc := NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil, nil)
	stats, err := svc.GetStats(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 5, stats["total_codes"])
	require.EqualValues(t, 2, stats["unused_codes"])
	require.EqualValues(t, 2, stats["used_codes"])
	require.Equal(t, 12.5, stats["total_value"])
}

func TestRedeemService_GetStats_RepositoryError(t *testing.T) {
	repoErr := errors.New("db unavailable")
	repo := &redeemStatsRepoStub{
		listWithFiltersFn: func(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
			return nil, nil, repoErr
		},
	}

	svc := NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil, nil)
	stats, err := svc.GetStats(context.Background())
	require.Error(t, err)
	require.Nil(t, stats)
	require.Contains(t, err.Error(), "list redeem codes for stats")
	require.ErrorIs(t, err, repoErr)
}

func TestRedeemService_GetStats_TotalFallbackUsesScannedCount(t *testing.T) {
	repo := &redeemStatsRepoStub{
		listWithFiltersFn: func(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
			require.Equal(t, 1, params.Page)
			return []RedeemCode{
				{ID: 1, Status: StatusUnused},
				{ID: 2, Status: StatusExpired},
				{ID: 3, Status: StatusUsed, Value: 3},
			}, nil, nil
		},
	}

	svc := NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil, nil)
	stats, err := svc.GetStats(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 3, stats["total_codes"])
	require.EqualValues(t, 1, stats["unused_codes"])
	require.EqualValues(t, 1, stats["used_codes"])
	require.Equal(t, 3.0, stats["total_value"])
}
