-- Lottery entertainment foundation:
-- 1. core lottery tables with lottery_type for future expansion
-- 2. extend existing bank ledger transaction types for lottery bet/payout

ALTER TABLE transactions_log
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
    );

COMMENT ON COLUMN transactions_log.tx_type IS '交易类型：接口扣费、充值、提现、转账、游戏下注、中奖、彩票下注、彩票派奖、借贷、利息、奖励、退款、冻结、解冻等';

CREATE TABLE IF NOT EXISTS lottery_issue (
    id                BIGSERIAL PRIMARY KEY,
    lottery_type      VARCHAR(32) NOT NULL,
    issue_no          VARCHAR(32) NOT NULL,
    open_time         TIMESTAMPTZ NOT NULL,
    status            VARCHAR(16) NOT NULL DEFAULT 'pending',
    result_synced_at  TIMESTAMPTZ,
    settled_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lottery_issue_type_issue_unique UNIQUE (lottery_type, issue_no),
    CONSTRAINT lottery_issue_status_check CHECK (status IN ('pending', 'opened', 'settled'))
);

CREATE INDEX IF NOT EXISTS idx_lottery_issue_open_time ON lottery_issue(open_time);
CREATE INDEX IF NOT EXISTS idx_lottery_issue_type_status_open_time ON lottery_issue(lottery_type, status, open_time DESC);

COMMENT ON TABLE lottery_issue IS '娱乐彩票期号表，按玩法维护每期开奖计划与结算状态';
COMMENT ON COLUMN lottery_issue.lottery_type IS '彩票玩法标识，预留 ssq/dlt/fc3d/pl3/kl8 等扩展';
COMMENT ON COLUMN lottery_issue.issue_no IS '官方期号，例如双色球 2026062';
COMMENT ON COLUMN lottery_issue.open_time IS '计划开奖时间（北京时间对应的绝对时间）';
COMMENT ON COLUMN lottery_issue.result_synced_at IS '官方结果同步入库时间';
COMMENT ON COLUMN lottery_issue.settled_at IS '完成自动结算时间';

CREATE TABLE IF NOT EXISTS lottery_result (
    id              BIGSERIAL PRIMARY KEY,
    lottery_type    VARCHAR(32) NOT NULL,
    issue_no        VARCHAR(32) NOT NULL,
    red_balls       VARCHAR(64) NOT NULL,
    blue_ball       VARCHAR(8) NOT NULL,
    source          VARCHAR(64) NOT NULL,
    source_ref      VARCHAR(255) NOT NULL DEFAULT '',
    source_payload  JSONB NOT NULL DEFAULT '{}',
    opened_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lottery_result_type_issue_unique UNIQUE (lottery_type, issue_no)
);

CREATE INDEX IF NOT EXISTS idx_lottery_result_type_created_at ON lottery_result(lottery_type, created_at DESC);

COMMENT ON TABLE lottery_result IS '娱乐彩票开奖结果表，保存官方同步的原始开奖号码与来源信息';
COMMENT ON COLUMN lottery_result.red_balls IS '逗号分隔的红球号码，已标准化为两位数';
COMMENT ON COLUMN lottery_result.blue_ball IS '两位数蓝球号码';
COMMENT ON COLUMN lottery_result.source IS '数据来源 Provider 标识，例如 fucai';
COMMENT ON COLUMN lottery_result.source_ref IS '来源引用，例如官方详情页 URL';
COMMENT ON COLUMN lottery_result.source_payload IS '来源原始 JSON/字段快照，便于后续审计与兼容';

CREATE TABLE IF NOT EXISTS lottery_order (
    id              BIGSERIAL PRIMARY KEY,
    lottery_type    VARCHAR(32) NOT NULL,
    issue_no        VARCHAR(32) NOT NULL,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    red_balls       VARCHAR(64) NOT NULL,
    blue_ball       VARCHAR(8) NOT NULL,
    cost            NUMERIC(38, 18) NOT NULL DEFAULT 0,
    reward          NUMERIC(38, 18) NOT NULL DEFAULT 0,
    prize_level     VARCHAR(32) NOT NULL DEFAULT '',
    red_hits        INT NOT NULL DEFAULT 0,
    blue_hit        BOOLEAN NOT NULL DEFAULT FALSE,
    status          VARCHAR(16) NOT NULL DEFAULT 'pending',
    settled_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lottery_order_status_check CHECK (status IN ('pending', 'win', 'lose')),
    CONSTRAINT lottery_order_cost_non_negative CHECK (cost >= 0),
    CONSTRAINT lottery_order_reward_non_negative CHECK (reward >= 0),
    CONSTRAINT lottery_order_red_hits_range CHECK (red_hits BETWEEN 0 AND 6)
);

CREATE INDEX IF NOT EXISTS idx_lottery_order_type_issue_created_at ON lottery_order(lottery_type, issue_no, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lottery_order_user_issue ON lottery_order(user_id, lottery_type, issue_no);
CREATE INDEX IF NOT EXISTS idx_lottery_order_type_status_created_at ON lottery_order(lottery_type, status, created_at DESC);

COMMENT ON TABLE lottery_order IS '娱乐彩票投注记录表，当前阶段一行代表一注标准投注';
COMMENT ON COLUMN lottery_order.red_balls IS '用户投注的红球号码，逗号分隔，两位数且升序';
COMMENT ON COLUMN lottery_order.blue_ball IS '用户投注的蓝球号码，两位数';
COMMENT ON COLUMN lottery_order.prize_level IS '中奖档位，未中奖时为空字符串';

CREATE TABLE IF NOT EXISTS lottery_jackpot (
    id            BIGSERIAL PRIMARY KEY,
    lottery_type  VARCHAR(32) NOT NULL,
    balance       NUMERIC(38, 18) NOT NULL DEFAULT 0,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lottery_jackpot_type_unique UNIQUE (lottery_type),
    CONSTRAINT lottery_jackpot_balance_non_negative CHECK (balance >= 0)
);

CREATE INDEX IF NOT EXISTS idx_lottery_jackpot_type ON lottery_jackpot(lottery_type);

COMMENT ON TABLE lottery_jackpot IS '娱乐彩票奖池快照表，记录每个玩法的当前奖池余额';
COMMENT ON COLUMN lottery_jackpot.balance IS '当前奖池余额（DG 币）';

INSERT INTO lottery_jackpot (lottery_type, balance, created_at, updated_at)
VALUES ('ssq', 10000000, NOW(), NOW())
ON CONFLICT (lottery_type) DO NOTHING;

CREATE TABLE IF NOT EXISTS lottery_reward_log (
    id            BIGSERIAL PRIMARY KEY,
    lottery_type  VARCHAR(32) NOT NULL,
    user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issue_no      VARCHAR(32) NOT NULL,
    order_id      BIGINT REFERENCES lottery_order(id) ON DELETE SET NULL,
    reward        NUMERIC(38, 18) NOT NULL DEFAULT 0,
    remark        TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lottery_reward_log_reward_non_negative CHECK (reward >= 0)
);

CREATE INDEX IF NOT EXISTS idx_lottery_reward_log_user_created_at ON lottery_reward_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lottery_reward_log_type_issue ON lottery_reward_log(lottery_type, issue_no);

COMMENT ON TABLE lottery_reward_log IS '娱乐彩票派奖日志表，记录每笔中奖 DG 币发放';
COMMENT ON COLUMN lottery_reward_log.remark IS '派奖说明，包含奖级/期号等信息';
