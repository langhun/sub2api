CREATE TABLE IF NOT EXISTS custom_docs_revisions (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL,
    locale TEXT NOT NULL CHECK (locale IN ('zh', 'en')),
    content TEXT NOT NULL,
    edit_summary TEXT NOT NULL DEFAULT '',
    author_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_docs_revisions_slug_check CHECK (slug <> '' AND length(slug) <= 80),
    CONSTRAINT custom_docs_revisions_content_check CHECK (length(content) > 0 AND length(content) <= 200000)
);

CREATE INDEX IF NOT EXISTS custom_docs_revisions_lookup_idx
    ON custom_docs_revisions (slug, locale, id DESC);
