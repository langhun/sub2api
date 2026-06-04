# Financial Hub 资金场景覆盖报告

## 审计范围

- 审计时间：2026-06-04
- 目标：核对所有资金场景是否已经统一收口到 Financial Hub / BankService / Ledger
- 冻结边界：遵循 Architecture Freeze v1.1，不新增 P2P 高级撮合、自动投资、债权转让、保证金、杠杆、自动清算、分布式账本
- 结论口径：
  - 已接入：后端资金变动已经通过 `BankService` 写入 `transactions_log` 和 `ledger_entries`
  - 部分接入：核心资金写入已账本化，但合同、SDK、前端心智或审计能力仍有缺口
  - 延期：冻结令明确推迟到 V2，当前只保留底层兼容类型或入口审计

## 总体结论

Financial Hub 不是“全部完成”。

当前可以确认：

1. `/finance` 页面与 `/api/v1/finance/account`、`/api/v1/finance/transactions` 已经作为只读金融中心入口可用。
2. Legacy `users.balance` 已退出真实资金来源，运行时真实资金写入围绕 `users_bank_account`、`transactions_log`、`ledger_entries`。
3. API 消费、充值到账、转账、红包、签到奖励、老虎机、彩票娱乐、退款扣回、返佣提取等多条已有业务链路已经接入 `BankService`。
4. 本次已补齐两个 P1 缺口：
   - `LOTTERY_BET` / `LOTTERY_WIN` 已同步到 Proto、OpenAPI、DDL 合同。
   - `LOAN_REPAY` 总账分录已按“先息后本”拆分冲减应收利息和应收本金。

仍需继续处理：

1. 贷款合同创建、冲正、对账、审计日志在 Financial Hub SDK 合同中仍有 `NotImplemented` 适配器。
2. `business_module` 目前主要是审计标签，还没有和 `tx_type` 做严格兼容矩阵校验。
3. 幂等冲突比对还没有覆盖会影响分录路由的 metadata，例如彩票分摊和转账退款来源。
4. 多个前端页面仍使用“余额”心智展示，虽然后端已账本化，但用户认知还没有完全收口到“金融中心流水为准”。
5. 完整 P2P 放贷、投资撮合、收益市场属于 V2 延期内容，本阶段不得主动实现。

## 场景覆盖矩阵

| 资金场景 | 当前状态 | 后端账本入口 | 当前问题 | 下一步 |
| --- | --- | --- | --- | --- |
| API 消费 | 已接入 | `usage_service.go`、`usage_billing_repo.go` 使用 `BankTxTypeConsume` | 仍需高并发账本一致性压测 | Phase 2 做并发扣费压测与 reconciliation 验证 |
| 充值 | 已接入 | `redeem_service.go`、`promo_service.go`、`payment_fulfillment.go` 通过 `REWARD/DEPOSIT` 类账本入账 | 前端支付结果页未强引导到 `/finance` 查看流水 | 增加金融中心查看入口和文案收口 |
| 提现 | 部分接入 | `BankTxTypeWithdraw` 已存在，退款扣回、运气签到亏损、兑换码负数等使用该类型 | 尚未看到独立用户提现产品闭环 | V1 若开放提现，需要独立 API、审核、审计和幂等 |
| 转账 | 已接入 | `balance_transfer_service.go` 使用 `TRANSFER_OUT/TRANSFER_IN/WITHDRAW/REFUND` | 前端仍展示旧“当前余额”心智 | 文案改为“账户可用余额，以金融中心流水为准” |
| 红包 | 部分接入 | `balance_transfer_service.go` 红包创建、领取、过期退款走 BankService | 无专用 `RED_PACKET_*` tx type，业务语义靠 reference/metadata | Phase 1.6 评估是否补专用类型，或强化 reference 查询 |
| 签到奖励 | 已接入 | `checkin_service.go` 使用 `REWARD/WITHDRAW` | 前端 store/page 仍引用 `balance` 展示 | 文案和展示层收口到账户可用余额 |
| 槽机 /老虎机 | 已接入 | `game_service.go` 使用 `SLOT_BET/SLOT_WIN` 和 `TransferFundsBatch` | `/games` 页面实际接的是 lottery API，入口语义错位 | 先修入口命名或页面接线，避免“娱乐大厅”误导 |
| 娱乐大厅 /彩票 | 已接入并已补合同 | `lottery_bet_flow.go`、`lottery_settlement.go` 使用 `LOTTERY_BET/LOTTERY_WIN` | 本次前合同缺失已修；metadata 幂等比对仍需补 | Phase 1.6 补 metadata 幂等冲突检测 |
| 借贷 | 部分接入 | `LOAN_BORROW/LOAN_REPAY/LOAN_INTEREST` 计算和分录已存在 | SDK `CreateLoanContract` 仍未实现；合同与还款 API 未形成完整闭环 | P1 继续补平台贷款 API、合同、计息 job |
| P2P 放贷 | 延期 / 底层类型存在 | `LEND_INVEST/LEND_PROFIT/FREEZE/UNFREEZE` 类型和分录存在 | 完整 P2P 撮合、投资池、高级撮合按冻结令延期到 V2 | 仅保留类型，不实现撮合和市场 |
| 利息 | 部分接入 | `LOAN_INTEREST` 增加债务利息，应收利息和利息收入分录存在 | 计息 job、合同维度、借款人审计维度仍需补齐 | Phase 2/Loan 工作继续补 cron 和审计 |
| 退款 | 部分接入 | `payment_refund.go` 使用 `WITHDRAW/REFUND`，转账退款可走 clearing | `REFUND` 更像补偿入账，不是通用 reversal | 冲正能力单独通过 `financial_reversals` 落地 |
| 补偿 | 部分接入 | 可通过 `REWARD/REFUND` 表达 | 无专用 `COMPENSATION` 类型，审计语义不够清楚 | Phase 1.6 决定是否补专用类型或 reference 规范 |
| 分销返佣 | 已接入 | `affiliate_repo.go` 使用 `BankTxTypeReward` 将返佣额度转入账户 | 前端仍写“转入余额”，且无专用 `AFFILIATE_REBATE` tx type | 文案收口；后续评估专用类型 |
| 后续新增业务 | 规则已明确 | 必须通过 `financialhubsdk` 或 `BankService.TransferFunds` | 需要代码层 lint/审计规则阻止直接改余额 | 补静态扫描和 PR 检查规则 |

## 本次已处理的 P1 缺口

### 1. 彩票交易类型合同同步

已同步文件：

- `api/proto/financial_hub.proto`
- `api/financial_hub_v1.openapi.yaml`
- `api/ddl/financial_hub_v1.sql`

处理结果：

- `LOTTERY_BET`
- `LOTTERY_WIN`

已经和后端 `BankTxTypeLotteryBet` / `BankTxTypeLotteryWin` 对齐。

### 2. 贷款还款分录拆分

已修改文件：

- `backend/internal/service/bank_ledger.go`
- `backend/internal/service/bank_ledger_test.go`

处理结果：

- `LOAN_REPAY` 现在会根据还款后快照计算本次实际还息和还本。
- 还息部分贷记 `PLATFORM:RECEIVABLE:LOAN_INTEREST`。
- 还本部分贷记 `PLATFORM:RECEIVABLE:LOAN_PRINCIPAL`。
- 用户侧仍借记 `USER:{id}:BALANCE`。
- 新增测试覆盖“先息后本”的三分录场景。

## 剩余风险

### P1

1. 贷款合同 API 和计息 job 仍未形成完整闭环。
2. Financial Hub SDK 的贷款合同、冲正、对账、审计查询仍存在 `ErrNotImplemented`。

### P2

1. `business_module` 与 `tx_type` 未做严格兼容矩阵校验。
2. 幂等重放比对未覆盖影响分录路由的 metadata：
   - `refund_source`
   - `jackpot_amount`
   - `burn_amount`
   - `platform_amount`
3. 红包、补偿、返佣目前复用通用类型，审计语义依赖 reference/metadata。
4. 前端多个业务页面仍以“余额”文案为主，需要收口到“账户余额 / 金融中心流水”。

### V2 延期

以下内容保持延期，不在本阶段实现：

- P2P 高级撮合
- 自动投资
- 债权转让
- 保证金
- 杠杆
- 自动清算
- 分布式账本
- 收益市场

## 建议验收口径

Phase 1.5 / Phase 2 后续验收不要再只问“金融中心好了没”，应按以下检查：

1. 每个资金场景是否有唯一 `idempotency_key`。
2. 每个资金场景是否写入 `transactions_log`。
3. 每个 `transactions_log.tx_id` 是否有平衡的 `ledger_entries`。
4. 是否存在绕过 Financial Hub 的 `users.balance` 写入。
5. 前端是否明确提示“资金结果以金融中心流水为准”。
6. 每个新增业务是否只调用 Financial Hub SDK，而不是自行修改余额。
