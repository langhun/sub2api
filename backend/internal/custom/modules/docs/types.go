// Package docs owns user-editable documentation revisions.
package docs

import "time"

const ModuleID = "docs"

const (
	MaxContentBytes    = 200_000
	MaxSummaryRunes    = 200
	MaxReviewNoteRunes = 200
	MaxHistoryItems    = 50
)

type Revision struct {
	ID           int64      `json:"id"`
	Slug         string     `json:"slug"`
	Locale       string     `json:"locale"`
	Content      string     `json:"content"`
	EditSummary  string     `json:"edit_summary"`
	AuthorUserID int64      `json:"-"`
	AuthorName   string     `json:"author_name"`
	CreatedAt    time.Time  `json:"created_at"`
	Status       string     `json:"status"`
	ReviewerName string     `json:"reviewer_name,omitempty"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote   string     `json:"review_note,omitempty"`
}

type CreateInput struct {
	Slug, Locale, Content, EditSummary string
	AuthorUserID                       int64
	BaseRevisionID                     *int64
}

type ReviewInput struct {
	Slug, Locale, Decision, ReviewNote string
	RevisionID                         int64
	ReviewerUserID                     int64
}

type Revisions struct {
	Current *Revision  `json:"current"`
	History []Revision `json:"history"`
}
