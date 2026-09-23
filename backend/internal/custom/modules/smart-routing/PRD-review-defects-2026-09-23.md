# 智能路由缺陷修复 PRD

| 项 | 内容 |
| --- | --- |
| 文档版本 | v1.0 |
| 日期 | 2026-09-23 |
| 来源 | 全量代码审查（两轮：安全/资源审查 + 逻辑/控制流审查） |
| 审查范围 | `internal/custom/modules/smart-routing/` 全部源文件、`internal/service/smart_routing_*.go`、`gateway_scheduling.go` 解析器集成、`failover_loop.go` 与三个网关 handler 重试/转发路径、`012/015/018/019` 迁移、`custom/routes.go` 装配、认证中间件状态初始化 |
| 基线分支 | `sync/upstream-v0.2.4-20260910`（8498b879a） |
| 结论 | 未发现可利用的安全漏洞（注入/越权/认证绕过/计费绕过均有完备防线）；发现 3 个 P1、2 个 P2、3 个 P3 缺陷 |

---

## 1. 背景与目标

2026-09-19 故障隔离重构之后，智能路由代码经过两轮独立审查。第一轮聚焦安全与资源（注入、越权、TOCTOU、计费边界、无界资源），第二轮聚焦控制流不变量（同站重试 pin 生命周期、分组耗尽标记链、输出已开始边界、切换预算计数点、handler 间行为差异）。两轮审查的合并结论是：核心路由、熔断、计费、权限逻辑可以信赖，但**遥测写入路径的资源管理**与**请求内失败状态的跨组作用域**存在真实缺陷，另有若干与文档承诺不一致的行为偏差。

本 PRD 定义这些缺陷的修复需求、验收标准与实施顺序。目标读者为实现本轮修复的工程师与验收人。

### 1.1 总体原则

- 所有修复不得改变已验证正确的核心不变量（见第 4 节）。
- 修复必须遵守 Fork 策略：改动隔离在 `custom/` 模块内；确需动上游共享文件（如 `smart_routing_context.go`、`routes.go`）的项目在各自条目中单独标注，实现前需确认边界。
- 每个缺陷修复必须附带回归测试（见各条目的测试要求）。

---

## 2. 问题总览与优先级

| 编号 | 问题 | 严重度 | 优先级 | 类型 | 主要位置 |
| --- | --- | --- | --- | --- | --- |
| SR-1 | `custom_route_decisions` 表无限增长，无任何清理机制 | 中危 | P1 | 资源 | `module.go` / `audit.go` / 迁移 |
| SR-2 | 决策遥测使用无界 goroutine 写库 | 中危 | P1 | 资源 | `module.go` |
| SR-3 | 请求内账号失败集跨分组泄漏，错误排除其他分组的健康账号 | 中低危 | P1 | 逻辑 | `smart_routing_feedback.go` / `module.go` |
| SR-4 | preview 端点缺少速率限制与审计中间件 | 中低危 | P2 | 滥用面 | `custom/routes.go` |
| SR-5 | 容量重试预算使用账号级参照系，且绕过硬尝试预算 | 低危 | P2 | 逻辑 | `smart_routing_retry.go` |
| SR-6 | `RecordSelectedAccount` 依赖竞态重试，可能丢失账号归因 | 低危 | P3 | 遥测质量 | `module.go` |
| SR-7 | 30 秒墙钟预算不包含排队等待，与文档表述有偏差 | 低危 | P3 | 设计偏差 | `smart_routing_context.go` |
| SR-8 | Gemini 入口的智能路由调用点为不可达死代码 | 信息级 | P3 | 清理 | `gateway_handler.go` |

实施顺序建议：SR-1 → SR-2 → SR-3 → SR-4 → SR-5 → SR-6 →（SR-7、SR-8 可选）。

---

## 3. 需求详述

### SR-1 `custom_route_decisions` 表无限增长

**严重度**：中危（生产库长期膨胀）　**优先级**：P1

**现状**
- 每次分组决策写一行：`recordDecision`（`module.go:795`）在 resolver 每次返回决策时插入；`recordNoRoute`（`audit.go:39`）在无路由时插入。智能路由请求在预算内最多重入 resolver 十几次，每次都写。
- 对比另外两张遥测表：`custom_route_attempts` 与 `custom_route_outcomes` 在 `flushFeedback`（`feedback.go:197`）和 `flushOutcomes`（`feedback.go:154`）中都有 24 小时保留、每分钟一次、每批 `LIMIT 10000` 的分批清理。`custom_route_decisions` 全仓库无任何 DELETE，迁移 `012_smart_routing.sql` 也没有保留策略设计。README 第 68 行的保留承诺只覆盖 outcomes。
- 该表索引仅有 `(request_id)` 与 `(api_key_id, created_at DESC)`，没有 `created_at` 单列索引，直接按时间清理会顺序扫描。

**影响**
- 生产库该表持续增长（每次智能路由请求 1~15 行），最终拖慢索引、备份与整库 vacuum。
- 一旦表膨胀到需要紧急清理时，没有现成索引支撑，清理动作本身会形成长事务。

**需求**
- R1.1 `custom_route_decisions` 具备与 `custom_route_attempts`/`custom_route_outcomes` 一致的保留策略：默认 24 小时，分批删除，每批上限 10000 行，清理频率每分钟至多一次（复用现有 cleanup 节拍模式）。
- R1.2 清理失败只记 warn 日志，不影响遥测写入路径，不阻塞请求。
- R1.3 新增迁移 `020_smart_routing_decisions_retention.sql`：为清理谓词补充 `created_at` 索引（`CREATE INDEX IF NOT EXISTS custom_route_decisions_created_idx ON custom_route_decisions (created_at);`），并同步更新示例部署文档中的迁移说明。
- R1.4 保留时长不做配置化（与另外两张表的硬编码口径一致），如未来需要可另行立项。

**验收标准**
- AC1.1 写入 N 行 decision 后，将系统时间推进过保留窗口（或用测试桩控制 `created_at`），清理触发后表内无过期行、非过期行不受影响。
- AC1.2 清理批次不超过 10000 行；剩余过期行在后续节拍内继续清理。
- AC1.3 数据库清理语句失败时，仅产生 `smart_routing_decisions_cleanup_failed` warn 日志，遥测插入照常。
- AC1.4 `custommigrations.Apply` 在全新库与已有库上均幂等通过。

**测试要求**
- 单元测试：仿照 `TestRoutingOutcomeWriterIsBoundedAndCleansExpiredRows`，用 sqlmock 验证清理 SQL 的参数与节拍。
- 迁移测试：仿照 `migrations_test.go` 现有模式，验证 020 迁移幂等。

**风险与边界**
- 不清理历史已有数据之外的内容；上线后第一个清理节拍可能遇到大量过期行，分批 + LIMIT 已限流。
- 不修改 decisions 的写入路径（写入路径治理归 SR-2）。

---

### SR-2 决策遥测使用无界 goroutine 写库

**严重度**：中危（数据库变慢时 goroutine 堆积）　**优先级**：P1

**现状**
- `recordDecision`（`module.go:792`）每次决策直接 `go func()` 写库，无并发上限。
- `RecordSelectedAccount`（`module.go:811`）同样直接 `go func()`，且内部带 5 次 × 20ms 的重试循环（等待异步 INSERT 可见）。
- 对比：`recordNoRoute`（`audit.go:28-34`）使用 `sync.Once` 初始化的 64 槽位信号量（`auditSlots`），满载即丢弃并告警。README 第 68 行"数据库变慢不会产生无界 goroutine"的承诺实际只覆盖 outcomes 队列，未覆盖 decisions。

**影响**
- 数据库短暂变慢（慢于 2 秒超时）时，高并发智能路由流量会瞬时堆积大量 goroutine（每个持有 2 秒写超时；`RecordSelectedAccount` 额外持有最多约 100ms 重试等待）。
- 与 README 的资源承诺不一致。

**需求**
- R2.1 `recordDecision` 与 `RecordSelectedAccount` 的异步写入受有界并发约束：复用或扩展现有的 `auditSlots` 信号量模式（独立于 no-route 的 64 槽位，可单独一个 64 槽信号量），满载时丢弃该条遥测并输出 warn 日志（日志键建议 `smart_routing_decision_audit_full`）。
- R2.2 `RecordSelectedAccount` 的 5×20ms 重试循环保留，但整个重试过程必须在持有槽位的同一个 goroutine 内完成，不得在等待期间额外占用新槽位。
- R2.3 丢弃只影响遥测，不影响任何路由决策与请求结果。
- R2.4 指标：槽位满载丢弃时输出可聚合的 warn（保持 slog 现有风格即可，不引入新指标系统）。

**验收标准**
- AC2.1 用 sqlmock 模拟慢 Exec（阻塞超过 2 秒），并发触发 500 次决策，进程内因这两个函数存活的 goroutine 数有硬上限（≤ 槽位数），其余被丢弃且有 warn。
- AC2.2 数据库正常时，决策与账号归因写入行为与现状完全一致（成功率不下降）。
- AC2.3 `go vet` 与 `golangci-lint --new-from-rev=HEAD` 无新增告警。

**测试要求**
- 单元测试：慢库注入下有界性验证（参考 `TestRoutingOutcomeWriterIsBoundedAndCleansExpiredRows` 的 Eventually 模式）。
- 回归：现有 `module_test.go` 决策写入用例全部保持通过。

**风险与边界**
- 槽位满载意味着遥测有损——这是有意的降级，与 no-route 审计的既有语义一致。
- 不改变 `RecordSelectedAccount` 的 UPDATE 匹配语义（归因准确性问题归 SR-6）。

---

### SR-3 请求内账号失败集跨分组泄漏

**严重度**：中低危（缩小候选组可用池，违背文档承诺语义）　**优先级**：P1（逻辑正确性）

**现状**
- Handler 切组后重建 failover 状态（`gateway_handler.go:665` 的 `fs := NewFailoverState(...)`、`gateway_handler_responses.go:217`、OpenAI 的 `resetSmartRoutingAccountFailover`），清空 handler 本地的账号排除集——注释明确要求"账号 ID 与重试计数属于刚失败的组"。
- 但智能路由侧的 `state.FailedAccountIDs` 跨组存留，写入点有五处：
  1. `BeginSmartRoutingAttempt` 准入拒绝（`smart_routing_feedback.go:71-79`）；
  2. `ObserveSmartRoutingAttempt` 失败观察（`smart_routing_feedback.go:246-251`）；
  3. `FinalizeSmartRoutingPoolFailure` 池终态失败（`smart_routing_feedback.go:339-342`）；
  4. `retrySmartRoutingCapacity` 容量重试（`smart_routing_retry.go:34-40`）；
  5. `inspectModelAccounts` 标记冷却账号（`module.go:884-891`）。
- 合并点 `SmartRoutingExcludedAccounts`（`smart_routing_feedback.go:361-374`）把这些 ID **不分组地**并入每次选号排除集；`module.go:894-897` 的候选检查同样直接读取。
- README 第 15 行明确承诺"不会让同一分组内的普通账号共享冷却、粘性或失败计数"，跨组共享违背该意图。

**影响**
- 场景：同一账号（网站入口）同时挂在候选组 A 和组 B（合法配置）。请求在 A 中失败后切到 B，B 的选号将该账号永久排除，即使它在 B 组完全健康；容量检查（`inspectModelAccounts`）也会把 B 组里的它当作已失败。
- 有界性：12 次尝试预算内自然终止，不会死循环；纯粹缩小了后续组的可用池，极端情况下导致本可成功的请求返回 503。
- 注意与"账号健康熔断"（跨组共享，`custom:route:account:v2` 命名空间）的区分：熔断按设计就是网站级跨组共享（README 第 46 行），**不是**本问题；本问题仅指请求内（request-local）的失败标记跨组生效。

**需求**
- R3.1 `state.FailedAccountIDs` 的排除作用域限定为"记录失败时所在的分组"。
- R3.2 推荐实现：在 `SmartRoutingRequestState` 增加并行映射 `FailedAccountGroups map[int64]int64`（accountID → 失败时 `CurrentGroupID`），所有写入点同步登记；`SmartRoutingExcludedAccounts` 与 `inspectModelAccounts` 读取时只合并"登记分组 == 当前分组"的条目。
- R3.3 明确不采用"切组时清空 `FailedAccountIDs`"方案：主组单次重入路径（`SmartRoutingPrimaryRetryUsed`）需要保留主组内的失败记录，清空会导致已死账号在主组被重试。
- R3.4 该字段属于上游共享文件 `smart_routing_context.go` 的请求状态结构——按 Fork 策略，动此文件前需确认边界（改动是新增字段与过滤逻辑，不修改既有字段语义）。
- R3.5 兼容性：旧字段保留（供日志/审计展示失败总数），仅排除逻辑改为分组限定。

**验收标准**
- AC3.1 账号 X 同时属于组 A、组 B；请求在 A 中使 X 转发失败并切组；B 组选号可以选中 X（stub 仓储单测）。
- AC3.2 请求停留在同一分组时，行为与现状完全一致（组内失败账号仍被排除）。
- AC3.3 主组重入路径（primary retry）仍排除主组内已失败账号。
- AC3.4 `smart_routing_group_retry_test.go`、`smart_routing_reliability_test.go` 现有全部用例不回归。

**测试要求**
- 新增回归测试：双组共享账号的排除作用域（覆盖上述 AC3.1-AC3.3 三个路径）。
- 既有 `TestResolveNeverReadmitsExhaustedGroup` 等不变量测试保持通过。

**风险与边界**
- 若真实部署中"同账号挂多组"配置不存在，则该修复无行为变化（防御性正确）。
- 熔断（Redis/本地 circuit）行为不变。

---

### SR-4 preview 端点缺少速率限制与审计中间件

**严重度**：中低危（已认证用户可制造数据库压力）　**优先级**：P2

**现状**
- `custom/routes.go:101-107`：SmartRouting 路由组只挂了 `jwtAuth` + `BackendModeUserGuard`，无 `panelRateLimiter`、无 `auditLog`。
- `preview`（`module.go:469-514`）每次调用触发大量数据库查询：profile 读取、主组 + 每个候选组的 `loadStats`（含 percentile_cont 统计）、账号池枚举（`inspectModelAccounts`）、用户费率（`GetUserGroupRates`）、权限分组列表（`GetAvailableGroups`，内部含用户、活跃分组、订阅三次查询）。单查询有 250ms 超时但总查询数无上限。
- 对比：同文件 `ActivityCheckin`（`routes.go:108-118`）挂了 `panelRateLimiter.Global()`；`RechargePromotions` 等用户面板模块均带 `auditLog`。

**影响**
- 已认证用户高频刷 preview 可对数据库形成放大压力（每次 10+ 查询 × 候选组数）。
- 用户面板操作无审计记录，与其余 custom 模块不一致。

**需求**
- R4.1 SmartRouting 路由组（`/api/v1/keys/:id/routing` 的 GET/PUT/preview）补挂 `panelRateLimiter.Global()` 与 `auditLog` 中间件，对齐 `ActivityCheckin` 的装配方式（`routes.go` 属 custom 文件，可直接改）。
- R4.2 限流命中时返回标准 429 响应（由 `panelRateLimiter` 现有行为提供，不新做格式）。
- R4.3 preview 内部查询的数据库压力不得新增额外防护（限流已足够），不改变 preview 的返回结构。

**验收标准**
- AC4.1 超过面板限流阈值的连续请求收到 429；阈值内请求行为与现状一致。
- AC4.2 GET/PUT/preview 三条路由的操作进入审计日志。
- AC4.3 `pnpm test:run` 前端相关用例（若有路由装配断言）不回归。

**测试要求**
- 后端：路由装配测试（仿照现有 custom 模块的装配测试模式）。
- 手工验证：本地起服务后刷 preview 观察限流与审计。

**风险与边界**
- `panelRateLimiter` 为可选注入（`RegisterRoutes` 参数），需确认生产装配已提供；未提供时保持现状（无中间件），不报错。

---

### SR-5 容量重试预算使用账号级参照系，且绕过硬尝试预算

**严重度**：低危（有界但语义偏差 + 选号压力放大）　**优先级**：P2

**现状**
- `retrySmartRoutingCapacity`（`smart_routing_retry.go:31`）的预算判断是 `state.Attempt+state.CapacityRetries >= maxSwitches+1`，调用方传入 `h.maxAccountSwitches`（默认 10，即最多 11 次容量重试）。
- 分组切换预算 `state.MaxGroupSwitches` 为 2（`api_key_auth.go:346`）。容量重试本质是分组级重入（重走 `resolveSmartRoutingGroup` → 完整 `rank()`，每组多次 DB 查询），却按账号级预算计数。
- 更关键的缺口：`TotalAttempts` 硬预算（12 次）只在 `forwardWithSmartRouting` 内的 `SmartRoutingAttemptBudgetAvailable` 递增（`smart_routing_retry.go:79`），容量重试发生在转发之前，**完全不消耗也不检查硬预算**。容量重试的唯一上界就是上述 11 次。
- `state.Attempt` 是遥测计数（`ObserveSmartRoutingAttempt` 递增），与 `CapacityRetries` 相加作预算是双轨混用。

**影响**
- 高并发槽位紧张时，单个请求可触发 11 轮完整 rank()（每轮 10+ DB 查询 × 候选组数），放大选号数据库压力。
- 与文档"分组最多切换 2 次"的直觉严重不符。

**需求（含决策点）**
- R5.1 **决策点**（需在实现前确认）：
  - 方案 A（推荐）：容量重试预算改用分组级 `state.MaxGroupSwitches`（2），即 `Attempt+CapacityRetries >= MaxGroupSwitches+1` 时停止，与文档语义对齐；
  - 方案 B：保持 11 次上限，但在需求文档与 README 中明确"容量重试按账号级预算计数"。
  - 推荐理由：容量不可用通常短时间内不可恢复，3 次重入已足够覆盖瞬时抖动；多余重试只消耗数据库。
- R5.2 若采纳方案 A：`maxSwitches` 参数从调用方移除或改为从 `state.MaxGroupSwitches` 读取；`maxSwitches < 0` 的防御分支保留。
- R5.3 行为变更需在 PR 说明中显式标注：高并发下部分请求会更快返回 503（而不是重试 11 次）。
- R5.4 无论方案 A/B：为容量重试路径补充日志字段 `capacity_retries`，便于上线后观察新预算是否过紧。

**验收标准**
- AC5.1（方案 A）容量重试最多 3 次（`MaxGroupSwitches=2` + 1）后按无可用账号路径返回。
- AC5.2 普通转发失败的重试路径不受影响（`TotalAttempts` 硬预算 12 次语义不变）。
- AC5.3 现有 `smart_routing_retry_test.go` 用例按新预算更新后全部通过。

**测试要求**
- 单元测试更新：`retrySmartRoutingCapacity` 的边界（第 3 次拒绝）。
- 回归：`gateway_handler_stream_failover_test.go` 等容量路径用例。

**风险与边界**
- 行为收紧可能影响依赖长重试的边缘场景；上线后观察 503 率与 `capacity_retries` 分布，必要时回调阈值。

---

### SR-6 `RecordSelectedAccount` 依赖竞态重试，可能丢失账号归因

**严重度**：低危（纯遥测质量）　**优先级**：P3

**现状**
- `module.go:803-826`：`recordDecision` 异步 INSERT 与 `RecordSelectedAccount` 异步 UPDATE 解耦，后者靠 5 次 × 20ms 重试弥补"INSERT 尚未提交"的窗口；窗口错过则该决策行的 `selected_account_id` 保持 NULL。
- UPDATE 匹配条件：`request_id+api_key_id+selected_group_id+selected_account_id IS NULL` 的最新一行。

**影响**
- 约 100ms 内 INSERT 未可见时账号归因丢失；只影响 `custom_route_decisions` 遥测的账号维度，不影响路由与计费。

**需求**
- R6.1（低成本方案，推荐）在 SR-2 的有界槽位实现中，让 INSERT 完成后（同一 worker 内）直接尝试 UPDATE，将重试窗口从"盲等 20ms"改为"INSERT 后立即可见"；5 次重试保留为兜底。
- R6.2 不引入跨请求状态或同步等待；归因丢失率以日志可观测（`RecordSelectedAccount` 放弃时 debug 日志）。

**验收标准**
- AC6.1 数据库正常时归因成功率不低于现状；模拟慢 INSERT（延迟 < 2s）时重试窗口内完成归因。
- AC6.2 请求延迟不受影响（全部异步）。

**测试要求**
- 单元测试：INSERT 延迟注入下归因最终一致。

---

### SR-7 30 秒墙钟预算不包含排队等待（设计偏差文档化）

**严重度**：低危（有意设计，但与表述有差距）　**优先级**：P3

**现状**
- `SmartRoutingStartedAt` 在第一次 `SmartRoutingAttemptBudgetAvailable` 调用时才初始化（`smart_routing_context.go:221-223`），即首次转发尝试。
- 账号槽位排队（WaitPlan）最长 sticky 45s / fallback 30s，不计入 30 秒墙钟预算。README 第 37 行表述为"墙钟预算默认为 30 秒"，未区分排队阶段。

**需求（含决策点）**
- R7.1 **决策点**：
  - 方案 A（推荐）：保持现状，更新 README 第 37 行表述为"30 秒为上游尝试阶段的墙钟预算，不含账号槽位排队等待"；
  - 方案 B：把起算点提前到智能路由激活（`Resolve` 确认激活时初始化 `SmartRoutingStartedAt`）——会使排队 45s 的请求在首次转发前即耗尽预算直接 503，**不推荐**；
  - 方案 C：为排队阶段增加独立的时长上限（如 60s），与转发预算并行。
- R7.2 若采纳方案 A：仅改文档，无代码变更；在 outcome 记录中已有 `duration_ms` 与 `attempts` 可间接观察排队占比，无需新增指标。

**验收标准**
- AC7.1（方案 A）README 表述更新，与代码行为一致。

---

### SR-8 Gemini 入口的智能路由调用点为不可达死代码

**严重度**：信息级　**优先级**：P3

**现状**
- `smartRoutingEndpoint`（`api_key_auth.go:324-335`）只识别 `/chat/completions`、`/responses`、`/messages` 三个后缀，Gemini 的 `/v1beta/...:generateContent` 落到 `unsupported`。
- resolver 端点门禁（`module.go:572`）与主组平台白名单（`module.go:728-733`，仅 OpenAI/Anthropic）双重拒绝，Gemini 请求的 `SmartRoutingActive` 永远为 false。
- 但 `gateway_handler.go` 的 Gemini 循环（334、722 行等）仍保留 `ShouldRetrySmartRoutingGroup` / `TrySmartRoutingGroupFailover` 调用点。

**需求**
- R8.1 保留调用点作为防御（`SmartRoutingActive=false` 时这些函数是安全 no-op）——**建议不删代码**，仅在 Gemini 分支的调用点补一行注释说明"防御性调用，Gemini 端点当前不激活智能路由"，避免后人误以为 Gemini 参与智能路由。
- R8.2 若未来要支持 Gemini 智能路由，需先过端点识别、平台白名单、messages 兼容性三道门禁并单独立项。

**验收标准**
- AC8.1 注释补充后 `go vet` / lint 通过；无行为变更。

---

## 4. 已验证不修改项（范围外声明）

以下不变量在两轮审查中逐一验证正确，明确不在本 PRD 修改范围，回归测试须保持其通过：

1. **同站重试 pin 生命周期闭合**：`SetSmartRoutingRetryAccount` → `LimitSmartRoutingAccountsForRetry`（两个调度器内裁剪）→ `CompleteSmartRoutingRetrySelection` / `ClearSmartRoutingRetryAccount`；pin 账号消失时自动清空回退普通池；pin 生效时清空 sessionHash 防粘性抢跑（`gateway_scheduling.go:36-40`）。
2. **池模式健康晋级链**：中间失败 `HealthFinal=false` 只进遥测；`FinalizeSmartRoutingPoolFailure` 仅在重试预算耗尽后以 `HealthOnly=true` 补健康写入，带五重防御；已产生输出路径正确调用 Finalize。
3. **探索交换顺序**：探索只影响 `ranked[0]` 排序，耗尽/计费拒绝/切换预算/已试过滤四道门禁全部生效在其后；门禁条件保证仅首决策触发。审查中曾疑虑，已核实**不是缺陷**。
4. **`GroupSwitches` 单点递增**：仅 `resolveSmartRoutingGroup` 实际采纳新候选时递增；主组恢复不计切换；`SmartRoutingPrimaryRetryUsed` 保证主组最多重入一次。
5. **输出已开始的切组边界**：三条 handler 路径（`gatewayStreamSemanticOutputStarted` / `openAISmartRoutingSelectionOutputStarted`（排除 compact 心跳字节）/ `writerSizeBeforeForward` 对比）均在切组前检查。
6. **`MainGroupID` 请求内不可变**；rank 从 state 重读主组，主组消失时 fail-closed。
7. **`restorePrimary` 失败语义**：主组已耗尽或连续性锁不属于主组时置 `NoRoute`，配合 `SmartRoutingContractValid` 立即失败。
8. **TOCTOU 双重防护**：rank 打分后、admission 前重读 primary/candidate 行核对 platform/subscription/status。
9. **计费边界**：候选组独立走订阅/余额/分组 RPM 检查，不重复计 key/user 级计数；长上下文加价、渠道定价不可比时候选排除。
10. **权限链**：`ownedKey` 归属校验 + 三处独立分组权限复查 + Redis 响应路由键含 UserID/KeyID/PrimaryID。
11. **Lua 熔断脚本**：探测租约 token fencing、指数退避封顶、迟到成功不解除冷却。
12. **流式失败路径 fail-closed**：profile 损坏、权限服务不可用、账号仓储失败时回退已认证主组。

---

## 5. 实施顺序与工作量估计

| 顺序 | 编号 | 预估工作量 | 依赖 |
| --- | --- | --- | --- |
| 1 | SR-1 | 0.5 天 | 无（含 020 迁移与测试） |
| 2 | SR-2 | 0.5 天 | 无（可与 SR-1 同一 PR） |
| 3 | SR-3 | 1~2 天 | 需确认 `smart_routing_context.go` 改动边界（上游共享文件） |
| 4 | SR-4 | 0.5 天 | 无 |
| 5 | SR-5 | 0.5 天（方案 A） | 实现前确认决策点 R5.1 |
| 6 | SR-6 | 0.5 天 | 建议搭 SR-2 的槽位实现顺带完成 |
| 7 | SR-7 | 0.25 天（方案 A 纯文档） | 无 |
| 8 | SR-8 | 0.25 天 | 无 |

建议分组提交：
- PR-1（SR-1 + SR-2 + SR-6）：遥测资源治理，全部位于 smart-routing 模块内，零上游文件改动。
- PR-2（SR-3）：请求状态作用域修复，含上游共享文件 `smart_routing_context.go` 的新增字段，动前确认。
- PR-3（SR-4 + SR-5）：面板防护与预算对齐。
- PR-4（SR-7 + SR-8）：文档与注释收尾。

## 6. 测试与验收计划

每轮修复完成后，在合并前必须通过与 CI 等价的检查（对齐 README 2026-09-22 验证基线）：

```
cd backend
go test ./...
go test -tags=unit ./...
go vet ./...
golangci-lint run --new-from-rev=HEAD
```

- 针对性回归：`go test ./internal/custom/modules/smart-routing/ ./internal/service/ -run "SmartRouting|Routing" -count=1`、`go test ./internal/handler/ -run "SmartRouting|Failover" -count=1`。
- 每个缺陷的新增回归测试按各条目"测试要求"执行。
- `go test -race` 在本机 Windows 无 GCC 环境不可用（README 第 86 行已知限制），并发改动（SR-2、SR-3）如有条件应在 Linux CI 补跑。

## 7. 发布与回滚注意

- 上线顺序按第 5 节 PR 顺序；PR-1（遥测治理）风险最低可先行。
- SR-5（方案 A）是唯一收紧请求行为的项目：上线后观察 `capacity_retries` 分布与 routing 503 率，异常可单独回滚该提交。
- SR-3 改变请求内排除语义：上线后观察切组请求的成功率（`custom_route_outcomes.final_success` 按切组数分组）。
- 020 迁移只加索引，幂等可回滚（DROP INDEX）。
- 全部改动不触碰账号管理状态与生产 Redis 既有命名空间，无需数据迁移或缓存清理。

## 8. 2026-09-23 实施复核状态

本轮代码已按上述审查结论完成收口，当前状态如下：

- SR-1：新增 `020_smart_routing_decisions_retention.sql`，决策表增加 `created_at` 索引；Overlay 独立 ticker 每分钟分批清理 24 小时前数据。
- SR-2/SR-6：决策、无路由和选中网站归因统一进入按 `request_id` 分片的有界 FIFO 队列；满载只丢遥测，不阻断请求；选中网站归因保留有限重试。
- SR-3：失败网站记录改为账号 ID + 分组集合，调度器按当前分组合并排除集，主组重入时仍保留主组失败记录。
- SR-4：Key 路由 GET/PUT/preview 接入面板限流与审计，GET 加入敏感读取审计白名单。
- SR-5：容量重试改用 `MaxGroupSwitches + 1` 的请求级预算，不再混用账号切换上限和遥测 `Attempt`。
- 契约安全：Gateway/OpenAI 主分组契约查询任意错误均 fail-closed；候选最终准入再次检查 active、平台、订阅类型、用户权限/订阅和 Composite；未知订阅类型拒绝；账号模型映射拒绝 Gemini/Claude/Bedrock Anthropic 外部族进入 GPT/OpenAI 路径。
- SR-7/SR-8：README 已明确 30 秒仅覆盖上游尝试阶段；Gemini 保留防御性调用并标注当前不会激活智能路由。

仍需在发布前独立完成：生产制品哈希核对、独立端口桌面/移动端验收、真实配置下 OpenAI/Anthropic 请求与账单抽样、Redis/数据库故障演练，以及完整 `go test ./...`、前端 lint/typecheck/Vitest。此次代码验证不等同于已发布或生产已验收。
