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

func (s *Service) List(ctx context.Context) ([]Plan, error) {
	plans, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	return plans, nil
}

func (s *Service) Create(ctx context.Context, input CreatePlanInput) (*Plan, error) {
	input, err := normalizeCreateInput(input)
	if err != nil {
		return nil, err
	}
	plan, err := s.repository.Create(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.RefreshPolicy(ctx); err != nil {
		return nil, fmt.Errorf("refresh account drain policy: %w", err)
	}
	return plan, nil
}

func (s *Service) Stop(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid plan ID")
	}
	if err := s.repository.Stop(ctx, id); err != nil {
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
