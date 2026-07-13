package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rewardDeliveryStoreStub struct {
	claimed     []RewardDelivery
	delivered   []int64
	failed      []rewardDeliveryFailure
	recovered   int
	processErr  map[int64]error
	markFailErr map[int64]error
}

type rewardDeliveryFailure struct {
	id          int64
	lastError   string
	nextRetryAt *time.Time
}

func (s *rewardDeliveryStoreStub) CreatePending(context.Context, CreateRewardDelivery) (*RewardDelivery, error) {
	panic("unexpected CreatePending")
}

func (s *rewardDeliveryStoreStub) ClaimDue(_ context.Context, _ time.Time, _ int) ([]RewardDelivery, error) {
	return s.claimed, nil
}

func (s *rewardDeliveryStoreStub) ClaimByID(_ context.Context, id int64, _ time.Time) (*RewardDelivery, error) {
	for i := range s.claimed {
		if s.claimed[i].ID == id {
			return &s.claimed[i], nil
		}
	}
	return nil, nil
}

func (s *rewardDeliveryStoreStub) MarkDelivered(_ context.Context, id int64, _ string, _ time.Time) error {
	s.delivered = append(s.delivered, id)
	return nil
}

func (s *rewardDeliveryStoreStub) ProcessClaimed(ctx context.Context, id int64, apply RewardDeliveryApply) error {
	if err := s.processErr[id]; err != nil {
		return err
	}
	var delivery RewardDelivery
	for i := range s.claimed {
		if s.claimed[i].ID == id {
			delivery = s.claimed[i]
			break
		}
	}
	if _, err := apply(ctx, delivery); err != nil {
		return err
	}
	s.delivered = append(s.delivered, id)
	return nil
}

func (s *rewardDeliveryStoreStub) MarkFailed(_ context.Context, id int64, lastError string, nextRetryAt *time.Time) error {
	if err := s.markFailErr[id]; err != nil {
		return err
	}
	s.failed = append(s.failed, rewardDeliveryFailure{id: id, lastError: lastError, nextRetryAt: nextRetryAt})
	return nil
}

func (s *rewardDeliveryStoreStub) RecoverStale(_ context.Context, _ time.Time, _ time.Time) (int, error) {
	s.recovered++
	return 1, nil
}

func (s *rewardDeliveryStoreStub) GetByID(context.Context, int64) (*RewardDelivery, error) {
	panic("unexpected GetByID")
}

func (s *rewardDeliveryStoreStub) List(context.Context, RewardDeliveryFilter) ([]RewardDelivery, int64, error) {
	panic("unexpected List")
}

type rewardDeliveryProcessorStub struct {
	errors map[int64]error
}

func (s rewardDeliveryProcessorStub) ProcessRewardDelivery(_ context.Context, delivery RewardDelivery) (string, error) {
	return "detail", s.errors[delivery.ID]
}

func TestRewardDeliveryWorkerRetriesAndTerminatesIndependently(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := &rewardDeliveryStoreStub{claimed: []RewardDelivery{
		{ID: 1, Attempts: 1},
		{ID: 2, Attempts: 5},
		{ID: 3, Attempts: 1},
	}}
	worker := NewRewardDeliveryWorker(store, rewardDeliveryProcessorStub{errors: map[int64]error{
		1: errors.New("temporary"),
		2: errors.New("permanent"),
	}}, RewardDeliveryWorkerOptions{MaxAttempts: 5, RetryDelay: 30 * time.Second})
	worker.now = func() time.Time { return now }

	require.NoError(t, worker.RunOnce(context.Background()))
	require.Equal(t, []int64{3}, store.delivered)
	require.Len(t, store.failed, 2)
	require.Equal(t, now.Add(30*time.Second), *store.failed[0].nextRetryAt)
	require.Nil(t, store.failed[1].nextRetryAt)
}

func TestRewardDeliveryWorkerContinuesAfterPersistenceError(t *testing.T) {
	store := &rewardDeliveryStoreStub{
		claimed:    []RewardDelivery{{ID: 1, Attempts: 1}, {ID: 2, Attempts: 1}},
		processErr: map[int64]error{1: errors.New("database unavailable")},
		markFailErr: map[int64]error{
			1: errors.New("cannot persist failure"),
		},
	}
	worker := NewRewardDeliveryWorker(store, rewardDeliveryProcessorStub{}, RewardDeliveryWorkerOptions{})

	err := worker.RunOnce(context.Background())

	require.ErrorContains(t, err, "mark reward delivery 1 failed")
	require.Equal(t, []int64{2}, store.delivered)
}

func TestRewardDeliveryWorkerRecoversStaleClaims(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := &rewardDeliveryStoreStub{}
	worker := NewRewardDeliveryWorker(store, rewardDeliveryProcessorStub{}, RewardDeliveryWorkerOptions{StaleAfter: time.Minute})
	worker.now = func() time.Time { return now }

	worker.recoverStale(context.Background())

	require.Equal(t, 1, store.recovered)
}

func TestRewardDeliveryWorkerRunByIDUsesRetryPolicy(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	store := &rewardDeliveryStoreStub{claimed: []RewardDelivery{{ID: 9, Attempts: 1}}}
	worker := NewRewardDeliveryWorker(store, rewardDeliveryProcessorStub{errors: map[int64]error{9: errors.New("temporary")}}, RewardDeliveryWorkerOptions{RetryDelay: time.Minute})
	worker.now = func() time.Time { return now }

	err := worker.RunByID(context.Background(), 9)

	require.ErrorContains(t, err, "temporary")
	require.Len(t, store.failed, 1)
	require.Equal(t, now.Add(time.Minute), *store.failed[0].nextRetryAt)
}
