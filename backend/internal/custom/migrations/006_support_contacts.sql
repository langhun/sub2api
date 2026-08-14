-- Customer-service contacts and their optional, administrator-uploaded QR images.
-- This table is owned exclusively by the Overlay migration ledger.
CREATE TABLE IF NOT EXISTS custom_support_contacts (
    id TEXT PRIMARY KEY,
    channel TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    contact_value TEXT NOT NULL,
    contact_link TEXT NOT NULL DEFAULT '',
    qr_image_url TEXT NOT NULL DEFAULT '',
    qr_image_data BYTEA,
    qr_image_mime TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT custom_support_contacts_channel_check
        CHECK (channel IN ('wechat', 'qq', 'telegram', 'email', 'link')),
    CONSTRAINT custom_support_contacts_qr_source_check
        CHECK (
            (qr_image_data IS NULL AND qr_image_mime = '') OR
            (qr_image_data IS NOT NULL AND qr_image_mime IN ('image/png', 'image/jpeg', 'image/webp'))
        )
);

CREATE INDEX IF NOT EXISTS custom_support_contacts_visible_order_idx
    ON custom_support_contacts (enabled, sort_order, created_at);
