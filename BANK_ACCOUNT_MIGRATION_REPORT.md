# ensureBankAccount 迁移审计报告

## 审计范围

- 审计时间：2026-06-04
- 目标函数：`ensureBankAccount(...)`
- 相关调用点：
  - `backend/internal/service/bank_service.go`
  - `backend/internal/service/bank_account_store.go`

## 核心结论

结论：`ensureBankAccount(...)` 当前属于 **LOW RISK**。

它仍然会读取一次 `users.balance`，但语义是“老用户缺失银行账户时的一次性迁移初始化”，不是运行时真实资金来源，也不是消费、充值、调账时的余额真相源。

## 证据

### 1. 实现逻辑只做缺账户补建，不做余额扣改

文件：

- `backend/internal/service/bank_account_store.go`

关键行为：

- 函数内部执行：
  - `INSERT INTO users_bank_account (...)`
  - `SELECT id, COALESCE(balance, 0), ... FROM users`
  - `ON CONFLICT (user_id) DO NOTHING`

判断：

- 该 SQL 只在 `users_bank_account` 缺行时写入
- 已存在银行账户时不会覆盖
- 不会把后续账本余额再回写到 `users.balance`
- 不会直接修改 `users.balance`

### 2. 运行时真实读写已经切到银行账户

文件：

- `backend/internal/service/bank_service.go`
- `backend/internal/service/bank_account_store.go`

关键行为：

- `GetAccountView(...)` 先 `ensureBankAccount(...)`，随后读取 `users_bank_account`
- `lockBankAccountForUpdate(...)` 若查不到账户，会先 `ensureBankAccount(...)`，随后 `SELECT ... FOR UPDATE`
- `updateBankAccount(...)` 只更新 `users_bank_account`
- `createBankTransaction(...)` 记录 `transactions_log`

判断：

- 正常运行中的资金变动、并发扣费、账本流水、账户快照都围绕 `users_bank_account`
- `users.balance` 没有参与后续的真实扣减和结算

### 3. 注释与调用边界已经明确标注为迁移兼容

文件：

- `backend/internal/service/bank_account_store.go`
- `backend/internal/service/bank_service.go`

注释含义：

- `ensureBankAccount` 注释明确说明“仅在旧用户缺少银行账户时读取 users.balance 初始化，不做任何余额扣改”
- `GetAccountView` 注释说明“旧用户首次读取时会从 users.balance 初始化银行账户”

判断：

- 这说明当前仓库已经把它定义成迁移兼容，而不是核心账务设计的一部分

## 风险分级

### LOW RISK 的原因

1. 它不是稳定态余额来源
2. 它不会覆盖已存在的银行账户
3. 它不会参与后续每一笔资金计算
4. 真正的事务更新已经统一写入 `users_bank_account + transactions_log + ledger`

### 仍需注意的剩余风险

虽然结论是 LOW RISK，但仍有一个迁移尾巴：

- 如果某个历史用户一直没有创建 `users_bank_account`
- 且其 `users.balance` 与目标迁移值不一致
- 那么首次触发 `ensureBankAccount(...)` 时，仍会把这个 legacy 值带入新账户

这属于 **历史数据初始化风险**，不是 **运行时账本一致性风险**。

## 是否仍把 `users.balance` 当作运行时资金来源

结论：**否**。

更精确地说：

- 它仍是“建账迁移输入”
- 不是“后续真实余额来源”
- 不是“扣费时的可用余额判断来源”
- 不是“调账时的目标余额来源”

## Phase 1.6 建议

### 建议 1：补齐缺失银行账户

在进入 Phase 2 前，优先做一次全量巡检：

- 找出仍缺少 `users_bank_account` 的活跃用户
- 先离线补齐，再让运行时不再承担迁移职责

### 建议 2：把自动补建改为显式异常

当全量数据迁移确认完成后：

- 删除 `ensureBankAccount(...)` 中对 `users.balance` 的读取
- 缺失账户时直接报错并进入补偿/修复流程

### 建议 3：补一条审计 SQL

建议后续加一条运维审计 SQL 或后台检查：

- 统计 `users` 中有效用户与 `users_bank_account` 的缺口
- 作为删除 legacy 初始化逻辑前的验收依据

## 最终结论

`ensureBankAccount(...)` 当前不应判定为 HIGH RISK。

它的风险等级更准确地说是：

```text
LOW RISK
迁移兼容逻辑仍在
运行时真实资金来源已切换
```

因此，Phase 1.5 可以认定为：

- `users.balance` 已退出运行时资金真相源
- `ensureBankAccount(...)` 只剩迁移尾巴待清理
- 这项工作应进入 Phase 1.6 收尾，而不是回退当前架构
