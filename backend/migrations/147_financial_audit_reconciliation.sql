-- Financial Hub V1 append-only audit, reconciliation, and reversal tables.

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

COMMENT ON TABLE financial_audit_logs IS 'Financial Hub 审计日志，记录关键资金动作、冲正、对账动作的不可变审计轨迹';
COMMENT ON COLUMN financial_audit_logs.audit_id IS '审计事件唯一标识';
COMMENT ON COLUMN financial_audit_logs.tx_id IS '关联的 Financial Hub 交易流水号';
COMMENT ON COLUMN financial_audit_logs.target_type IS '审计目标类型，例如 transaction、loan、reconciliation_run';
COMMENT ON COLUMN financial_audit_logs.target_id IS '审计目标标识';

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

COMMENT ON TABLE financial_reconciliation_runs IS 'Financial Hub 每日自动对账任务主表';
COMMENT ON COLUMN financial_reconciliation_runs.run_id IS '对账批次唯一标识';
COMMENT ON COLUMN financial_reconciliation_runs.summary IS '对账摘要，记录核对统计和异常概览';

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

COMMENT ON TABLE financial_reconciliation_issues IS 'Financial Hub 对账异常明细表';
COMMENT ON COLUMN financial_reconciliation_issues.issue_type IS '对账异常类型';
COMMENT ON COLUMN financial_reconciliation_issues.expected_amount IS '期望值';
COMMENT ON COLUMN financial_reconciliation_issues.actual_amount IS '实际值';

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

COMMENT ON TABLE financial_reversals IS 'Financial Hub 冲正请求与执行记录表';
COMMENT ON COLUMN financial_reversals.original_tx_id IS '被冲正的原始交易流水号';
COMMENT ON COLUMN financial_reversals.reversal_tx_id IS '冲正后生成的新交易流水号';
