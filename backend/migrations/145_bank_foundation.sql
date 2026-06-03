-- 虚拟银行基础表：银行账户、不可变账本流水、借贷合约。
CREATE TABLE IF NOT EXISTS users_bank_account (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    balance         NUMERIC(38, 18) NOT NULL DEFAULT 0,
    frozen_amount   NUMERIC(38, 18) NOT NULL DEFAULT 0,
    credit_limit    NUMERIC(38, 18) NOT NULL DEFAULT 0,
    total_debt      NUMERIC(38, 18) NOT NULL DEFAULT 0,
    version         BIGINT NOT NULL DEFAULT 1,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT users_bank_account_user_id_unique UNIQUE (user_id),
    CONSTRAINT users_bank_account_balance_non_negative CHECK (balance >= 0),
    CONSTRAINT users_bank_account_frozen_non_negative CHECK (frozen_amount >= 0),
    CONSTRAINT users_bank_account_credit_limit_non_negative CHECK (credit_limit >= 0),
    CONSTRAINT users_bank_account_total_debt_non_negative CHECK (total_debt >= 0),
    CONSTRAINT users_bank_account_version_positive CHECK (version > 0),
    CONSTRAINT users_bank_account_status_check CHECK (status IN ('ACTIVE', 'FROZEN', 'CLOSED'))
);

CREATE INDEX IF NOT EXISTS idx_users_bank_account_user_id ON users_bank_account(user_id);
CREATE INDEX IF NOT EXISTS idx_users_bank_account_status ON users_bank_account(status);

COMMENT ON TABLE users_bank_account IS '虚拟银行账户表，保存用户可用余额、冻结金额、信用额度和总负债快照';
COMMENT ON COLUMN users_bank_account.balance IS '可用余额，所有变动必须由账本服务写入';
COMMENT ON COLUMN users_bank_account.frozen_amount IS '冻结金额，用于放贷资金池、担保或风控冻结';
COMMENT ON COLUMN users_bank_account.credit_limit IS '信用额度上限';
COMMENT ON COLUMN users_bank_account.total_debt IS '当前总负债';
COMMENT ON COLUMN users_bank_account.version IS '乐观锁版本号，供后续并发控制使用';

INSERT INTO users_bank_account (user_id, balance, created_at, updated_at)
SELECT id, COALESCE(balance, 0), NOW(), NOW()
FROM users
WHERE deleted_at IS NULL
ON CONFLICT (user_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS transactions_log (
    id                    BIGSERIAL PRIMARY KEY,
    tx_id                 UUID NOT NULL,
    user_id               BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id            BIGINT NOT NULL REFERENCES users_bank_account(id) ON DELETE CASCADE,
    tx_type               VARCHAR(32) NOT NULL,
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
            'LOAN_BORROW',
            'LOAN_REPAY',
            'LEND_INVEST',
            'LEND_PROFIT',
            'FREEZE',
            'UNFREEZE'
        )
    ),
    CONSTRAINT transactions_log_balance_before_non_negative CHECK (balance_before >= 0),
    CONSTRAINT transactions_log_balance_after_non_negative CHECK (balance_after >= 0),
    CONSTRAINT transactions_log_frozen_before_non_negative CHECK (frozen_before >= 0),
    CONSTRAINT transactions_log_frozen_after_non_negative CHECK (frozen_after >= 0),
    CONSTRAINT transactions_log_credit_limit_snapshot_non_negative CHECK (credit_limit_snapshot >= 0),
    CONSTRAINT transactions_log_debt_snapshot_non_negative CHECK (debt_snapshot >= 0)
);

CREATE INDEX IF NOT EXISTS idx_transactions_log_user_id ON transactions_log(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_log_account_id ON transactions_log(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_log_tx_type ON transactions_log(tx_type);
CREATE INDEX IF NOT EXISTS idx_transactions_log_created_at ON transactions_log(created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_log_user_created ON transactions_log(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_transactions_log_reference ON transactions_log(reference_type, reference_id);

COMMENT ON TABLE transactions_log IS '虚拟银行不可变账本流水，记录每次资金变动及变动后快照';
COMMENT ON COLUMN transactions_log.tx_id IS '业务流水号，单笔财务操作唯一';
COMMENT ON COLUMN transactions_log.tx_type IS '流水类型：消费、充值、借款、还款、放贷投入、放贷收益、冻结、解冻';
COMMENT ON COLUMN transactions_log.amount IS '本次变动金额，收入为正，支出为负';
COMMENT ON COLUMN transactions_log.balance_after IS '本次变动后的可用余额';
COMMENT ON COLUMN transactions_log.idempotency_scope IS '幂等作用域';
COMMENT ON COLUMN transactions_log.idempotency_key_hash IS '幂等键哈希，用于防止重复入账';

CREATE OR REPLACE FUNCTION reject_transactions_log_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'transactions_log is append-only and cannot be updated or deleted';
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_transactions_log_append_only'
    ) THEN
        CREATE TRIGGER trg_transactions_log_append_only
        BEFORE UPDATE OR DELETE ON transactions_log
        FOR EACH ROW
        EXECUTE FUNCTION reject_transactions_log_mutation();
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

COMMENT ON TABLE loans_contract IS '虚拟银行借贷合约表，支持平台贷款和用户放贷';
COMMENT ON COLUMN loans_contract.loan_id IS '业务合约号';
COMMENT ON COLUMN loans_contract.borrower_id IS '借款人用户 ID';
COMMENT ON COLUMN loans_contract.lender_type IS '放贷方类型：PLATFORM 表示平台，USER 表示用户';
COMMENT ON COLUMN loans_contract.lender_id IS '用户放贷时的放贷人用户 ID，平台放贷时为空';
COMMENT ON COLUMN loans_contract.principal IS '借款本金';
COMMENT ON COLUMN loans_contract.interest_rate IS '日利率';
COMMENT ON COLUMN loans_contract.accrued_interest IS '已累计未还利息';
