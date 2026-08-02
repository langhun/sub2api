CREATE TABLE IF NOT EXISTS custom_support_tickets (
    id BIGSERIAL PRIMARY KEY,
    public_id VARCHAR(32) NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(32) NOT NULL,
    subject VARCHAR(200) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'open',
    priority VARCHAR(16) NOT NULL DEFAULT 'normal',
    last_reply_by VARCHAR(16) NOT NULL DEFAULT 'user',
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_last_read_at TIMESTAMPTZ,
    admin_last_read_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_support_tickets_category_check CHECK (category IN ('account', 'billing', 'api', 'model', 'feature', 'other')),
    CONSTRAINT custom_support_tickets_status_check CHECK (status IN ('open', 'waiting_user', 'resolved', 'closed')),
    CONSTRAINT custom_support_tickets_priority_check CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    CONSTRAINT custom_support_tickets_last_reply_check CHECK (last_reply_by IN ('user', 'admin'))
);

CREATE INDEX IF NOT EXISTS custom_support_tickets_user_activity_idx
    ON custom_support_tickets (user_id, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS custom_support_tickets_status_activity_idx
    ON custom_support_tickets (status, last_activity_at DESC);

CREATE TABLE IF NOT EXISTS custom_support_ticket_messages (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES custom_support_tickets(id) ON DELETE CASCADE,
    author_type VARCHAR(16) NOT NULL,
    author_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    message_type VARCHAR(16) NOT NULL DEFAULT 'reply',
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_support_ticket_messages_author_check CHECK (author_type IN ('user', 'admin')),
    CONSTRAINT custom_support_ticket_messages_type_check CHECK (message_type IN ('reply', 'internal_note')),
    CONSTRAINT custom_support_ticket_messages_body_check CHECK (char_length(body) BETWEEN 1 AND 5000)
);

CREATE INDEX IF NOT EXISTS custom_support_ticket_messages_ticket_created_idx
    ON custom_support_ticket_messages (ticket_id, created_at ASC, id ASC);

CREATE TABLE IF NOT EXISTS custom_support_ticket_events (
    id BIGSERIAL PRIMARY KEY,
    ticket_id BIGINT NOT NULL REFERENCES custom_support_tickets(id) ON DELETE CASCADE,
    actor_type VARCHAR(16) NOT NULL,
    actor_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    event_type VARCHAR(32) NOT NULL,
    old_value VARCHAR(64),
    new_value VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_support_ticket_events_actor_check CHECK (actor_type IN ('user', 'admin'))
);

CREATE INDEX IF NOT EXISTS custom_support_ticket_events_ticket_created_idx
    ON custom_support_ticket_events (ticket_id, created_at ASC, id ASC);
