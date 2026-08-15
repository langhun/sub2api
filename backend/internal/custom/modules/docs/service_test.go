package docs

import (
	"context"
	"strings"
	"testing"
)

func TestValidatePath(t *testing.T) {
	for _, test := range []struct {
		name   string
		slug   string
		locale string
		valid  bool
	}{
		{name: "valid chinese", slug: "quick-start", locale: "zh", valid: true},
		{name: "valid english", slug: "grok-cli", locale: "en", valid: true},
		{name: "path traversal", slug: "../quick-start", locale: "zh"},
		{name: "invalid locale", slug: "quick-start", locale: "fr"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validatePath(test.slug, test.locale)
			if (err == nil) != test.valid {
				t.Fatalf("validatePath(%q, %q) error = %v, valid = %v", test.slug, test.locale, err, test.valid)
			}
		})
	}
}

func TestCreateRejectsUnsafeOrOversizedMarkdown(t *testing.T) {
	service := NewService(nil)
	for _, content := range []string{
		"<script>alert(1)</script>",
		"[x](javascript:alert(1))",
		"<div onclick=alert(1)>click</div>",
		"<svg onload=alert(1)></svg>",
		strings.Repeat("x", MaxContentBytes+1),
	} {
		_, err := service.Create(context.Background(), CreateInput{Slug: "quick-start", Locale: "zh", Content: content, AuthorUserID: 1})
		if err == nil {
			t.Fatalf("Create accepted unsafe/oversized content: %q", content[:min(30, len(content))])
		}
	}
}

func TestContainsUnsafeMarkdown(t *testing.T) {
	if !containsUnsafeMarkdown("Use `<script>` as an example in a code block") {
		t.Fatal("unsafe marker should be rejected consistently")
	}
	if containsUnsafeMarkdown("# Safe\n\nUse **markdown**") {
		t.Fatal("safe markdown was rejected")
	}
}

func TestServiceReviewRequiresAnExplicitDecision(t *testing.T) {
	service := NewService(nil)
	_, err := service.Review(context.Background(), ReviewInput{
		Slug: "quick-start", Locale: "zh", RevisionID: 1, ReviewerUserID: 2, Decision: "publish",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid revision review decision") {
		t.Fatalf("expected invalid decision error, got %v", err)
	}
}

func TestServiceReviewRejectsOversizedReviewNote(t *testing.T) {
	service := NewService(nil)
	_, err := service.Review(context.Background(), ReviewInput{
		Slug: "quick-start", Locale: "zh", RevisionID: 1, ReviewerUserID: 2, Decision: "reject",
		ReviewNote: strings.Repeat("x", MaxReviewNoteRunes+1),
	})
	if err == nil || !strings.Contains(err.Error(), "review note must be at most") {
		t.Fatalf("expected review note limit error, got %v", err)
	}
}

func TestServiceReviewRejectsInvalidIdentity(t *testing.T) {
	service := NewService(nil)
	_, err := service.Review(context.Background(), ReviewInput{
		Slug: "quick-start", Locale: "zh", RevisionID: 0, ReviewerUserID: 2, Decision: "approve",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid revision review") {
		t.Fatalf("expected invalid revision review error, got %v", err)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
