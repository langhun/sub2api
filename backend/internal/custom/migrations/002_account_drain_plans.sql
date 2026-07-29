CREATE TABLE IF NOT EXISTS custom_account_drain_plans (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'stopped')),
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS custom_account_drain_plan_accounts (
    plan_id BIGINT NOT NULL REFERENCES custom_account_drain_plans(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    PRIMARY KEY (plan_id, account_id)
);

CREATE INDEX IF NOT EXISTS custom_account_drain_plans_active_idx
    ON custom_account_drain_plans (status, expires_at);

CREATE INDEX IF NOT EXISTS custom_account_drain_plan_accounts_account_idx
    ON custom_account_drain_plan_accounts (account_id);
