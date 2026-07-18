package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type redeemGenerateRepo struct {
	redeemRejectRepo
	created []RedeemCode
}

type redeemGenerateSettingRepo struct {
	values map[string]string
}

func (r *redeemGenerateSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, ErrSettingNotFound
}

func (r *redeemGenerateSettingRepo) GetValue(context.Context, string) (string, error) {
	return "", ErrSettingNotFound
}

func (r *redeemGenerateSettingRepo) Set(context.Context, string, string) error { return nil }

func (r *redeemGenerateSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return r.values, nil
}

func (r *redeemGenerateSettingRepo) SetMultiple(context.Context, map[string]string) error { return nil }

func (r *redeemGenerateSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}

func (r *redeemGenerateSettingRepo) Delete(context.Context, string) error { return nil }

func (r *redeemGenerateRepo) CreateBatch(_ context.Context, codes []RedeemCode) error {
	r.created = append(r.created, codes...)
	return nil
}

func TestRedeemGenerateCodesUsesConfiguredFormat(t *testing.T) {
	repo := &redeemGenerateRepo{}
	settings := NewSettingService(&redeemGenerateSettingRepo{values: map[string]string{
		SettingKeyCodeFormatBalance: `{"prefix":"BAL","character_set":"numeric","separator":"-","group_length":2,"group_count":2}`,
	}}, nil)
	service := NewRedeemService(repo, nil, nil, nil, nil, nil, nil, nil, settings)

	_, err := service.GenerateCodes(context.Background(), GenerateCodesRequest{Count: 1, Type: RedeemTypeBalance, Value: 5})

	require.NoError(t, err)
	require.Len(t, repo.created, 1)
	require.Regexp(t, `^BAL-[0-9]{2}-[0-9]{2}$`, repo.created[0].Code)
}

type redeemRejectRepo struct {
	code      RedeemCode
	useCalled bool
}

func (r *redeemRejectRepo) Create(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Create call")
}

func (r *redeemRejectRepo) CreateBatch(ctx context.Context, codes []RedeemCode) error {
	panic("unexpected CreateBatch call")
}

func (r *redeemRejectRepo) GetByID(ctx context.Context, id int64) (*RedeemCode, error) {
	if r.code.ID != id {
		return nil, ErrRedeemCodeNotFound
	}
	clone := r.code
	return &clone, nil
}

func (r *redeemRejectRepo) GetByCode(ctx context.Context, code string) (*RedeemCode, error) {
	if r.code.Code != code {
		return nil, ErrRedeemCodeNotFound
	}
	clone := r.code
	return &clone, nil
}

func (r *redeemRejectRepo) Update(ctx context.Context, code *RedeemCode) error {
	panic("unexpected Update call")
}

func (r *redeemRejectRepo) BatchUpdate(ctx context.Context, ids []int64, fields RedeemCodeBatchUpdateFields) (int64, error) {
	panic("unexpected BatchUpdate call")
}

func (r *redeemRejectRepo) Delete(ctx context.Context, id int64) error {
	panic("unexpected Delete call")
}

func (r *redeemRejectRepo) Use(ctx context.Context, id, userID int64) error {
	r.useCalled = true
	r.code.Status = StatusUsed
	r.code.UsedBy = &userID
	return nil
}

func (r *redeemRejectRepo) List(ctx context.Context, params pagination.PaginationParams) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}

func (r *redeemRejectRepo) ListWithFilters(ctx context.Context, params pagination.PaginationParams, codeType, status, search string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}

func (r *redeemRejectRepo) ListByUser(ctx context.Context, userID int64, limit int) ([]RedeemCode, error) {
	panic("unexpected ListByUser call")
}

func (r *redeemRejectRepo) ListByUserPaginated(ctx context.Context, userID int64, params pagination.PaginationParams, codeType string) ([]RedeemCode, *pagination.PaginationResult, error) {
	panic("unexpected ListByUserPaginated call")
}

func (r *redeemRejectRepo) SumPositiveBalanceByUser(ctx context.Context, userID int64) (float64, error) {
	panic("unexpected SumPositiveBalanceByUser call")
}

func TestRedeemRejectsInvitationCodeBeforeTransaction(t *testing.T) {
	ctx := context.Background()
	redeemRepo := &redeemRejectRepo{
		code: RedeemCode{
			ID:     1,
			Code:   "INVITE-001",
			Type:   RedeemTypeInvitation,
			Status: StatusUnused,
		},
	}
	redeemService := NewRedeemService(redeemRepo, nil, nil, nil, nil, nil, nil, nil)

	got, err := redeemService.Redeem(ctx, 2, redeemRepo.code.Code)

	require.Nil(t, got)
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
	require.Equal(t, "REDEEM_CODE_UNSUPPORTED_TYPE", infraerrors.Reason(err))
	require.Equal(t, "invitation codes can only be used during registration", infraerrors.Message(err))
	require.False(t, redeemRepo.useCalled)
	require.Equal(t, StatusUnused, redeemRepo.code.Status)
	require.Nil(t, redeemRepo.code.UsedBy)
}
