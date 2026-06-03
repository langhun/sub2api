package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type announcementRepoStub struct {
	item   *Announcement
	active []Announcement
}

func (s *announcementRepoStub) Create(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (s *announcementRepoStub) GetByID(_ context.Context, _ int64) (*Announcement, error) {
	if s.item == nil {
		return nil, ErrAnnouncementNotFound
	}
	return s.item, nil
}

func (s *announcementRepoStub) Update(_ context.Context, a *Announcement) error {
	s.item = a
	return nil
}

func (*announcementRepoStub) Delete(context.Context, int64) error {
	return nil
}

func (*announcementRepoStub) List(context.Context, pagination.PaginationParams, AnnouncementListFilters) ([]Announcement, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *announcementRepoStub) ListActive(context.Context, time.Time) ([]Announcement, error) {
	return s.active, nil
}

type announcementUserRepoStub struct {
	UserRepository
	user *User
}

func (s *announcementUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	return s.user, nil
}

type announcementReadRepoStub struct {
	AnnouncementReadRepository
}

func (*announcementReadRepoStub) GetReadMapByUser(context.Context, int64, []int64) (map[int64]time.Time, error) {
	return map[int64]time.Time{}, nil
}

type announcementUserSubRepoStub struct {
	UserSubscriptionRepository
}

func (*announcementUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	return nil, nil
}

func TestAnnouncementServiceCreateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{}
	svc := NewAnnouncementService(repo, nil, nil, nil)
	now := time.Unix(1776790020, 0)

	_, err := svc.Create(context.Background(), &CreateAnnouncementInput{
		Title:      "公告",
		Content:    "内容",
		Status:     AnnouncementStatusActive,
		NotifyMode: AnnouncementNotifyModePopup,
		StartsAt:   &now,
		EndsAt:     &now,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceUpdateRejectsEqualStartEndTimes(t *testing.T) {
	repo := &announcementRepoStub{
		item: &Announcement{
			ID:         1,
			Title:      "公告",
			Content:    "内容",
			Status:     AnnouncementStatusActive,
			NotifyMode: AnnouncementNotifyModePopup,
		},
	}
	svc := NewAnnouncementService(repo, nil, nil, nil)
	now := time.Unix(1776790020, 0)
	startsAt := &now
	endsAt := &now

	_, err := svc.Update(context.Background(), 1, &UpdateAnnouncementInput{
		StartsAt: &startsAt,
		EndsAt:   &endsAt,
	})
	require.ErrorIs(t, err, ErrAnnouncementInvalidSchedule)
}

func TestAnnouncementServiceListForUserUsesBankBalanceForTargeting(t *testing.T) {
	repo := &announcementRepoStub{
		active: []Announcement{
			{
				ID:         1,
				Title:      "balance targeted",
				Content:    "content",
				Status:     AnnouncementStatusActive,
				NotifyMode: AnnouncementNotifyModePopup,
				Targeting: AnnouncementTargeting{
					AnyOf: []AnnouncementConditionGroup{
						{
							AllOf: []AnnouncementCondition{
								{
									Type:     AnnouncementConditionTypeBalance,
									Operator: AnnouncementOperatorGTE,
									Value:    50,
								},
							},
						},
					},
				},
			},
		},
	}
	userRepo := &announcementUserRepoStub{
		user: &User{
			ID:      1,
			Balance: 100,
			BankAccount: &BankAccountView{
				Balance: decimal.Zero,
				Status:  BankAccountStatusActive,
			},
		},
	}
	svc := NewAnnouncementService(repo, &announcementReadRepoStub{}, userRepo, &announcementUserSubRepoStub{})

	items, err := svc.ListForUser(context.Background(), 1, false)

	require.NoError(t, err)
	require.Empty(t, items)
}
