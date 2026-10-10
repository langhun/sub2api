CREATE TABLE IF NOT EXISTS custom_invoice_applications (
    id BIGSERIAL PRIMARY KEY,
    local_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    remote_application_id BIGINT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (local_user_id, remote_application_id)
);

CREATE INDEX IF NOT EXISTS custom_invoice_applications_user_idx
    ON custom_invoice_applications (local_user_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS custom_invoice_tax_orders (
    id BIGSERIAL PRIMARY KEY,
    local_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    remote_application_id BIGINT NOT NULL,
    tax_order_no TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (local_user_id, remote_application_id, tax_order_no)
);

CREATE INDEX IF NOT EXISTS custom_invoice_tax_orders_owner_idx
    ON custom_invoice_tax_orders (local_user_id, tax_order_no);
