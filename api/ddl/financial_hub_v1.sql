-- Financial Hub V1 DDL snapshot
-- Authoritative schema snapshot derived from migrations 145_bank_foundation.sql
-- and 146_core_banking_ledger.sql for Architecture Freeze v1.1.

CREATE TABLE IF NOT EXISTS users_bank_account (
    id                   BIGSERIAL PRIMARY KEY,
    user_id              BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance              NUMERIC(38, 18) NOT NULL DEFAULT 0,
    frozen_amount        NUMERIC(38, 18) NOT NULL DEFAULT 0,
    credit_limit         NUMERIC(38, 18) NOT NULL DEFAULT 0,
    debt_principal       NUMERIC(38, 18) NOT NULL DEFAULT 0,
    debt_interest        NUMERIC(38, 18) NOT NULL DEFAULT 0,
    total_debt           NUMERIC(38, 18) NOT NULL DEFAULT 0,
    version              BIGINT NOT NULL DEFAULT 1,
    status               VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_bank_account_user_id_unique UNIQUE (user_id),
    CONSTRAINT users_bank_account_frozen_non_negative CHECK (frozen_amount >= 0),
    CONSTRAINT users_bank_account_credit_limit_non_negative CHECK (credit_limit >= 0),
    CONSTRAINT users_bank_account_total_debt_non_negative CHECK (total_debt >= 0),
    CONSTRAINT users_bank_account_debt_principal_non_negative CHECK (debt_principal >= 0),
    CONSTRAINT users_bank_account_debt_interest_non_negative CHECK (debt_interest >= 0),
    CONSTRAINT users_bank_account_version_positive CHECK (version > 0),
    CONSTRAINT users_bank_account_status_check CHECK (status IN ('ACTIVE', 'FROZEN', 'CLOSED'))
);

CREATE INDEX IF NOT EXISTS idx_users_bank_account_user_id ON users_bank_account(user_id);
CREATE INDEX IF NOT EXISTS idx_users_bank_account_status ON users_bank_account(status);

CREATE TABLE IF NOT EXISTS transactions_log (
    id                    BIGSERIAL PRIMARY KEY,
    tx_id                 UUID NOT NULL,
    user_id               BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id            BIGINT NOT NULL REFERENCES users_bank_account(id) ON DELETE CASCADE,
    tx_type               VARCHAR(32) NOT NULL,
    business_module       VARCHAR(32) NOT NULL DEFAULT 'FINANCIAL_HUB',
    amount                NUMERIC(38, 18) NOT NULL,
    balance_before        NUMERIC(38, 18) NOT NULL,
    balance_after         NUMERIC(38, 18) NOT NULL,
    frozen_before         NUMERIC(38, 18) NOT NULL,
    frozen_after          NUMERIC(38, 18) NOT NULL,
    credit_limit_snapshot NUMERIC(38, 18) NOT NULL DEFAULT 0,
    debt_snapshot         NUMERIC(38, 18) NOT NULL DEFAULT 0,
    description           TEXT NOT NULL DEFAULT '',
    reference_type        VARCHAR(64),
    reference_id          VARCHAR(128),
    request_id            VARCHAR(128),
    idempotency_scope     VARCHAR(128) NOT NULL,
    idempotency_key_hash  VARCHAR(64) NOT NULL,
    metadata              JSONB NOT NULL DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transactions_log_tx_id_unique UNIQUE (tx_id),
    CONSTRAINT transactions_log_idempotency_unique UNIQUE (idempotency_scope, idempotency_key_hash),
    CONSTRAINT transactions_log_tx_type_check CHECK (
        tx_type IN (
            'CONSUME',
            'DEPOSIT',
            'WITHDRAW',
            'TRANSFER_OUT',
            'TRANSFER_IN',
            'SLOT_BET',
            'SLOT_WIN',
            'LOTTERY_BET',
            'LOTTERY_WIN',
            'LOAN_BORROW',
            'LOAN_REPAY',
            'LOAN_INTEREST',
            'LEND_INVEST',
            'LEND_PROFIT',
            'REWARD',
            'REFUND',
            'FREEZE',
            'UNFREEZE'
        )
    ),
    CONSTRAINT transactions_log_frozen_before_non_negative CHECK (frozen_before >= 0),
    CONSTRAINT transactions_log_frozen_after_non_negative CHECK (frozen_after >= 0),
    CONSTRAINT transactions_log_credit_limit_snapshot_non_negative CHECK (credit_limit_snapshot >= 0),
    CONSTRAINT transactions_log_debt_snapshot_non_negative CHECK (debt_snapshot >= 0)
);

CREATE INDEX IF NOT EXISTS idx_transactions_log_user_id ON transactions_log(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_log_account_id ON transactions_log(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_log_tx_type ON transactions_log(tx_type);
CREATE INDEX IF NOT EXISTS idx_transactions_log_business_module ON transactions_log(business_module);
CREATE INDEX IF NOT EXISTS idx_transactions_log_module_type ON transactions_log(business_module, tx_type);
CREATE INDEX IF NOT EXISTS idx_transactions_log_created_at ON transactions_log(created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_log_user_created ON transactions_log(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_log_reference ON transactions_log(reference_type, reference_id);

CREATE OR REPLACE FUNCTION reject_transactions_log_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'transactions_log is append-only and cannot be updated or deleted';
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_transactions_log_append_only'
    ) THEN
        CREATE TRIGGER trg_transactions_log_append_only
        BEFORE UPDATE OR DELETE ON transactions_log
        FOR EACH ROW
        EXECUTE FUNCTION reject_transactions_log_mutation();
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS ledger_accounts (
    id                   BIGSERIAL PRIMARY KEY,
    account_code         VARCHAR(128) NOT NULL,
    account_name         VARCHAR(128) NOT NULL,
    account_type         VARCHAR(16) NOT NULL,
    normal_balance       VARCHAR(8) NOT NULL,
    owner_type           VARCHAR(16) NOT NULL DEFAULT 'PLATFORM',
    owner_user_id        BIGINT REFERENCES users(id) ON DELETE CASCADE,
    user_bank_account_id BIGINT REFERENCES users_bank_account(id) ON DELETE CASCADE,
    currency             VARCHAR(16) NOT NULL DEFAULT 'USD',
    status               VARCHAR(16) NOT NULL DEFAULT 'ACTIVE',
    metadata             JSONB NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ledger_accounts_code_unique UNIQUE (account_code),
    CONSTRAINT ledger_accounts_type_check CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    CONSTRAINT ledger_accounts_normal_balance_check CHECK (normal_balance IN ('DEBIT', 'CREDIT')),
    CONSTRAINT ledger_accounts_owner_type_check CHECK (owner_type IN ('PLATFORM', 'USER', 'SYSTEM')),
    CONSTRAINT ledger_accounts_status_check CHECK (status IN ('ACTIVE', 'FROZEN', 'CLOSED')),
    CONSTRAINT ledger_accounts_owner_shape_check CHECK (
        (owner_type = 'USER' AND owner_user_id IS NOT NULL)
        OR (owner_type IN ('PLATFORM', 'SYSTEM') AND owner_user_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_ledger_accounts_owner_user_id ON ledger_accounts(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_accounts_bank_account_id ON ledger_accounts(user_bank_account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_accounts_type ON ledger_accounts(account_type);
CREATE INDEX IF NOT EXISTS idx_ledger_accounts_status ON ledger_accounts(status);

CREATE TABLE IF NOT EXISTS ledger_entries (
    id                 BIGSERIAL PRIMARY KEY,
    entry_id           UUID NOT NULL,
    transaction_log_id BIGINT NOT NULL REFERENCES transactions_log(id) ON DELETE CASCADE,
    tx_id              UUID NOT NULL REFERENCES transactions_log(tx_id) ON DELETE CASCADE,
    ledger_account_id  BIGINT NOT NULL REFERENCES ledger_accounts(id) ON DELETE RESTRICT,
    user_id            BIGINT REFERENCES users(id) ON DELETE CASCADE,
    entry_side         VARCHAR(8) NOT NULL,
    amount             NUMERIC(38, 18) NOT NULL,
    business_module    VARCHAR(32) NOT NULL,
    tx_type            VARCHAR(32) NOT NULL,
    reference_type     VARCHAR(64),
    reference_id       VARCHAR(128),
    description        TEXT NOT NULL DEFAULT '',
    metadata           JSONB NOT NULL DEFAULT '{}',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ledger_entries_entry_id_unique UNIQUE (entry_id),
    CONSTRAINT ledger_entries_side_check CHECK (entry_side IN ('DEBIT', 'CREDIT')),
    CONSTRAINT ledger_entries_amount_positive CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_ledger_entries_transaction_log_id ON ledger_entries(transaction_log_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_tx_id ON ledger_entries(tx_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_ledger_account_id ON ledger_entries(ledger_account_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_id ON ledger_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_module_type ON ledger_entries(business_module, tx_type);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_created_at ON ledger_entries(created_at);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_reference ON ledger_entries(reference_type, reference_id);

CREATE OR REPLACE FUNCTION reject_ledger_entries_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'ledger_entries is append-only and cannot be updated or deleted';
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_ledger_entries_append_only'
    ) THEN
        CREATE TRIGGER trg_ledger_entries_append_only
        BEFORE UPDATE OR DELETE ON ledger_entries
        FOR EACH ROW
        EXECUTE FUNCTION reject_ledger_entries_mutation();
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS loans_contract (
    id                BIGSERIAL PRIMARY KEY,
    loan_id           UUID NOT NULL,
    borrower_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lender_type       VARCHAR(16) NOT NULL,
    lender_id         BIGINT REFERENCES users(id) ON DELETE SET NULL,
    principal         NUMERIC(38, 18) NOT NULL,
    interest_rate     NUMERIC(18, 12) NOT NULL,
    accrued_interest  NUMERIC(38, 18) NOT NULL DEFAULT 0,
    repaid_principal  NUMERIC(38, 18) NOT NULL DEFAULT 0,
    repaid_interest   NUMERIC(38, 18) NOT NULL DEFAULT 0,
    status            VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    due_date          TIMESTAMPTZ NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT loans_contract_loan_id_unique UNIQUE (loan_id),
    CONSTRAINT loans_contract_lender_type_check CHECK (lender_type IN ('PLATFORM', 'USER')),
    CONSTRAINT loans_contract_status_check CHECK (status IN ('ACTIVE', 'REPAID', 'DEFAULTED', 'CANCELLED')),
    CONSTRAINT loans_contract_lender_shape_check CHECK (
        (lender_type = 'PLATFORM' AND lender_id IS NULL)
        OR (lender_type = 'USER' AND lender_id IS NOT NULL)
    ),
    CONSTRAINT loans_contract_principal_non_negative CHECK (principal >= 0),
    CONSTRAINT loans_contract_interest_rate_non_negative CHECK (interest_rate >= 0),
    CONSTRAINT loans_contract_accrued_interest_non_negative CHECK (accrued_interest >= 0),
    CONSTRAINT loans_contract_repaid_principal_non_negative CHECK (repaid_principal >= 0),
    CONSTRAINT loans_contract_repaid_interest_non_negative CHECK (repaid_interest >= 0)
);

CREATE INDEX IF NOT EXISTS idx_loans_contract_borrower_id ON loans_contract(borrower_id);
CREATE INDEX IF NOT EXISTS idx_loans_contract_lender ON loans_contract(lender_type, lender_id);
CREATE INDEX IF NOT EXISTS idx_loans_contract_status ON loans_contract(status);
CREATE INDEX IF NOT EXISTS idx_loans_contract_due_date ON loans_contract(due_date);
CREATE INDEX IF NOT EXISTS idx_loans_contract_active_due ON loans_contract(status, due_date);

CREATE TABLE IF NOT EXISTS financial_audit_logs (
    id               BIGSERIAL PRIMARY KEY,
    audit_id         UUID NOT NULL,
    action           VARCHAR(64) NOT NULL,
    actor_type       VARCHAR(16) NOT NULL DEFAULT 'SYSTEM',
    actor_user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    request_id       VARCHAR(128),
    tx_id            UUID REFERENCES transactions_log(tx_id) ON DELETE SET NULL,
    reversal_id      UUID,
    target_type      VARCHAR(64) NOT NULL,
    target_id        VARCHAR(128) NOT NULL,
    metadata         JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT financial_audit_logs_audit_id_unique UNIQUE (audit_id),
    CONSTRAINT financial_audit_logs_actor_type_check CHECK (actor_type IN ('SYSTEM', 'USER', 'ADMIN', 'JOB'))
);

CREATE INDEX IF NOT EXISTS idx_financial_audit_logs_action ON financial_audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_financial_audit_logs_tx_id ON financial_audit_logs(tx_id);
CREATE INDEX IF NOT EXISTS idx_financial_audit_logs_target ON financial_audit_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_financial_audit_logs_created_at ON financial_audit_logs(created_at);

CREATE OR REPLACE FUNCTION reject_financial_audit_logs_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'financial_audit_logs is append-only and cannot be updated or deleted';
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger WHERE tgname = 'trg_financial_audit_logs_append_only'
    ) THEN
        CREATE TRIGGER trg_financial_audit_logs_append_only
        BEFORE UPDATE OR DELETE ON financial_audit_logs
        FOR EACH ROW
        EXECUTE FUNCTION reject_financial_audit_logs_mutation();
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS financial_reconciliation_runs (
    id                     BIGSERIAL PRIMARY KEY,
    run_id                 UUID NOT NULL,
    run_date               DATE NOT NULL,
    status                 VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    checked_transactions   BIGINT NOT NULL DEFAULT 0,
    checked_ledger_entries BIGINT NOT NULL DEFAULT 0,
    mismatch_count         BIGINT NOT NULL DEFAULT 0,
    summary                JSONB NOT NULL DEFAULT '{}',
    started_at             TIMESTAMPTZ,
    finished_at            TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT financial_reconciliation_runs_run_id_unique UNIQUE (run_id),
    CONSTRAINT financial_reconciliation_runs_status_check CHECK (status IN ('PENDING', 'RUNNING', 'COMPLETED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_financial_reconciliation_runs_run_date ON financial_reconciliation_runs(run_date);
CREATE INDEX IF NOT EXISTS idx_financial_reconciliation_runs_status ON financial_reconciliation_runs(status);
CREATE INDEX IF NOT EXISTS idx_financial_reconciliation_runs_created_at ON financial_reconciliation_runs(created_at);

CREATE TABLE IF NOT EXISTS financial_reconciliation_issues (
    id                BIGSERIAL PRIMARY KEY,
    issue_id          UUID NOT NULL,
    reconciliation_id BIGINT NOT NULL REFERENCES financial_reconciliation_runs(id) ON DELETE CASCADE,
    issue_type        VARCHAR(32) NOT NULL,
    tx_id             UUID REFERENCES transactions_log(tx_id) ON DELETE SET NULL,
    ledger_account_id BIGINT REFERENCES ledger_accounts(id) ON DELETE SET NULL,
    expected_amount   NUMERIC(38, 18) NOT NULL DEFAULT 0,
    actual_amount     NUMERIC(38, 18) NOT NULL DEFAULT 0,
    detail            TEXT NOT NULL DEFAULT '',
    metadata          JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT financial_reconciliation_issues_issue_id_unique UNIQUE (issue_id),
    CONSTRAINT financial_reconciliation_issues_type_check CHECK (
        issue_type IN (
            'LEDGER_IMBALANCE',
            'SNAPSHOT_DRIFT',
            'MISSING_LEDGER_ENTRY',
            'MISSING_TRANSACTION',
            'DEBT_MISMATCH'
        )
    )
);

CREATE INDEX IF NOT EXISTS idx_financial_reconciliation_issues_run_id ON financial_reconciliation_issues(reconciliation_id);
CREATE INDEX IF NOT EXISTS idx_financial_reconciliation_issues_type ON financial_reconciliation_issues(issue_type);
CREATE INDEX IF NOT EXISTS idx_financial_reconciliation_issues_tx_id ON financial_reconciliation_issues(tx_id);

CREATE TABLE IF NOT EXISTS financial_reversals (
    id                   BIGSERIAL PRIMARY KEY,
    reversal_id          UUID NOT NULL,
    original_tx_id       UUID NOT NULL REFERENCES transactions_log(tx_id) ON DELETE RESTRICT,
    reversal_tx_id       UUID REFERENCES transactions_log(tx_id) ON DELETE SET NULL,
    requested_by_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    approved_by_user_id  BIGINT REFERENCES users(id) ON DELETE SET NULL,
    request_id           VARCHAR(128),
    reason               TEXT NOT NULL,
    status               VARCHAR(16) NOT NULL DEFAULT 'PENDING',
    metadata             JSONB NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at           TIMESTAMPTZ,
    CONSTRAINT financial_reversals_reversal_id_unique UNIQUE (reversal_id),
    CONSTRAINT financial_reversals_original_tx_unique UNIQUE (original_tx_id),
    CONSTRAINT financial_reversals_status_check CHECK (status IN ('PENDING', 'APPLIED', 'REJECTED', 'FAILED'))
);

CREATE INDEX IF NOT EXISTS idx_financial_reversals_reversal_tx_id ON financial_reversals(reversal_tx_id);
CREATE INDEX IF NOT EXISTS idx_financial_reversals_status ON financial_reversals(status);
CREATE INDEX IF NOT EXISTS idx_financial_reversals_created_at ON financial_reversals(created_at);

CREATE OR REPLACE VIEW ledger_transaction_balances AS
SELECT
    tx_id,
    COUNT(*) AS entry_count,
    COALESCE(SUM(CASE WHEN entry_side = 'DEBIT' THEN amount ELSE 0 END), 0)::NUMERIC(38, 18) AS debit_total,
    COALESCE(SUM(CASE WHEN entry_side = 'CREDIT' THEN amount ELSE 0 END), 0)::NUMERIC(38, 18) AS credit_total,
    (
        COALESCE(SUM(CASE WHEN entry_side = 'DEBIT' THEN amount ELSE 0 END), 0)
        =
        COALESCE(SUM(CASE WHEN entry_side = 'CREDIT' THEN amount ELSE 0 END), 0)
    ) AS balanced
FROM ledger_entries
GROUP BY tx_id;
