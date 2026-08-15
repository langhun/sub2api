package docs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrConflict         = errors.New("documentation revision is out of date")
	ErrRevisionNotFound = errors.New("documentation revision not found")
)

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, slug, locale string) (*Revisions, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("documentation database is unavailable")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.slug, d.locale, d.content, d.edit_summary, d.author_user_id,
		       COALESCE(NULLIF(u.username, ''), 'user #' || u.id::text), d.created_at
		FROM custom_docs_revisions d
		JOIN users u ON u.id = d.author_user_id
		WHERE d.slug = $1 AND d.locale = $2
		ORDER BY d.id DESC
		LIMIT $3`, slug, locale, MaxHistoryItems)
	if err != nil {
		return nil, fmt.Errorf("list documentation revisions: %w", err)
	}
	defer rows.Close()
	result := &Revisions{History: make([]Revision, 0)}
	for rows.Next() {
		var revision Revision
		if err := rows.Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary, &revision.AuthorUserID, &revision.AuthorName, &revision.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan documentation revision: %w", err)
		}
		result.History = append(result.History, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documentation revisions: %w", err)
	}
	if len(result.History) > 0 {
		result.Current = &result.History[0]
	}
	return result, nil
}

func (r *Repository) Create(ctx context.Context, input CreateInput) (*Revision, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("documentation database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin documentation revision: %w", err)
	}
	defer tx.Rollback()

	var currentID sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id FROM custom_docs_revisions WHERE slug = $1 AND locale = $2 ORDER BY id DESC LIMIT 1 FOR UPDATE`, input.Slug, input.Locale).Scan(&currentID)
	if errors.Is(err, sql.ErrNoRows) {
		currentID = sql.NullInt64{}
	} else if err != nil {
		return nil, fmt.Errorf("lock current documentation revision: %w", err)
	}
	if (input.BaseRevisionID == nil && currentID.Valid) || (input.BaseRevisionID != nil && (!currentID.Valid || *input.BaseRevisionID != currentID.Int64)) {
		return nil, ErrConflict
	}

	var revision Revision
	err = tx.QueryRowContext(ctx, `
		INSERT INTO custom_docs_revisions (slug, locale, content, edit_summary, author_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, slug, locale, content, edit_summary, author_user_id, created_at`,
		input.Slug, input.Locale, input.Content, input.EditSummary, input.AuthorUserID,
	).Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary, &revision.AuthorUserID, &revision.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert documentation revision: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(username, ''), 'user #' || id::text) FROM users WHERE id = $1`, revision.AuthorUserID).Scan(&revision.AuthorName); err != nil {
		return nil, fmt.Errorf("read documentation author: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit documentation revision: %w", err)
	}
	return &revision, nil
}

func (r *Repository) Revert(ctx context.Context, slug, locale string, revisionID, authorUserID int64) (*Revision, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("documentation database is unavailable")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin documentation revert: %w", err)
	}
	defer tx.Rollback()
	var source Revision
	err = tx.QueryRowContext(ctx, `SELECT id, slug, locale, content FROM custom_docs_revisions WHERE id = $1 AND slug = $2 AND locale = $3`, revisionID, slug, locale).Scan(&source.ID, &source.Slug, &source.Locale, &source.Content)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRevisionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read documentation revision to revert: %w", err)
	}
	var revision Revision
	err = tx.QueryRowContext(ctx, `
		INSERT INTO custom_docs_revisions (slug, locale, content, edit_summary, author_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, slug, locale, content, edit_summary, author_user_id, created_at`,
		slug, locale, source.Content, fmt.Sprintf("reverted to revision #%d", revisionID), authorUserID,
	).Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary, &revision.AuthorUserID, &revision.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert documentation revert: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(username, ''), 'user #' || id::text) FROM users WHERE id = $1`, authorUserID).Scan(&revision.AuthorName); err != nil {
		return nil, fmt.Errorf("read reverting author: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit documentation revert: %w", err)
	}
	return &revision, nil
}
