# Legacy Balance 最终审计报告

## 审计范围

- 审计时间：2026-06-04
- 仓库根目录：`D:/CodeSpace/sub2api`
- 审计方式：基于当前工作区代码搜索结果进行静态审计，不回退、不覆盖其他人的改动
- 目标：回答 Phase 1.5 夜间计划中关于 Legacy Balance 的三个问题：
  - 哪些路径已经删除或已经失效
  - 哪些兼容残留仍然存在
  - Phase 1.6 应该怎样继续收口

## 结论摘要

Phase 1.5 可以判定为“资金真相源迁移已经完成，但兼容展示层尚未彻底删除”。

当前仓库中：

1. `users.balance` 已不再是扣费、充值、准入、账本更新的权威来源。
2. 运行时真实余额已切换到 `users_bank_account` 与银行账本事务日志。
3. `users.balance` 仍作为旧字段镜像保留，主要用于：
   - 老用户首建银行账户时的一次性初始化
   - 部分返回对象中的兼容展示字段
   - 若干测试对迁移期行为的断言

换句话说，Legacy Balance 现在是“兼容层”，不再是“业务真相源”。

## 一、已删除 / 已失效路径

以下路径虽然名字、结构或接口上还保留了 `balance` 概念，但已经不再允许把 `users.balance` 当作真实资金源使用。

### 1. 直接从 `users.balance` 回读真实余额的运行时路径已被硬阻断

证据：

- `backend/internal/service/user_service.go`
  - 定义了 `ErrLegacyBalanceMutationDisabled`
  - 错误文案明确为：legacy user balance mutation is disabled; use bank service
- `backend/internal/service/billing_cache_service.go`
  - `getUserBalanceFromDB(...)` 固定返回 `ErrLegacyBalanceMutationDisabled`
  - `GetUserBalance(...)` 即使 Redis miss，也不会再从 `users.balance` 取真实余额
- `backend/internal/service/billing_cache_service_singleflight_test.go`
  - 明确断言：
    - `balance cache must not load users.balance`
    - `legacy balance must not be cached`

结论：

- 旧的“余额缓存 miss 后回源 `users.balance`”逻辑已实质删除。
- 即使代码路径还存在函数名，行为上已经被禁用。

### 2. 直接修改 `users.balance` 的主业务路径已被移除

证据：

- `backend/internal/service/promo_service.go`
  - 注释明确：优惠码余额奖励必须通过银行账本，禁止直接修改 `users.balance`
- `backend/internal/service/usage_service.go`
  - 注释明确：API 使用扣费必须走银行账本，不能直接修改 `users.balance`
- `backend/internal/service/bank_account_store.go`
  - `updateBankAccount(...)` 注释明确：只更新银行账户表，禁止直接修改 `users.balance`

结论：

- 充值、扣费、奖励、消费等核心资金写路径已经从 legacy 字段迁出。
- 资金变动真相源已经切到银行账户与交易流水。

### 3. 管理端“修改用户资料时顺手改 balance”已经被冻结

证据：

- `backend/internal/service/admin_service.go`
  - `UpdateUser(...)` 中如果 `input.Balance != nil`，直接返回 `ErrAdminUserBalanceFieldFrozen`
- `backend/internal/handler/admin/user_handler.go`
  - 更新用户资料接口中若提交 `req.Balance`，直接拒绝
- 测试：
  - `backend/internal/service/admin_service_update_user_rpm_test.go`
  - `backend/internal/handler/admin/admin_basic_handlers_test.go`
  - 都有 `RejectsLegacyBalanceField` 断言

结论：

- “编辑用户资料时直接写旧余额字段”的管理面入口已被关闭。

## 二、仍存在的 Legacy Balance 残留

这些残留不代表资金逻辑还依赖 `users.balance`，但说明兼容层还没有完全删干净。

### 1. `User.Balance` 兼容展示字段仍然存在

证据：

- `backend/internal/service/user_balance_projection.go`
  - `UpdateUserBalanceProjection(...)` 仍会写入 `user.Balance`
  - 文件注释已把该文件定位为：
    - Legacy Compatibility Layer
    - 旧接口 / 旧页面 / 旧测试的展示兼容层

结论：

- `User.Balance` 还在，但语义已经变成“展示镜像”。
- 它不应该再被新业务当作资金真相源消费。

### 2. 读取用户对象时，仍会把银行账户余额投影回 `user.Balance`

证据：

- `backend/internal/repository/api_key_repo.go`
  - `userEntityToService(...)` 先把 `u.Balance` 放入 `service.User`
  - 若 `u.Edges.BankAccount != nil`，再调用 `UpdateUserBalanceProjectionFromBankAccount(...)`
- `backend/internal/repository/user_bank_balance_mapper_test.go`
  - `TestUserEntityToServiceUsesBankAccountBalance`
  - `TestUserEntityToServiceKeepsLegacyBalanceWithoutBankAccount`

结论：

- 当前仓储层行为是：
  - 有银行账户时，`user.Balance` 仅作为银行余额镜像返回
  - 无银行账户时，仍回落到 legacy 字段

这说明兼容逻辑仍然在线。

### 3. 老用户缺少银行账户时，仍会读取 `users.balance` 做一次性初始化

证据：

- `backend/internal/service/bank_account_store.go`
  - `ensureBankAccount(...)` 使用：
    - `INSERT INTO users_bank_account (...) SELECT id, COALESCE(balance, 0) ... FROM users`
  - 注释明确：
    - 仅在旧用户缺少银行账户时读取 `users.balance` 初始化
    - 不做任何余额扣改

结论：

- 这是当前最关键的 legacy 读残留。
- 它不参与后续资金流转，但仍是老数据迁移兜底逻辑的一部分。

### 4. 旧字段仍被用于排序 / 结构字段 / 兼容测试

证据：

- `backend/internal/repository/user_repo.go`
  - 用户列表排序仍支持 `dbuser.FieldBalance`
- `backend/internal/service/admin_service.go`
  - 创建用户时 `User{ Balance: input.Balance }` 结构字段仍保留
- 多个集成测试 / 单测仍构造 `service.User{ Balance: ... }` 或断言 legacy 值

结论：

- 这些地方多数属于兼容结构和测试基建残留，不代表运行时还依赖旧余额做结算。
- 但它们会拖慢 Phase 1.6 的彻底清理。

## 三、当前可以明确判定“已迁出真相源”的模块

### 1. 扣费准入

证据：

- `backend/internal/server/middleware/api_key_auth.go`
  - 注释明确：API 入口只按银行账户判断容量与冻结状态；旧 `users.balance` 不再作为准入依据
- `backend/internal/service/billing_cache_service.go`
  - `checkBalanceEligibility(...)` 在缺少 `user.BankAccount` 时，会加载银行账户并投影回 `user.Balance`
  - 真正的准入判断基于 `account.CanConsume(...)`

结论：

- API 鉴权与可用余额判断已经不再依赖 legacy balance。

### 2. 账本资金变更

证据：

- `backend/internal/service/bank_account_store.go`
  - 实际余额变化只写 `users_bank_account`
  - 并记录 `transaction_logs`
- `backend/internal/service/admin_service.go`
  - 管理员调余额后只拿 `TransferFundsResult` 回写展示镜像

结论：

- 真正的加减款来源已经统一到银行账本。

### 3. 注册赠送 / OAuth 首次授予

证据：

- `backend/internal/service/auth_signup_balance_grant.go`
- `backend/internal/service/admin_signup_balance_grant.go`
- `backend/internal/service/auth_service.go`
- `backend/internal/service/auth_oauth_email_flow.go`

这些流程里：

- 真正的余额授予通过银行账本完成
- `UpdateUserBalanceProjection*` 只在返回对象层做兼容镜像

结论：

- 注册和首次绑定流程已经是“账本写入 + 兼容投影”模式，而不是直接改旧字段。

## 四、Phase 1.5 判定

基于当前代码状态，Phase 1.5 可以这样验收：

### 已完成部分

- 已切断 `users.balance` 作为真实余额来源的主业务读路径
- 已切断大部分直接写 `users.balance` 的主业务写路径
- 已把资金变更主逻辑迁到银行账户与交易流水
- 已把管理端资料更新中的 legacy balance 写入口冻结
- 已建立明确的兼容层边界：`user_balance_projection.go`

### 未完成部分

- 老用户缺失银行账户时，仍会从 `users.balance` 初始化
- 返回对象层仍持续维护 `user.Balance` 兼容镜像
- 列表排序、结构字段、测试夹具仍大量保留 legacy balance 形态

结论：

Phase 1.5 不应表述为“Legacy Balance 已完全删除”，更准确的说法是：

> Legacy Balance 已退出资金真相源角色，但兼容展示层与迁移兜底层仍存在。

## 五、Phase 1.6 后续计划

建议把 Phase 1.6 拆成三个层次推进。

### A. 先清理运行时兼容读取

优先级：最高

建议动作：

1. 盘点所有 API / DTO / ViewModel 中 `user.Balance` 的实际消费者
2. 能直接改为 `BankAccount.Balance` 或专门余额字段的地方，逐步替换
3. 删除“无银行账户时继续回退 legacy balance”的返回层逻辑

完成标志：

- `userEntityToService(...)` 不再依赖 `u.Balance` 回退
- `UpdateUserBalanceProjectionFromBankAccount(...)` 的调用点显著减少

### B. 下线老用户初始化兜底

优先级：最高

建议动作：

1. 补做一次性迁移或启动前巡检，确保所有有效用户都已有银行账户
2. 在确认线上数据完成迁移后，删除 `ensureBankAccount(...)` 中对 `users.balance` 的读取初始化
3. 把“缺银行账户”从自动补建改成显式异常，避免旧字段继续参与系统行为

完成标志：

- `ensureBankAccount(...)` 不再读取 `users.balance`
- 旧余额字段完全退出运行时数据修复链

### C. 收缩结构字段与测试残留

优先级：中

建议动作：

1. 清理对 `dbuser.FieldBalance` 的排序依赖
2. 重写测试夹具，让测试直接初始化银行账户，而不是构造 `service.User{ Balance: ... }`
3. 逐步删除仅为 legacy balance 兼容保留的断言

完成标志：

- 新测试默认不再以 `users.balance` 作为前置条件
- 旧字段只剩数据库兼容壳，或者可进入最终 schema 清理阶段

## 六、最终审计结论

### 已删除 / 已失效

- 从 `users.balance` 回读真实余额的主运行时路径：已失效
- 直接修改 `users.balance` 的核心业务写路径：已迁出
- 管理端在普通用户资料更新里改 legacy balance：已冻结

### 仍存在

- `user_balance_projection.go` 兼容写入口仍在
- 仓储层仍会把银行余额投影回 `user.Balance`
- 老用户缺少银行账户时仍会从 `users.balance` 初始化
- 若干排序、结构字段、测试夹具仍保留 legacy shape

### Phase 1.6 重点

- 先干掉运行时回退和初始化兜底
- 再收缩 projection 兼容层调用点
- 最后再处理测试与 schema 级彻底清理
