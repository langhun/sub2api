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

func (r *Repository) List(ctx context.Context, slug, locale string, viewerUserID int64, includeAll bool) (*Revisions, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("documentation database is unavailable")
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.id, d.slug, d.locale, d.content, d.edit_summary, d.author_user_id,
		       COALESCE(NULLIF(u.username, ''), 'user #' || u.id::text), d.created_at,
		       d.status, COALESCE(NULLIF(reviewer.username, ''), 'user #' || reviewer.id::text),
		       d.reviewed_at, d.review_note
		FROM custom_docs_revisions d
		JOIN users u ON u.id = d.author_user_id
		LEFT JOIN users reviewer ON reviewer.id = d.reviewed_by_user_id
		WHERE d.slug = $1 AND d.locale = $2
		  AND (d.status = 'approved' OR d.author_user_id = $3 OR $4)
		ORDER BY d.id DESC
		LIMIT $5`, slug, locale, viewerUserID, includeAll, MaxHistoryItems)
	if err != nil {
		return nil, fmt.Errorf("list documentation revisions: %w", err)
	}
	defer rows.Close()
	result := &Revisions{History: make([]Revision, 0)}
	for rows.Next() {
		var revision Revision
		if err := rows.Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary, &revision.AuthorUserID, &revision.AuthorName, &revision.CreatedAt, &revision.Status, &revision.ReviewerName, &revision.ReviewedAt, &revision.ReviewNote); err != nil {
			return nil, fmt.Errorf("scan documentation revision: %w", err)
		}
		result.History = append(result.History, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documentation revisions: %w", err)
	}
	var current Revision
	if err := r.db.QueryRowContext(ctx, `
		SELECT d.id, d.slug, d.locale, d.content, d.edit_summary, d.author_user_id,
		       COALESCE(NULLIF(u.username, ''), 'user #' || u.id::text), d.created_at,
		       d.status, COALESCE(NULLIF(reviewer.username, ''), 'user #' || reviewer.id::text),
		       d.reviewed_at, d.review_note
		FROM custom_docs_revisions d
		JOIN users u ON u.id = d.author_user_id
		LEFT JOIN users reviewer ON reviewer.id = d.reviewed_by_user_id
		WHERE d.slug = $1 AND d.locale = $2 AND d.status = 'approved'
		ORDER BY d.id DESC LIMIT 1`, slug, locale).Scan(
		&current.ID, &current.Slug, &current.Locale, &current.Content,
		&current.EditSummary, &current.AuthorUserID, &current.AuthorName,
		&current.CreatedAt, &current.Status, &current.ReviewerName,
		&current.ReviewedAt, &current.ReviewNote,
	); err == nil {
		// Current points to a separately loaded approved revision so pending or
		// rejected submissions can never change the document shown to readers.
		result.Current = &current
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("read current documentation revision: %w", err)
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
	err = tx.QueryRowContext(ctx, `SELECT id FROM custom_docs_revisions WHERE slug = $1 AND locale = $2 AND status = 'approved' ORDER BY id DESC LIMIT 1 FOR UPDATE`, input.Slug, input.Locale).Scan(&currentID)
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
		INSERT INTO custom_docs_revisions (slug, locale, content, edit_summary, author_user_id, status)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		RETURNING id, slug, locale, content, edit_summary, author_user_id, created_at, status`,
		input.Slug, input.Locale, input.Content, input.EditSummary, input.AuthorUserID,
	).Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary, &revision.AuthorUserID, &revision.CreatedAt, &revision.Status)
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
	err = tx.QueryRowContext(ctx, `SELECT id, slug, locale, content FROM custom_docs_revisions WHERE id = $1 AND slug = $2 AND locale = $3 AND status = 'approved'`, revisionID, slug, locale).Scan(&source.ID, &source.Slug, &source.Locale, &source.Content)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRevisionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read documentation revision to revert: %w", err)
	}
	var revision Revision
	err = tx.QueryRowContext(ctx, `
		INSERT INTO custom_docs_revisions (slug, locale, content, edit_summary, author_user_id, status, reviewed_by_user_id, reviewed_at)
		VALUES ($1, $2, $3, $4, $5, 'approved', $5, NOW())
		RETURNING id, slug, locale, content, edit_summary, author_user_id, created_at, status, reviewed_at`,
		slug, locale, source.Content, fmt.Sprintf("reverted to revision #%d", revisionID), authorUserID,
	).Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary, &revision.AuthorUserID, &revision.CreatedAt, &revision.Status, &revision.ReviewedAt)
	if err != nil {
		return nil, fmt.Errorf("insert documentation revert: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(username, ''), 'user #' || id::text) FROM users WHERE id = $1`, authorUserID).Scan(&revision.AuthorName); err != nil {
		return nil, fmt.Errorf("read reverting author: %w", err)
	}
	revision.ReviewerName = revision.AuthorName
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit documentation revert: %w", err)
	}
	return &revision, nil
}

func (r *Repository) Review(ctx context.Context, input ReviewInput) (*Revision, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("documentation database is unavailable")
	}
	var revision Revision
	err := r.db.QueryRowContext(ctx, `
		UPDATE custom_docs_revisions
		SET status = $1, reviewed_by_user_id = $2, reviewed_at = NOW(), review_note = $3
		WHERE id = $4 AND slug = $5 AND locale = $6 AND status = 'pending'
		RETURNING id, slug, locale, content, edit_summary, author_user_id, created_at, status, reviewed_at, review_note`,
		input.Decision+"d", input.ReviewerUserID, input.ReviewNote, input.RevisionID, input.Slug, input.Locale,
	).Scan(&revision.ID, &revision.Slug, &revision.Locale, &revision.Content, &revision.EditSummary,
		&revision.AuthorUserID, &revision.CreatedAt, &revision.Status, &revision.ReviewedAt, &revision.ReviewNote)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRevisionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("review documentation revision: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(username, ''), 'user #' || id::text) FROM users WHERE id = $1`, revision.AuthorUserID).Scan(&revision.AuthorName); err != nil {
		return nil, fmt.Errorf("read documentation author: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(NULLIF(username, ''), 'user #' || id::text) FROM users WHERE id = $1`, input.ReviewerUserID).Scan(&revision.ReviewerName); err != nil {
		return nil, fmt.Errorf("read documentation reviewer: %w", err)
	}
	return &revision, nil
}
