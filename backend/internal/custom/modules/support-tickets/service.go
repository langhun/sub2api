package supporttickets

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

type Service struct {
	store    *Repository
	settings SettingsReader
}

func NewService(store *Repository, settings SettingsReader) *Service {
	return &Service{store: store, settings: settings}
}

func (s *Service) ListUser(ctx context.Context, userID int64, page, pageSize int) ([]Ticket, int64, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, 0, err
	}
	return s.store.List(ctx, ListFilter{Page: page, PageSize: pageSize, UserID: &userID})
}
func (s *Service) ListAdmin(ctx context.Context, filter ListFilter) ([]Ticket, int64, error) {
	return s.store.List(ctx, filter)
}
func (s *Service) DetailUser(ctx context.Context, userID int64, publicID string) (*TicketDetail, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, err
	}
	if !validPublicID(publicID) {
		return nil, errors.New("invalid ticket id")
	}
	return s.store.Detail(ctx, publicID, &userID, false)
}
func (s *Service) DetailAdmin(ctx context.Context, publicID string) (*TicketDetail, error) {
	if !validPublicID(publicID) {
		return nil, errors.New("invalid ticket id")
	}
	return s.store.Detail(ctx, publicID, nil, true)
}
func (s *Service) Create(ctx context.Context, input CreateInput) (*Ticket, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, err
	}
	if err := validateCreate(input); err != nil {
		return nil, err
	}
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	id, err := newPublicID()
	if err != nil {
		return nil, err
	}
	return s.store.Create(ctx, Ticket{PublicID: id, UserID: input.UserID, Category: input.Category, Subject: input.Subject}, input.Body)
}
func (s *Service) ReplyUser(ctx context.Context, input ReplyInput) (*Ticket, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, err
	}
	input.ActorType = "user"
	if err := validateReply(input); err != nil {
		return nil, err
	}
	input.Body = strings.TrimSpace(input.Body)
	return s.store.AddReply(ctx, input)
}
func (s *Service) ReplyAdmin(ctx context.Context, input ReplyInput) (*Ticket, error) {
	input.ActorType = "admin"
	if err := validateReply(input); err != nil {
		return nil, err
	}
	input.Body = strings.TrimSpace(input.Body)
	return s.store.AddReply(ctx, input)
}
func (s *Service) Reopen(ctx context.Context, publicID string, userID int64) (*Ticket, error) {
	if err := s.enabled(ctx); err != nil {
		return nil, err
	}
	if !validPublicID(publicID) {
		return nil, errors.New("invalid ticket id")
	}
	return s.store.Reopen(ctx, publicID, userID)
}
func (s *Service) Update(ctx context.Context, input UpdateInput) (*Ticket, error) {
	if !validPublicID(input.PublicID) {
		return nil, errors.New("invalid ticket id")
	}
	if input.Status != "" {
		if !validStatus(input.Status) {
			return nil, errors.New("invalid ticket status")
		}
		return s.store.UpdateStatus(ctx, input.PublicID, input.UserID, input.Status)
	}
	if input.Priority != "" {
		if !validPriority(input.Priority) {
			return nil, errors.New("invalid ticket priority")
		}
		return s.store.UpdatePriority(ctx, input.PublicID, input.UserID, input.Priority)
	}
	return nil, errors.New("status or priority is required")
}
func (s *Service) enabled(ctx context.Context) error {
	if s == nil || s.store == nil {
		return errors.New("support tickets are unavailable")
	}
	if s.settings == nil {
		return nil
	}
	config, err := s.settings.ReadSupportTickets(ctx)
	if err != nil {
		return fmt.Errorf("read support ticket settings: %w", err)
	}
	if !config.Enabled {
		return errors.New("support tickets are disabled")
	}
	return nil
}

func validateCreate(input CreateInput) error {
	if input.UserID <= 0 {
		return errors.New("invalid user")
	}
	if !validCategory(input.Category) {
		return errors.New("invalid ticket category")
	}
	if err := plainText(input.Subject, 1, 200, "subject"); err != nil {
		return err
	}
	return plainText(input.Body, 1, 5000, "message")
}
func validateReply(input ReplyInput) error {
	if input.UserID <= 0 || !validPublicID(input.PublicID) {
		return errors.New("invalid ticket id")
	}
	return plainText(input.Body, 1, 5000, "message")
}
func plainText(value string, min, max int, name string) error {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) < min || utf8.RuneCountInString(value) > max {
		return fmt.Errorf("%s must be between %d and %d characters", name, min, max)
	}
	if strings.ContainsAny(value, "\x00\r") {
		return fmt.Errorf("%s contains unsupported characters", name)
	}
	return nil
}
func validCategory(v string) bool {
	for _, x := range []string{"account", "billing", "api", "model", "feature", "other"} {
		if v == x {
			return true
		}
	}
	return false
}
func validStatus(v string) bool {
	for _, x := range []string{StatusOpen, StatusWaitingUser, StatusResolved, StatusClosed} {
		if v == x {
			return true
		}
	}
	return false
}
func validPriority(v string) bool {
	for _, x := range []string{PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent} {
		if v == x {
			return true
		}
	}
	return false
}
func validPublicID(v string) bool { return len(v) == 20 && strings.HasPrefix(v, "SUP-") }
func newPublicID() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate ticket id: %w", err)
	}
	return "SUP-" + strings.ToUpper(hex.EncodeToString(buf)), nil
}
