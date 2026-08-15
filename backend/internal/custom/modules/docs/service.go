package docs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,79}$`)
var unsafeMarkdownPattern = regexp.MustCompile(`(?i)<\s*(script|style|iframe|object|embed|form|svg|video|audio|source)\b|\bon[a-z]+\s*=|\b(?:javascript|vbscript):|\bdata:text/html`)

type Service struct{ repository *Repository }

func NewService(repository *Repository) *Service { return &Service{repository: repository} }

func (s *Service) List(ctx context.Context, slug, locale string, viewerUserID int64, includeAll bool) (*Revisions, error) {
	if err := validatePath(slug, locale); err != nil {
		return nil, err
	}
	if viewerUserID <= 0 {
		return nil, errors.New("invalid viewer")
	}
	return s.repository.List(ctx, slug, locale, viewerUserID, includeAll)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Revision, error) {
	if err := validatePath(input.Slug, input.Locale); err != nil {
		return nil, err
	}
	if input.AuthorUserID <= 0 {
		return nil, errors.New("invalid author")
	}
	input.Content = strings.TrimSpace(input.Content)
	input.EditSummary = strings.TrimSpace(input.EditSummary)
	if input.Content == "" {
		return nil, errors.New("documentation content is required")
	}
	if len([]byte(input.Content)) > MaxContentBytes {
		return nil, fmt.Errorf("documentation content must be at most %d bytes", MaxContentBytes)
	}
	if utf8.RuneCountInString(input.EditSummary) > MaxSummaryRunes {
		return nil, fmt.Errorf("edit summary must be at most %d characters", MaxSummaryRunes)
	}
	if containsUnsafeMarkdown(input.Content) {
		return nil, errors.New("documentation contains unsupported HTML or URL content")
	}
	return s.repository.Create(ctx, input)
}

func (s *Service) Revert(ctx context.Context, slug, locale string, revisionID, authorUserID int64) (*Revision, error) {
	if err := validatePath(slug, locale); err != nil {
		return nil, err
	}
	if revisionID <= 0 || authorUserID <= 0 {
		return nil, errors.New("invalid revision")
	}
	return s.repository.Revert(ctx, slug, locale, revisionID, authorUserID)
}

func (s *Service) Review(ctx context.Context, input ReviewInput) (*Revision, error) {
	if err := validatePath(input.Slug, input.Locale); err != nil {
		return nil, err
	}
	if input.RevisionID <= 0 || input.ReviewerUserID <= 0 {
		return nil, errors.New("invalid revision review")
	}
	if input.Decision != "approve" && input.Decision != "reject" {
		return nil, errors.New("invalid revision review decision")
	}
	input.ReviewNote = strings.TrimSpace(input.ReviewNote)
	if utf8.RuneCountInString(input.ReviewNote) > MaxReviewNoteRunes {
		return nil, fmt.Errorf("review note must be at most %d characters", MaxReviewNoteRunes)
	}
	return s.repository.Review(ctx, input)
}

func validatePath(slug, locale string) error {
	if !slugPattern.MatchString(slug) {
		return errors.New("invalid documentation slug")
	}
	if locale != "zh" && locale != "en" {
		return errors.New("invalid documentation locale")
	}
	return nil
}

func containsUnsafeMarkdown(content string) bool {
	return unsafeMarkdownPattern.MatchString(content)
}
