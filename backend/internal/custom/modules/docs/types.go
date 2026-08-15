// Package docs owns user-editable documentation revisions.
package docs

import "time"

const ModuleID = "docs"

const (
	MaxContentBytes = 200_000
	MaxSummaryRunes = 200
	MaxHistoryItems = 50
)

type Revision struct {
	ID           int64     `json:"id"`
	Slug         string    `json:"slug"`
	Locale       string    `json:"locale"`
	Content      string    `json:"content"`
	EditSummary  string    `json:"edit_summary"`
	AuthorUserID int64     `json:"-"`
	AuthorName   string    `json:"author_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type CreateInput struct {
	Slug, Locale, Content, EditSummary string
	AuthorUserID                       int64
	BaseRevisionID                     *int64
}

type Revisions struct {
	Current *Revision  `json:"current"`
	History []Revision `json:"history"`
}
