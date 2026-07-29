package accountdrain

import (
	"context"
	"fmt"
)

type Service struct {
	repository *Repository
	policy     *Policy
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository, policy: NewPolicy()}
}

func (s *Service) Policy() *Policy { return s.policy }

func (s *Service) AccountStatus(ctx context.Context, accountID int64) (AccountTargetStatus, error) {
	if accountID <= 0 {
		return AccountTargetStatus{}, fmt.Errorf("invalid account ID")
	}
	active, err := s.repository.IsAccountTargeted(ctx, accountID)
	if err != nil {
		return AccountTargetStatus{}, err
	}
	return AccountTargetStatus{AccountID: accountID, Active: active}, nil
}

func (s *Service) EnableAccount(ctx context.Context, accountID int64) (AccountTargetStatus, error) {
	if accountID <= 0 {
		return AccountTargetStatus{}, fmt.Errorf("invalid account ID")
	}
	if _, err := s.repository.EnableAccount(ctx, accountID); err != nil {
		return AccountTargetStatus{}, err
	}
	if err := s.RefreshPolicy(ctx); err != nil {
		return AccountTargetStatus{}, fmt.Errorf("refresh account drain policy: %w", err)
	}
	return AccountTargetStatus{AccountID: accountID, Active: true}, nil
}

func (s *Service) DisableAccount(ctx context.Context, accountID int64) error {
	if accountID <= 0 {
		return fmt.Errorf("invalid account ID")
	}
	if err := s.repository.DisableAccount(ctx, accountID); err != nil {
		return err
	}
	return s.RefreshPolicy(ctx)
}

func (s *Service) RefreshPolicy(ctx context.Context) error {
	plans, err := s.repository.List(ctx)
	if err != nil {
		return err
	}
	s.policy.Replace(plans)
	return nil
}
