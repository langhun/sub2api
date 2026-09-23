-- Keep routing audit data bounded independently of attempt/outcome workers.
-- Decisions can be emitted by preview/no-route paths that never enqueue a
-- feedback outcome, so they need their own retention index and cleanup loop.
CREATE INDEX IF NOT EXISTS custom_route_decisions_created_idx
    ON custom_route_decisions (created_at);
