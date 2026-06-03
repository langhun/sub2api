# 测试余额依赖审计报告

## 审计范围

- 审计时间：2026-06-04
- 目录范围：`backend/**`
- 重点模式：
  - `SetBalance(...)`
  - `user.Balance =`
  - `user.Balance +=`
  - `user.Balance -=`
  - `Balance:`

## 结论摘要

当前测试体系已经明显向银行账户/账本语义迁移，但还没有彻底摆脱 legacy balance 形态。

本次静态统计结果：

- 直接使用 `SetBalance(...)` 初始化 legacy 余额的测试文件：`7` 个
- 在测试 stub / 假仓储里直接做 `user.Balance += / -= / =` 的文件：`2` 个
- 广义上仍包含 `Balance:` 字段、legacy 断言或兼容镜像断言的测试/fixture 文件：`57` 个

判断：

- 已迁移：主业务测试名称和行为大多已经围绕 `*_bank_*`、`financialhubsdk`、`bank service` 展开
- 待迁移：仍直接操作 legacy balance 的测试夹具和 stub
- 阻塞项：运行时兼容层尚在，导致部分测试必须继续断言 `User.Balance` 镜像

## 一、已迁移

以下文件虽然仍出现 `Balance` 字样，但测试目标已经是银行账户、账本流水或兼容投影后的结果，不再把 legacy balance 当真实资金真相源：

- `backend/internal/repository/admin_balance_bank_integration_test.go`
- `backend/internal/repository/balance_transfer_bank_integration_test.go`
- `backend/internal/repository/usage_billing_repo_integration_test.go`
- `backend/internal/repository/usage_service_bank_integration_test.go`
- `backend/internal/repository/user_bank_balance_mapper_test.go`
- `backend/internal/pkg/financialhubsdk/sdk_test.go`
- `backend/internal/service/bank_transfer_calc_test.go`
- `backend/internal/service/openai_gateway_record_usage_test.go`

说明：

- 这一组测试更多是在验证“银行账户是权威来源”或“legacy projection 是否与银行账户一致”
- 它们不一定需要立刻删除，但默认应该继续向 `BankAccount` / `TransferFundsResult` / `TransactionLog` 断言收口

## 二、待迁移

### 1. 仍直接 `SetBalance(...)` 初始化 legacy 余额

文件数：`7`

文件列表：

- `backend/internal/handler/auth_oauth_pending_flow_test.go`
- `backend/internal/repository/bank_service_integration_test.go`
- `backend/internal/repository/checkin_bank_integration_test.go`
- `backend/internal/repository/fixtures_integration_test.go`
- `backend/internal/repository/payment_refund_bank_integration_test.go`
- `backend/internal/service/auth_service_email_bind_test.go`
- `backend/internal/service/auth_service_identity_sync_test.go`

判断：

- 这些文件大多已经在测银行链路，但初始化方式仍借助 legacy 字段或迁移兼容行为
- 适合在 Phase 1.6 逐步改成“直接创建 `users_bank_account` 测试前置数据”

### 2. 测试 stub 仍直接修改 `user.Balance`

文件数：`2`

文件列表：

- `backend/internal/service/balance_transfer_service_test.go`
- `backend/internal/service/payment_order_lifecycle_test.go`

具体风险：

- `balance_transfer_service_test.go` 中 stub 仍保留 `UpdateBalance(...)` / `DeductBalance(...)`，并直接做 `user.Balance += / -=`
- `payment_order_lifecycle_test.go` 中测试假仓储仍直接在内存对象上累加 `Balance`

判断：

- 这是当前测试层最典型的 legacy 直接变更残留
- 应优先迁移，因为它们最容易把旧接口语义重新带回新测试

## 三、阻塞项

### 1. 兼容投影层尚未下线

文件：

- `backend/internal/service/user_balance_projection.go`
- `backend/internal/repository/api_key_repo.go`
- `backend/internal/service/billing_cache_service.go`

影响：

- 只要运行时还要求 `User.Balance` 作为兼容展示镜像存在
- 测试就仍会出现：
  - `storedUser.Balance`
  - `require.Equal(..., user.Balance)`
  - `Balance:` 结构字段

### 2. `ensureBankAccount(...)` 仍读取 legacy 字段初始化

文件：

- `backend/internal/service/bank_account_store.go`

影响：

- 部分集成测试目前是依赖“旧用户首次建账时从 `users.balance` 带入”的迁移语义
- 在该逻辑删除前，某些测试无法一次性完全切成纯银行账户前置

### 3. `service.User` 结构仍保留 `Balance`

影响：

- 很多 handler / service / middleware 测试仍通过构造 `User{ Balance: ... }` 传递兼容展示值
- 这会让 `Balance:` 命中数量维持在较高水平

## 四、Phase 1.6 建议

### 优先级 P0

先迁移这两类：

1. `balance_transfer_service_test.go`
2. `payment_order_lifecycle_test.go`

原因：

- 它们仍直接做 `user.Balance += / -=`
- 风险高于单纯的只读断言

### 优先级 P1

把以下 7 个 `SetBalance(...)` 初始化文件改成“直接创建银行账户前置”：

- `backend/internal/handler/auth_oauth_pending_flow_test.go`
- `backend/internal/repository/bank_service_integration_test.go`
- `backend/internal/repository/checkin_bank_integration_test.go`
- `backend/internal/repository/fixtures_integration_test.go`
- `backend/internal/repository/payment_refund_bank_integration_test.go`
- `backend/internal/service/auth_service_email_bind_test.go`
- `backend/internal/service/auth_service_identity_sync_test.go`

### 优先级 P2

最后再清理广义 `Balance:` 命中：

- 默认改成断言 `BankAccount.Balance`
- 仅在明确测试 compatibility projection 时保留 `User.Balance` 断言

## 五、当前状态判定

可以把测试体系的现状描述为：

```text
主行为已迁移到 Bank / Ledger
测试夹具仍部分依赖 Legacy Balance
兼容投影尚未完全退出
```

这意味着：

- Phase 1.5 已经把生产主链路基本收口
- 但测试基建还需要一轮 Phase 1.6 的系统清理

## 最终结论

### 已迁移

- 账本服务、银行账户、API 扣费和大部分银行集成测试已围绕新模型运行

### 待迁移

- `7` 个文件仍直接 `SetBalance(...)`
- `2` 个文件仍直接修改 `user.Balance`

### 阻塞项

- `User.Balance` compatibility projection 仍存在
- `ensureBankAccount(...)` 仍保留 legacy 初始化
- `service.User` 结构仍广泛带有 `Balance` 兼容字段

因此，本报告建议把测试清理定义为：

```text
Phase 1.6 必做项
但不阻塞 Phase 1.5 收尾完成
```
