# Projection 调用点审计报告

## 审计范围

- 审计时间：2026-06-04
- 审计对象：
  - `UpdateUserBalanceProjection(...)`
  - `UpdateUserBalanceProjectionIfNeeded(...)`
- 扩展说明：
  - 当前仓库实际还存在两个包装函数：
    - `UpdateUserBalanceProjectionFromTransferResult(...)`
    - `UpdateUserBalanceProjectionFromBankAccount(...)`
  - 它们本质上也是 projection 兼容层的一部分，因此一并统计

## 结论摘要

当前仓库内：

1. `UpdateUserBalanceProjectionIfNeeded(...)` 已经不存在，调用点数量为 `0`。
2. 直接调用 `UpdateUserBalanceProjection(...)` 的运行时代码共有 `4` 处，全部位于“创建用户 / 注册赠送”的兼容返回路径。
3. 若把两个包装函数一并算入 projection 兼容层，则当前运行时调用点总数为 `11` 处：
   - `UpdateUserBalanceProjection(...)`：4 处
   - `UpdateUserBalanceProjectionFromTransferResult(...)`：5 处
   - `UpdateUserBalanceProjectionFromBankAccount(...)`：2 处
4. 从当前代码语义看，没有“现在立刻可无脑删除”的调用点；但其中大部分属于 Phase 1.6 可逐步收缩的兼容调用。

说明：

- 上面第 3 点按“调用语句数”统计。
- `user_balance_projection.go` 内部包装函数互相调用未计入“外部调用点”。

## 一、精确搜索结果

### 1. `UpdateUserBalanceProjectionIfNeeded(...)`

搜索结论：

- 当前代码库未搜索到该函数定义
- 当前代码库未搜索到任何调用点

审计判断：

- 该路线已经完成清理，不构成 Phase 1.6 的主要工作项

### 2. `UpdateUserBalanceProjection(...)` 外部调用点

共 `4` 处：

1. `backend/internal/service/admin_signup_balance_grant.go:19`
2. `backend/internal/service/admin_signup_balance_grant.go:32`
3. `backend/internal/service/auth_signup_balance_grant.go:25`
4. `backend/internal/service/auth_signup_balance_grant.go:38`

### 3. `UpdateUserBalanceProjectionFromTransferResult(...)` 外部调用点

共 `5` 处：

1. `backend/internal/service/admin_signup_balance_grant.go:36`
2. `backend/internal/service/admin_service.go:985`
3. `backend/internal/service/auth_signup_balance_grant.go:42`
4. `backend/internal/service/auth_service.go:719`
5. `backend/internal/service/auth_oauth_email_flow.go:296`

### 4. `UpdateUserBalanceProjectionFromBankAccount(...)` 外部调用点

共 `2` 处：

1. `backend/internal/repository/api_key_repo.go:689`
2. `backend/internal/service/billing_cache_service.go:862`

## 二、逐点分类

分类原则：

- 必须保留：当前删掉会直接破坏现有兼容返回、鉴权热路径对象形态，或会让现有返回对象失去余额镜像
- 可删除：当前语义上已经只是兼容残留，可在 Phase 1.6 具备前置条件后删除

## 三、必须保留

### 1. `backend/internal/repository/api_key_repo.go:689`

调用：

- `UpdateUserBalanceProjectionFromBankAccount(out, userBankAccountEntityToView(...))`

原因：

- 这是 API Key 鉴权读取用户对象时的统一投影点
- 当前 `userEntityToService(...)` 先把 `u.Balance` 填入结构，再在存在银行账户时用银行余额覆盖
- `backend/internal/repository/user_bank_balance_mapper_test.go` 已明确要求：
  - 有银行账户时，`user.Balance` 应等于银行账户余额
  - 无银行账户时，才保留 legacy balance

结论：

- 在调用方仍有 `user.Balance` 兼容需求之前，这个调用点需要保留。

### 2. `backend/internal/service/billing_cache_service.go:862`

调用：

- `UpdateUserBalanceProjectionFromBankAccount(user, account)`

原因：

- `checkBalanceEligibility(...)` 在 `user.BankAccount == nil` 时，会主动加载银行账户快照
- 加载完成后立即把余额镜像补回 `user.Balance`
- 这保证后续同一请求内持有的 `user` 对象与权威账户快照保持一致

结论：

- 在运行时仍复用 `user.Balance` 展示镜像的前提下，这个调用点应保留。

### 3. `backend/internal/service/admin_service.go:985`

调用：

- `UpdateUserBalanceProjectionFromTransferResult(user, result)`

原因：

- 管理员手工调余额后，真实余额来自账本事务结果
- 当前函数需要把更新后的余额镜像回返回对象 `user`
- 如果现在直接删掉，返回给上层的 `user.Balance` 会不同步

结论：

- 这是“账本已更新，返回对象仍需兼容”的典型调用点，当前应保留。

## 四、可删除

以下“可删除”不是说现在立刻删，而是说它们属于 Phase 1.6 可计划性移除的兼容路径，只要先完成调用方改造，就可以收口。

### 1. `backend/internal/service/admin_signup_balance_grant.go:19`

调用：

- `UpdateUserBalanceProjection(user, amount)`

原因：

- 分支条件是 `s.entClient == nil`
- 这是非账本事务分支下，为新建用户对象补一个兼容余额镜像
- 一旦创建用户返回对象不再依赖 `user.Balance`，该调用可删

分类：

- 可删除

### 2. `backend/internal/service/admin_signup_balance_grant.go:32`

调用：

- `UpdateUserBalanceProjection(user, 0)`

原因：

- 这是事务创建用户后、账本实际入账前的临时置零镜像
- 纯粹是为了让内存对象在后续 `TransferResult` 回来之前保持一个过渡值

分类：

- 可删除

### 3. `backend/internal/service/admin_signup_balance_grant.go:36`

调用：

- `UpdateUserBalanceProjectionFromTransferResult(user, result)`

原因：

- 真实账本结果已经存在
- 该调用只服务于“创建用户接口返回对象仍带 `user.Balance`”

分类：

- 可删除

前提：

- 创建用户返回 DTO 改为直接返回银行账户余额，或前端不再消费 `user.Balance`

### 4. `backend/internal/service/auth_signup_balance_grant.go:25`

调用：

- `UpdateUserBalanceProjection(user, balance)`

原因：

- 邮箱注册 / 默认赠送的非事务分支兼容镜像

分类：

- 可删除

### 5. `backend/internal/service/auth_signup_balance_grant.go:38`

调用：

- `UpdateUserBalanceProjection(user, 0)`

原因：

- 注册事务内的临时过渡镜像

分类：

- 可删除

### 6. `backend/internal/service/auth_signup_balance_grant.go:42`

调用：

- `UpdateUserBalanceProjectionFromTransferResult(user, result)`

原因：

- 账本入账后把结果回填到 `user.Balance`
- 本质是注册返回对象兼容层

分类：

- 可删除

### 7. `backend/internal/service/auth_service.go:719`

调用：

- `UpdateUserBalanceProjectionFromTransferResult(newUser, result)`

原因：

- 发生在注册流程事务中
- 作用仍是把账本结果镜像回返回给上层的 `newUser`

分类：

- 可删除

### 8. `backend/internal/service/auth_oauth_email_flow.go:296`

调用：

- `UpdateUserBalanceProjectionFromTransferResult(user, result)`

原因：

- OAuth / 首次绑定赠送完成后，为当前返回对象补 legacy 展示余额

分类：

- 可删除

## 五、分类汇总

### 必须保留

共 `3` 处：

1. `backend/internal/repository/api_key_repo.go:689`
2. `backend/internal/service/billing_cache_service.go:862`
3. `backend/internal/service/admin_service.go:985`

这些调用点的共同特点是：

- 仍直接服务当前运行时对象一致性
- 删除后容易立刻影响鉴权热路径、余额调整返回值或兼容展示结果

### 可删除

共 `8` 处：

1. `backend/internal/service/admin_signup_balance_grant.go:19`
2. `backend/internal/service/admin_signup_balance_grant.go:32`
3. `backend/internal/service/admin_signup_balance_grant.go:36`
4. `backend/internal/service/auth_signup_balance_grant.go:25`
5. `backend/internal/service/auth_signup_balance_grant.go:38`
6. `backend/internal/service/auth_signup_balance_grant.go:42`
7. `backend/internal/service/auth_service.go:719`
8. `backend/internal/service/auth_oauth_email_flow.go:296`

说明：

- 从“可计划收口”的语义上，这一组共有 `8` 处。
- 若严格按“现在立刻删除不会破坏兼容行为”的标准，则它们暂时还不能直接删。
- 因此本报告里的“可删除”含义是：
  - 已不属于基础账本真相源
  - 在 Phase 1.6 改完返回契约后可移除

## 六、Phase 1.6 建议删除顺序

建议按风险从低到高推进。

### 第一步：先删创建/注册流程内的过渡镜像

目标文件：

- `backend/internal/service/admin_signup_balance_grant.go`
- `backend/internal/service/auth_signup_balance_grant.go`
- `backend/internal/service/auth_service.go`
- `backend/internal/service/auth_oauth_email_flow.go`

原因：

- 这些调用点集中在“创建用户后返回对象”的兼容层
- 不属于鉴权热路径
- 改造成本相对最低

### 第二步：改造管理员调余额返回契约

目标文件：

- `backend/internal/service/admin_service.go`

原因：

- 这是剩余服务层最稳定的单点镜像调用
- 处理完后，服务层 projection 残留将进一步收缩

### 第三步：最后处理仓储层与鉴权热路径

目标文件：

- `backend/internal/repository/api_key_repo.go`
- `backend/internal/service/billing_cache_service.go`

原因：

- 这两处最接近系统运行时核心路径
- 需要先确认：
  - 调用方是否还依赖 `user.Balance`
  - DTO / 缓存对象 / 鉴权上下文是否已经全面改成使用 `BankAccount.Balance`

## 七、最终结论

### `UpdateUserBalanceProjectionIfNeeded`

- 定义：无
- 调用点：0
- 结论：已清理完成

### `UpdateUserBalanceProjection` 系列整体情况

- 直接基础函数外部调用：4 处
- 包装函数外部调用：7 处
- 当前真正需要短期保留的核心兼容点：3 处
- 适合在 Phase 1.6 继续收口的调用点：8 处

### 审计判断

当前 projection 兼容层已经被压缩到少数明确位置，说明 Phase 1.5 的“集中收口”方向是对的；但它还没有彻底退出运行时对象层。Phase 1.6 的重点不再是继续找散落调用，而是先改掉返回契约和调用方对 `user.Balance` 的依赖，再成批删除这些兼容投影调用。
