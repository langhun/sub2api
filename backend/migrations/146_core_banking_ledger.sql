-- Core Banking 双重记账基础：扩展用户账户债务字段，并新增总账账户与总账分录。

-- 新目标允许授信透支，因此银行账户可用余额允许为负数。
ALTER TABLE users_bank_account
    DROP CONSTRAINT IF EXISTS users_bank_account_balance_non_negative;

ALTER TABLE users_bank_account
    ADD COLUMN IF NOT EXISTS debt_principal NUMERIC(38, 18) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS debt_interest  NUMERIC(38, 18) NOT NULL DEFAULT 0;

UPDATE users_bank_account
SET debt_principal = total_debt
WHERE debt_principal = 0
  AND total_debt > 0;

ALTER TABLE users_bank_account
    DROP CONSTRAINT IF EXISTS users_bank_account_debt_principal_non_negative,
    DROP CONSTRAINT IF EXISTS users_bank_account_debt_interest_non_negative;

ALTER TABLE users_bank_account
    ADD CONSTRAINT users_bank_account_debt_principal_non_negative CHECK (debt_principal >= 0),
    ADD CONSTRAINT users_bank_account_debt_interest_non_negative CHECK (debt_interest >= 0);

COMMENT ON COLUMN users_bank_account.balance IS '可用余额，允许在授信额度内为负；所有变动必须由 Financial Hub 写入';
COMMENT ON COLUMN users_bank_account.debt_principal IS '待还本金，用于记录授信透支、借贷本金等本金类债务';
COMMENT ON COLUMN users_bank_account.debt_interest IS '待还利息，用于记录贷款、逾期等利息类债务';
COMMENT ON COLUMN users_bank_account.total_debt IS '当前总负债兼容字段，后续由待还本金与待还利息汇总维护';

-- 统一流水表增加业务模块字段，并扩展全站资金场景类型。
ALTER TABLE transactions_log
    ADD COLUMN IF NOT EXISTS business_module VARCHAR(32) NOT NULL DEFAULT 'FINANCIAL_HUB';

ALTER TABLE transactions_log
    DROP CONSTRAINT IF EXISTS transactions_log_balance_before_non_negative,
    DROP CONSTRAINT IF EXISTS transactions_log_balance_after_non_negative,
    DROP CONSTRAINT IF EXISTS transactions_log_tx_type_check;

ALTER TABLE transactions_log
    ADD CONSTRAINT transactions_log_tx_type_check CHECK (
        tx_type IN (
            'CONSUME',
            'DEPOSIT',
            'WITHDRAW',
            'TRANSFER_OUT',
            'TRANSFER_IN',
            'SLOT_BET',
            'SLOT_WIN',
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
    );

CREATE INDEX IF NOT EXISTS idx_transactions_log_business_module ON transactions_log(business_module);
CREATE INDEX IF NOT EXISTS idx_transactions_log_module_type ON transactions_log(business_module, tx_type);

COMMENT ON COLUMN transactions_log.business_module IS '业务模块：API调用中心、游戏中心、借贷中心、支付中心、系统后台等';
COMMENT ON COLUMN transactions_log.tx_type IS '交易类型：接口扣费、充值、提现、转账、游戏下注、中奖、借贷、利息、奖励、退款、冻结、解冻等';
COMMENT ON COLUMN transactions_log.amount IS '本次用户账单变动金额，收入为正，支出为负；总账借贷方向以 ledger_entries 为准';

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

COMMENT ON TABLE ledger_accounts IS 'Core Banking 总账账户科目表，定义平台、系统与用户维度的资产、负债、收入、费用、权益账户';
COMMENT ON COLUMN ledger_accounts.account_code IS '总账账户编码，全平台唯一，例如 PLATFORM:REVENUE:API 或 USER:123:CASH';
COMMENT ON COLUMN ledger_accounts.account_type IS '会计科目类型：资产、负债、权益、收入、费用';
COMMENT ON COLUMN ledger_accounts.normal_balance IS '科目正常余额方向：借方或贷方';
COMMENT ON COLUMN ledger_accounts.owner_type IS '账户归属类型：平台、用户或系统';

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

COMMENT ON TABLE ledger_entries IS 'Core Banking 双重记账分录表，每笔交易至少包含借贷两条分录且金额必须平衡';
COMMENT ON COLUMN ledger_entries.entry_id IS '总账分录号，全局唯一';
COMMENT ON COLUMN ledger_entries.tx_id IS '关联 transactions_log.tx_id，作为同一交易的凭证号';
COMMENT ON COLUMN ledger_entries.entry_side IS '分录方向：DEBIT 借方，CREDIT 贷方';
COMMENT ON COLUMN ledger_entries.amount IS '分录金额，必须为正数，借贷方向由 entry_side 表达';

CREATE OR REPLACE FUNCTION reject_ledger_entries_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'ledger_entries is append-only and cannot be updated or deleted';
END;
$$ LANGUAGE plpgsql;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_trigger
        WHERE tgname = 'trg_ledger_entries_append_only'
    ) THEN
        CREATE TRIGGER trg_ledger_entries_append_only
        BEFORE UPDATE OR DELETE ON ledger_entries
        FOR EACH ROW
        EXECUTE FUNCTION reject_ledger_entries_mutation();
    END IF;
END $$;

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

COMMENT ON VIEW ledger_transaction_balances IS '总账交易平衡审计视图，用于核验每个 tx_id 的借方合计是否等于贷方合计';
