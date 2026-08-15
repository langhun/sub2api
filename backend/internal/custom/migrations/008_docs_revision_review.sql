ALTER TABLE custom_docs_revisions
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'approved',
    ADD COLUMN IF NOT EXISTS reviewed_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS reviewed_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS review_note TEXT NOT NULL DEFAULT '';

UPDATE custom_docs_revisions
SET status = 'approved'
WHERE status IS NULL OR status = '';

ALTER TABLE custom_docs_revisions
    DROP CONSTRAINT IF EXISTS custom_docs_revisions_status_check;

ALTER TABLE custom_docs_revisions
    ADD CONSTRAINT custom_docs_revisions_status_check
    CHECK (status IN ('pending', 'approved', 'rejected'));

CREATE INDEX IF NOT EXISTS custom_docs_revisions_review_idx
    ON custom_docs_revisions (slug, locale, status, id DESC);
