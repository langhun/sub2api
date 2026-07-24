# 下游定制功能 Overlay 模块化 PRD

## 1. 文档信息

| 字段 | 内容 |
| --- | --- |
| 文档状态 | 待评审 |
| 创建日期 | 2026-07-24 |
| 适用范围 | 仅限 `sub2api` 下游定制功能；上游源代码保持不变 |
| 当前基线 | `origin/main` v0.1.164；`feat/port-balance-features` 相对该基线领先 78 个提交、落后 0 个提交 |
| 负责人 | 项目维护者 |
| 相关文档 | `docs/prd-activity-and-entertainment-center.md` |

## 2. 背景与问题

本项目在持续跟随上游 `sub2api` 的同时，已增加活动中心、余额转账、红包、排行榜、娱乐大厅、首页品牌体验等定制能力。当前定制代码横跨后端 Handler、Service、Repository、Ent schema、SQL 迁移、路由、设置、前端 API、Store、页面、导航和国际化。

这种方式短期交付快，但随着功能和上游更新积累，会带来以下问题：

1. 单一长期功能分支同时承担产品集成、开发和上游同步职责，提交边界不清晰。
2. 上游同步会在路由、Wire、全局设置、导航、i18n、Ent 生成物和迁移等共享文件产生高频冲突。
3. 新功能容易直接修改全局文件，后续难以判断哪些改动属于某一业务能力，也难以独立测试、关闭或回滚。
4. 数据库与资金相关功能的改动若缺少边界，容易在迁移、事务和权限校验中形成隐性耦合。
5. 用 Git submodule、Go plugin 或拆微服务来回避冲突，会额外引入多仓库版本协调、部署、鉴权和数据一致性成本，当前阶段得不偿失。

本 PRD 的目标是在保留现有单体部署模型的前提下，将已加入的定制功能收拢为独立 Overlay。上游业务代码不做模块化、不做重构；迁移完成后，上游更新应可直接合并，冲突只允许发生在受控的少数挂载点和可再生输出中。

## 3. 产品目标

### 3.1 核心目标

1. 让每项下游能力有可识别的所有权、入口、依赖、开关、测试和迁移记录，并且代码落在自定义目录。
2. 将上游同步的人工冲突压缩到少量固定挂载点和可再生输出，而不是散落到上游业务实现中。
3. 使新模块可以在不删除代码的情况下关闭入口、拒绝直达请求并停止业务执行。
4. 使新模块的开发、测试、代码审查和回滚可以按模块进行。
5. 维持单个 Go 二进制、单个前端构建产物和当前生产发布流程，不增加运行时网络调用。

### 3.2 可量化目标

1. 自 Phase 1 起，所有新增下游功能必须归属一个 Overlay 模块，且满足本 PRD 的模块完整性清单。
2. 上游同步提交不得夹带新产品功能；每次同步应有独立的 `sync/upstream-<version>` 分支和 `chore(sync)` 提交。
3. 三次连续上游同步后，除挂载点白名单外，不应再有下游业务逻辑与上游业务文件产生冲突。
4. 模块关闭时，用户入口、直达路由、公开 API、管理 API 和业务 Service 均不能继续提供该能力。
5. 每个模块至少拥有一组后端与前端回归测试，并在上游同步后执行。

## 4. 非目标

本期不做以下事项：

1. 不拆分微服务，不引入服务间 RPC、消息队列或独立数据库。
2. 不使用 Go runtime plugin；所有模块仍在编译期链接进主二进制。
3. 不将活动中心或现有下游代码拆到 Git submodule、Git subtree 或独立仓库。
4. 不对上游 Handler、Service、Repository、设置、路由或页面进行架构重构。
5. 不改变上游原有 API 的语义；定制接口通过 Overlay 路由注册，不把业务逻辑塞回上游 API 文件。
6. 不重写 Ent、迁移执行器、权限系统或全局设置系统。
7. 不为“目录整洁”批量移动上游文件；只移动本分支已加入的代码，并使被触及的上游文件尽可能恢复为上游版本。

## 5. 术语与角色

| 术语 | 定义 |
| --- | --- |
| 上游基线 | 原项目的主分支，作为同步来源。推荐远端名为 `upstream`。 |
| 产品集成分支 | 承载所有已验证下游定制并用于发布的长期分支，推荐名为 `product/main`。 |
| 功能分支 | 从产品集成分支创建的短生命周期分支，命名为 `feature/<scope>`。 |
| 同步分支 | 专门合并某个上游版本的临时分支，命名为 `sync/upstream-<version>`。 |
| Overlay 模块 | 仅包含本站新增能力、可独立识别、开关、测试并通过受限接口与主应用协作的一组完整业务能力。 |
| 挂载点 | 为接入 Overlay 而允许保留极小补丁的上游组合入口；不承载任何下游业务规则。 |
| 可再生输出 | 由 Ent、Wire 或前端构建生成的文件。合并时不手工解决其内容，而是由保留的源文件重新生成。 |

| 角色 | 职责 |
| --- | --- |
| 模块负责人 | 定义边界，维护业务实现、测试、配置、迁移和升级说明。 |
| 集成维护者 | 维护模块注册表、解决上游同步冲突、执行生成与回归门禁。 |
| 管理员 | 在后台配置模块开关和业务规则，核验审计记录。 |
| 用户 | 在被授权且功能开启时访问模块能力。 |

## 6. 总体方案

### 6.1 单体内模块化

模块化采用“编译期 Overlay”而不是运行时插件：

```text
单一 Go 服务 + 单一 PostgreSQL 数据库 + 单一 Vue 构建产物
        |
        +-- upstream core（不修改业务实现）
        +-- custom/activity
        +-- custom/wallet-extension
        +-- custom/game-hall
        +-- custom/brand-home（仅前端）
```

Overlay 模块可使用主应用提供的数据库、认证、审计、配置、HTTP 路由和前端基础组件，但不得绕过既有的权限、事务、错误处理和审计机制。模块本身必须位于 `custom` 命名空间；上游目录仅保留挂载调用，不能包含模块业务代码。

### 6.2 模块完整性

一个 Overlay 模块至少包含下列内容；缺少任一项时不得宣称已模块化：

| 层级 | 必需内容 |
| --- | --- |
| 身份 | 唯一模块 ID、名称、负责人、状态、依赖模块列表。 |
| 后端 | 路由注册、Handler、Service、Repository，以及明确的错误和权限策略。 |
| 数据 | Ent schema 归属、仅追加的 SQL 迁移、迁移回归测试。 |
| 配置 | 模块总开关和业务子开关；公开设置、后台设置和运行时消费点一致。 |
| 前端 | 路由、页面、组件、API、Store、导航、i18n、直达路由守卫。 |
| 质量 | 单元测试、关键集成测试、开关关闭测试、上游同步检查项。 |
| 运维 | 指标、审计动作、发布/回滚影响说明。 |

### 6.3 允许的依赖方向

```text
模块前端 -> 模块 API -> 模块 Handler -> 模块 Service -> 模块 Repository/Ent
                                  |                    |
                                  +-> core 公共能力 <-+

模块 A -- 受限接口 --> core 或模块 B 的公开契约
模块 A -- 禁止 ------> 模块 B 的私有 Repository、私有表实现或前端内部状态
```

允许模块依赖稳定的 core 公共能力，例如用户身份、余额读取、审计、配置和事务上下文。跨模块调用必须通过小而稳定的接口完成，例如 `WalletPort`、`RewardDeliveryPort`；不得直接导入对方私有 Service 或 Repository。

## 7. 模块目录与注册设计

### 7.1 后端目标目录

本项目当前按 `Handler -> Service -> Repository` 分层。Overlay 目录只承载本站新增的业务实现和装配；迁移时只移动本分支引入的代码，并使原有上游文件回到或接近上游版本。

```text
backend/internal/custom/
  registry.go
  runtime.go
  modules/
    activity/
    module.go
    routes.go
    handler/
    service/
    repository/
    contract/
    testdata/
    wallet-extension/
    module.go
    routes.go
    handler/
    service/
    repository/
    contract/
    game-hall/
    module.go
    routes.go
    handler/
    service/
    repository/
    contract/
```

要求：

1. `module.go` 声明模块 ID、依赖、功能定义和装配入口。
2. `routes.go` 分别注册用户、公开和管理端路由；现存于上游 `routes/*.go` 的本站路由必须迁出到此处。
3. `contract/` 只放跨模块可依赖的 DTO、接口和错误类型，不暴露私有数据访问实现。
4. 迁移只处理本分支加入的 Handler、Service、Repository 和测试；既有上游 `backend/internal/handler`、`service`、`repository` 不改名、不搬迁、不重构。
5. 新增下游业务文件默认放在 `backend/internal/custom/modules/`；不得再进入上游全局层。

### 7.2 前端目标目录

```text
frontend/src/custom/
  registry.ts
  modules/
    activity/
    index.ts
    routes.ts
    navigation.ts
    api/
    stores/
    views/
    components/
    i18n/
    __tests__/
    wallet-extension/
    index.ts
    routes.ts
    navigation.ts
    api/
    stores/
    views/
    components/
    i18n/
    __tests__/
    game-hall/
    index.ts
    routes.ts
    navigation.ts
    api/
    stores/
    views/
    components/
    i18n/
    __tests__/
    brand-home/
    routes.ts
    views/
    components/
    assets/
    __tests__/
```

要求：

1. 模块 `routes.ts` 导出自身路由和路由元数据，根路由只在固定挂载点聚合 `custom/registry.ts` 的导出。
2. 模块 `navigation.ts` 导出导航项，由 `custom/registry.ts` 汇总后交给固定导航挂载点。
3. 模块 API、Store、View、组件和 i18n 均在 `frontend/src/custom/` 内闭环；不移动或重写上游页面与组件。
4. 路由级动态加载由模块自行定义，不能因模块增加而无条件增大首屏包。
5. 前端不以“隐藏入口”代替权限或开关校验；直达路由仍必须守卫并依赖后端最终校验。

### 7.3 固定挂载点与自定义注册表

根应用不做通用重构。仅建立 `custom` 注册表，并在下表白名单中的位置保留一次性挂载调用。白名单以外的上游文件不允许再承载下游业务逻辑：

| 集成点 | 目标方式 |
| --- | --- |
| 后端依赖注入 | `backend/cmd/server/wire.go` 只负责构造 `custom.Runtime`；自定义 Provider 保留在 `backend/internal/custom/`。`wire_gen.go` 作为可再生输出。 |
| 后端路由 | `backend/internal/server/router.go` 只调用一次 `custom.RegisterRoutes(...)`；所有现存的签到、转账、红包、排行榜和娱乐大厅路由从 `routes/*.go` 迁出。 |
| Ent schema | 仍位于 `backend/ent/schema/` 以兼容生成工具，但仅新增 `custom_<module>_*.go` 文件；不编辑上游 schema。`backend/ent/**` 为可再生输出。 |
| SQL 迁移 | 仍位于 `backend/migrations/`；只新增含 `custom_<module>` 前缀的迁移文件，不修改历史迁移或上游迁移。 |
| 设置 | 优先由 Overlay 自己的设置表和 `/api/v1/custom/...` 管理接口承载，避免扩散修改上游 `SettingService` 和公开设置 DTO。 |
| 前端路由 | `frontend/src/router/index.ts` 只追加一次 `...customRoutes`；各模块路由均由 `frontend/src/custom/registry.ts` 汇总。 |
| 前端导航 | `AppSidebar.vue` 只接收一次 `customNavigation`；已有定制菜单迁出，禁止在上游导航数组继续添加业务项。 |
| 前端页头扩展 | 仅当确有需求时，`AppHeader.vue` 增加一个 `CustomHeaderActions` 挂载点；签到等下游 UI 必须迁入该组件。 |
| 国际化 | `frontend/src/i18n` 只追加一次 `custom` 语言片段入口；模块语言文件保留在 `frontend/src/custom/`。 |

每个白名单文件最多保留一次导入和一次注册调用。新增下游需求不得扩大白名单；若确实无法避免，必须在 PR 中说明原因、替代方案和预期合并影响。

### 7.4 上游文件归还规则

迁移一个现有定制能力时，必须同时完成以下工作：

1. 将本分支在上游文件中新增的业务逻辑移动到 `backend/internal/custom/` 或 `frontend/src/custom/`。
2. 删除该能力对 `backend/internal/server/routes/common.go`、`user.go`、`admin.go`、上游 Handler、Service、Repository、前端页面和布局的直接修改。
3. 除挂载点白名单与可再生输出外，对比 `origin/main...HEAD` 时，上游文件不应再出现该能力的下游 diff。
4. 若出于兼容必须暂存旧路径，兼容层也必须位于 `custom` 目录，写明移除版本和回归测试。
5. 每次迁移后运行一次“非白名单 diff”检查；发现新的上游文件改动即阻止合并。

### 7.5 白名单与 diff 检查

白名单分为三类：

| 类别 | 允许路径 | 规则 |
| --- | --- | --- |
| Overlay 自有代码 | `backend/internal/custom/**`、`frontend/src/custom/**` | 可自由新增和修改，必须有模块归属与测试。 |
| 新增数据定义 | `backend/ent/schema/custom_*.go`、`backend/migrations/*_custom_*.sql` | 只新增文件；迁移不可回写或改号。 |
| 固定挂载/生成输出 | `backend/cmd/server/wire.go`、`backend/internal/server/router.go`、`frontend/src/router/index.ts`、`frontend/src/components/layout/AppSidebar.vue`、必要时 `AppHeader.vue` 与 i18n 入口，以及 Ent/Wire 生成文件 | 仅允许表 7.3 定义的挂载调用或生成结果，禁止混入业务规则。 |

除以上三类外，任何相对 `origin/main` 的改动都默认视为不合规。若确有必要，必须先更新白名单说明并在 PR 中说明：为什么 Overlay 无法完成、为什么不能使用现有挂载点、合并上游时的预期冲突位置和回滚方法。

`go.mod`、`go.sum`、`package.json` 与锁文件在本次迁移中不应变更；Overlay 必须优先复用现有依赖。若后续模块确需新增依赖，必须单独评审，不得与业务迁移混在同一提交。

## 8. 初始模块边界

### 8.1 activity

范围：普通签到、运气签到、盲盒、奖励投递、红包、排行榜和相关管理配置。

不包含：通用用户余额字段、通用身份认证、支付渠道、Dg 游戏余额。

依赖：用户身份、余额/事务接口、审计、设置、公开设置和通知能力。

关键约束：签到、红包、奖励必须使用事务、幂等与并发校验；模块总开关关闭后，页面、直达路由、公开接口、管理接口和 Service 均拒绝执行。

### 8.2 wallet-extension

范围：站内余额转账、转账收款方搜索、转账限额/手续费设置、转账流水和管理端查询。

不包含：上游原有充值、支付、用户核心余额模型的所有权。

依赖：core 提供的用户、余额、事务、审计和权限接口。

关键约束：不得绕过 core 余额一致性和事务策略；收款人搜索、限额判断、扣款、入账、审计和展示应由模块自身闭环。

### 8.3 game-hall

范围：DG 钱包、双向兑换、奖池、游戏回合、派彩、用户级禁用、后台运营与记录查询。

不包含：activity 的签到、红包和通用奖励规则。

依赖：用户、余额/兑换接口、随机源、审计、设置和通知能力。

关键约束：任何派彩必须可审计、可幂等、可追溯；生产随机源不得退化为可预测伪随机实现。

### 8.4 brand-home

范围：根首页品牌体验、DaiGua 页面、Dino 页面和静态资源。

不包含：用户资金、后台设置、数据库写入。

依赖：前端路由、公共布局和静态资源嵌入。

关键约束：作为前端模块独立演进；发布时必须通过嵌入资源存在性、正确 Content-Type 和浏览器实际渲染验证。

## 9. 功能需求

### 9.1 模块身份与状态

1. 每个模块必须定义稳定且不可复用的 `moduleID`，例如 `activity`、`wallet_extension`、`game_hall`。
2. 每个模块必须声明状态：`enabled`、`disabled`、`experimental` 或 `deprecated`。
3. 每个模块必须声明模块版本和最小兼容上游版本，用于同步评审记录。
4. 每个模块必须声明直接依赖模块；启动时应检查不存在循环依赖或缺失依赖。
5. 后台诊断信息由 Overlay 管理路由提供；不得为此重构上游设置或管理端仪表盘。

### 9.2 功能开关

1. 模块总开关为所有子功能的前置条件。
2. 子功能开关只能缩小模块开放范围，不能在模块总开关关闭时绕过限制。
3. 开关判断必须覆盖：导航、前端路由守卫、公开 API、管理 API、Handler、Service 和后台任务。
4. 对外状态通过模块自己的公开端点提供；不得为了新增模块状态而修改上游公开设置 DTO。
5. 设置修改必须写入审计日志，记录模块 ID、设置键、修改前后值和操作人，不记录秘密内容；设置表与管理接口归属 Overlay。

### 9.3 路由与导航

1. 每个模块应通过 Overlay 注册入口提供用户、公开和管理端路由；路径统一位于 `/api/v1/custom/<module>` 或经 `custom` 兼容层映射的旧路径下。
2. 路由 meta 必须包含模块 ID、权限需求和功能开关标识。
3. 导航项必须从模块元数据派生，避免 AppHeader、AppSidebar 与路由分别维护同一菜单。
4. 停用模块后，用户看到的入口应隐藏或禁用，直接访问页面显示明确的不可用状态；后端返回稳定的业务错误码。
5. 上游已有路由与下游模块路由的路径冲突必须在同步分支处理，不得通过覆盖上游 Handler 静默解决；新增接口优先使用 `custom` 命名空间。

### 9.4 后端业务与跨模块契约

1. Overlay Handler 只负责鉴权、参数校验、调用 Overlay Service 和输出 DTO。
2. Overlay Service 负责业务规则、功能开关、权限、事务边界、幂等和审计。
3. Overlay Repository 只访问自身表和明确允许的 core 读取模型；跨模块写入必须经由放在 `custom/contract/` 的适配接口。
4. 跨模块契约应优先使用最小接口与 DTO，避免暴露 Ent client、事务实现或内部表结构。
5. 发生跨模块失败时，调用方必须返回可追踪错误并保留审计上下文；不得吞掉资金、奖励或派彩异常。
6. 禁止模块间相互导入私有 Service/Repository，避免循环依赖和升级时的连锁冲突。

### 9.5 数据、Ent 与迁移

1. Overlay 专属 Ent schema 文件仍放入 `backend/ent/schema/`，命名使用 `custom_<module>_` 前缀，例如 `custom_activity_checkin.go`、`custom_game_hall_round.go`；只新增文件，不编辑上游 schema。
2. 每项数据库变更必须新增迁移，禁止修改任何已在环境执行过的 SQL 文件。
3. 迁移名必须包含模块前缀，并与现有迁移序号规则兼容；引入上游迁移后必须复核排序、checksum 和实际表名。
4. 迁移应尽量保持向前兼容：先加表/列和双读写，再切换调用方，最后在后续版本清理废弃逻辑。
5. Ent 或 Wire 发生冲突时，保留上游生成物、合并自定义源 schema/Provider 后重新生成；禁止手工修补生成文件作为最终结果。
6. 资金、奖励和游戏派彩相关迁移必须有专项集成或回归测试，覆盖旧库升级与重复执行保护。

### 9.6 前端体验

1. Overlay 页面不得在根路由、上游全局布局或非本模块 Store 中承载业务状态。
2. 模块页面的加载、空态、权限不足、功能关闭、网络失败和数据一致性错误必须有可用界面。
3. Overlay API 类型、错误映射和前端文案应随模块维护，禁止散落在上游全局工具文件。
4. 对资金与奖励类操作，前端只提供交互保护；最终校验以后端事务和幂等为准。
5. 新模块页面需要同时验证桌面与移动端；不得因为模块拆分改变既有首页默认路由行为。

### 9.7 测试与质量门禁

每个模块至少应具备：

| 测试类别 | 最低要求 |
| --- | --- |
| 后端单元测试 | Service 规则、权限、开关关闭、错误映射。 |
| 后端集成测试 | 迁移、Repository、事务、幂等或并发路径。 |
| API 契约测试 | 用户、公开和管理端接口的成功与失败路径。 |
| 前端单元测试 | API、Store、路由守卫和关键组件状态。 |
| 前端视图测试 | 正常、功能关闭、权限不足、错误和空态。 |
| 上游同步回归 | 模块最小测试集、生成检查、前后端构建与嵌入资源检查。 |

必须执行的基础门禁：

```powershell
Set-Location backend
go generate ./ent
go generate ./cmd/server
go test ./...

Set-Location ..\frontend
pnpm typecheck
pnpm test:run
pnpm build
```

涉及嵌入式发布时，额外执行 `go test -tags embed ./internal/web`，并验证入口 HTML 引用的 JS/CSS 实际存在且响应类型正确。

## 10. 上游同步与分支流程

### 10.1 分支模型

```text
upstream/main
      |
      +--> sync/upstream-vX ------> product/main ------> feature/<scope>
                                       |                     |
                                       +------ release -------+
```

1. `upstream/main` 只反映上游，不包含本站定制提交。
2. `product/main` 是下游产品集成和发布基线，不 rebase、不强推。
3. `feature/<scope>` 从 `product/main` 创建，完成单一模块目标后合回。
4. `sync/upstream-<version>` 专门处理上游同步、冲突、生成和回归；不得混入产品功能。
5. 现有 `feat/port-balance-features` 在切换完成前可继续作为兼容分支名，但不再承担新增功能的默认落点。

### 10.2 同步步骤

1. 确认工作区干净，获取上游引用。
2. 从当前 `product/main` 建立 `sync/upstream-<version>`。
3. 合并上游版本并记录冲突文件及其归属模块。
4. 对 schema、Wire 和其他生成物先合并源文件，再重新生成。
5. 运行基础门禁和受影响模块的专项测试。
6. 检查路由、设置、公开接口、导航、迁移和静态资源嵌入是否保留模块行为。
7. 以单独的 `chore(sync): 合并上游 vX 更新` 提交完成同步，再合入 `product/main`。
8. 在变更记录中保存版本、冲突摘要、验证命令、残余风险和回滚点；不记录秘密信息。

### 10.3 冲突处理优先级

| 冲突区域 | 处理原则 |
| --- | --- |
| Ent schema / Wire | 先保留双方源定义，再执行生成；不手工保留生成结果。 |
| SQL 迁移 | 不修改已发布迁移；确认编号、执行顺序、checksum 和真实数据兼容性。 |
| 路由 / 导航 | 优先迁入注册表，保留上游路由并增加模块路由。 |
| 设置 / 公开设置 | 保留上游新增设置并合并模块设置键、默认值、校验与运行时消费点。 |
| 用户/余额/权限 | 上游安全和一致性修复优先；模块通过公开契约适配。 |
| 前端页面与组件 | 先保留上游的 bugfix 和可访问性改进，再恢复模块特有交互。 |

### 10.4 Git 远端、分支与本地保护

#### 远端职责

| 远端 | 唯一职责 | 允许操作 |
| --- | --- | --- |
| `upstream` | 原项目 `Wei-Shaw/sub2api` | `fetch`；同步分支从 `upstream/main` 合并变更。 |
| `origin` | 本站个人 fork | 推送 `product/main`、`feature/*`、`sync/*` 和标签。 |

当前仓库仅有名为 `origin` 的远端，且指向原项目。个人 fork URL 未配置前，禁止重命名该远端、禁止强推、禁止假定 `origin` 可以承载本站发布分支。待取得个人 fork URL 后，按“新增个人 `origin`、将原项目命名为 `upstream`”的顺序切换，并在切换后验证两者 URL。

#### 分支职责

| 分支 | 职责 | 允许写入 |
| --- | --- | --- |
| `main` | 上游镜像基线，仅用于对比 | 不直接提交、不作为开发基线。当前本地 `main` 已过期，未经明确确认不得重置。 |
| `product/main` | 已验证的本站产品集成和发布基线 | 仅合并验证过的 `feature/*` 与 `sync/*`；禁止直接开发、rebase 或强推。 |
| `feature/<scope>` | 单一 Overlay 功能或迁移任务 | 正常提交；完成后以 merge commit 合入 `product/main`。 |
| `sync/upstream-<version>` | 单个上游版本的同步、冲突和回归 | 只允许上游合并、冲突处理、生成输出和测试修复。 |
| `release/<version>` 或标签 | 已发布的不可变回滚点 | 只创建，不重写。 |

创建 `product/main` 时只增加一个指向当前已验证提交的分支，不删除或改写现有 `feat/port-balance-features`。待发布和远端策略稳定后，再决定是否保留该旧分支名。

#### 本地 Git 保护

本仓库必须启用以下本地配置：

```powershell
git config --local rerere.enabled true
git config --local merge.conflictStyle zdiff3
git config --local fetch.prune true
git config --local pull.ff only
git config --local diff.renames true
```

含义：记录已人工确认的冲突解决方式、显示共同祖先上下文、清理失效远端引用、阻止 `git pull` 产生隐式 merge commit，并提高移动文件的识别能力。`rerere` 只记录解决方案，不自动提交；每次复用结果仍须审查和测试。

#### 强制操作规则

1. 所有操作先执行 `git status --short`；工作区不干净时不开始同步、rebase、切换发布分支或构建发布。
2. 禁止在 `product/main`、`sync/*` 和已发布提交上执行 `git rebase`、`git push --force`、`git reset --hard` 或删除分支。
3. 禁止使用不带目标的 `git pull`；上游更新必须显式 `fetch` 后在 `sync/upstream-<version>` 执行 `git merge --no-ff upstream/main`。
4. 每次提交前执行 `git diff --check`，只暂存本任务文件，并使用中文 Conventional Commit。
5. 每次同步或模块迁移后，使用 `git diff --name-only <upstream-base>...HEAD` 检查是否出现白名单外的上游文件；任何异常文件必须在合并前解释或归还。
6. 发布前创建不可变标签；生产回滚仅使用已验证标签或保留二进制，不通过改写 Git 历史恢复。

## 11. 分阶段交付

### Phase 0：基线保护与设计确认

目标：建立长期产品分支和本 PRD 约束，不改变运行行为。

交付：

1. 确认 `upstream` 与个人 `origin` 的远端职责。
2. 创建 `product/main` 指向当前已验证的产品提交。
3. 为上游同步建立独立分支和提交规范。
4. 记录当前模块清单、已知共享冲突区和专项测试入口。

验收：现有工作区、生产行为和发布流程不改变。

### Phase 1：Overlay 挂载骨架

目标：建立 `custom` 注册表和固定挂载点白名单，不改动上游业务实现。

交付：

1. 后端 `custom.Runtime`、Overlay 元数据和路由注册接口。
2. 前端 `custom/registry.ts`、路由、导航和 i18n 聚合接口。
3. 挂载点白名单及“非白名单 diff”检查脚本。
4. `activity`、`wallet-extension`、`game-hall`、`brand-home` 的空壳 Overlay 注册。

验收：除白名单中的一次性挂载调用外，没有上游源文件业务逻辑变动；全量基础测试和构建通过。

### Phase 2：迁出现有下游功能

目标：将已经加入的功能迁入 Overlay，并归还被污染的上游文件。

交付：

1. 将签到、红包、排行榜、转账、娱乐大厅、Dino/品牌首页按模块迁出。
2. 将上游 `routes/common.go`、`user.go`、`admin.go` 中的本站路由和辅助函数删除，改由 Overlay 注册。
3. 将本站 API、Store、View、组件、i18n 和测试迁入 `frontend/src/custom/`。
4. 将自定义 schema 和迁移统一为 `custom_<module>` 命名；生成物只通过生成命令更新。

验收：每迁出一个能力，除挂载点和可再生输出外，该能力不再使上游文件与 `origin/main` 存在 diff；关闭模块后无入口和业务执行遗漏。

### Phase 3：后续功能 Overlay 优先

目标：禁止新功能再次进入上游目录。

要求：

1. 新功能按本 PRD 的完整性清单实现，所有业务文件进入 `custom` 目录。
2. 新增路由、导航、设置、API、Store、i18n 和测试从 Overlay 导出。
3. 新增 schema 和迁移使用 `custom_<module>` 前缀。
4. 对 core 余额、身份、审计等能力只通过适配接口调用，不把功能实现写回上游包。

验收：代码审查可明确判断 Overlay 归属，且非白名单 diff 检查通过。

### Phase 4：同步效率评估

目标：完成至少三次上游同步后，以数据评估模块化效果。

交付：冲突文件统计、模块测试耗时、功能回归问题数、模块关闭演练结果和需要调整的模块边界。

验收：确认继续增量演进、合并模块或调整公共契约的具体决定。

## 12. 验收标准

### 12.1 模块边界

1. 新增功能能在代码、路由、设置、迁移和测试中追溯到唯一模块 ID。
2. 模块之间不存在直接依赖对方私有 Repository、私有 Service 或前端内部 Store 的情况。
3. 上游文件只允许出现在挂载点白名单或可再生输出中，不存在下游业务逻辑散改。
4. 不因 Overlay 化改变上游 core 的既有默认路由、鉴权或支付行为。

### 12.2 开关与权限

1. 模块总开关关闭时，导航、直达页面、公开 API、管理 API、后台任务和 Service 均被阻止。
2. 子功能开关行为与模块总开关一致且可由自动化测试验证。
3. 管理端修改模块设置后具有审计记录，普通用户不能访问管理接口。

### 12.3 数据与一致性

1. 所有新迁移均为追加式，能从空库和已有生产结构的模拟副本完成升级。
2. Ent 与 Wire 在生成后无未提交差异。
3. 资金、奖励、派彩等模块通过事务、幂等和并发回归测试。
4. 上游同步不篡改已执行迁移的历史内容或 checksum。

### 12.4 上游同步

1. 每次同步在独立分支完成，并能提供冲突与验证记录。
2. 同步提交不包含新功能需求或无关重构。
3. 生成、后端测试、前端类型检查、前端测试和嵌入资源门禁通过。
4. 同步后所有启用模块的关键用户路径可用，关闭模块的阻断路径仍生效。
5. “非白名单 diff”检查通过；若产生新上游路径，必须有评审批准的白名单变更。

### 12.5 发布与回滚

1. 仍按现有生成、前端构建、嵌入、Linux 二进制构建、SHA-256 校验、原子替换、健康检查和回滚保留流程发布。
2. 发布后验证服务状态、`/health`、模块路由以及至少一个实际 JS/CSS 静态资源响应类型。
3. 任一模块导致异常时，能使用保留二进制回滚整个单体；高风险模块可先通过功能开关止血。

## 13. 风险与应对

| 风险 | 影响 | 应对 |
| --- | --- | --- |
| 一次性大迁移 | 引入大量无关冲突和回归 | 一次只迁移一个下游能力，并在每次迁移后归还对应上游文件。 |
| Overlay 名义化 | 文件分目录但仍散改上游 | 用挂载点白名单和非白名单 diff 检查阻止合并。 |
| 共享余额模型耦合 | 资金功能升级后出现账务错误 | 使用 core 契约、事务和集成测试；核心安全修复优先。 |
| Ent 生成冲突 | 生成物误合并或丢失 schema | 只合并源 schema/Provider，统一重新生成。 |
| 迁移编号冲突 | 部署失败或历史不一致 | 同步时优先处理迁移，检查执行顺序与 checksum。 |
| 开关覆盖不完整 | 功能关闭后仍可直达或写数据 | 前端、路由、Handler、Service、任务全部有测试。 |
| 上游 API 改动 | 模块兼容性失效 | 以模块契约测试尽早发现，避免直接调用上游内部实现。 |
| 过度抽象 | 开发速度下降 | 只抽取已出现两个以上消费者的稳定契约。 |

## 14. 待确认事项

1. 是否将当前 `origin` 明确为个人 fork，并新增 `upstream` 指向原项目；若当前 `origin` 已是上游，则先保留名称，待远端策略统一后再调整。
2. `product/main` 的最终名称是否改为 `daigua/main` 或其他更易识别的产品分支名。
3. 是否在管理端增加模块只读诊断页；本期可先只通过配置和日志查看。
4. 公开设置接口是否需要显式返回模块版本和模块状态，供前端诊断与灰度使用。
5. 余额契约是先抽取最小读写接口，还是保持现有 Service 并仅补充契约测试。
6. 是否启用 `git rerere` 记录重复冲突解决方案；该配置可降低重复人工处理，但不能替代模块边界治理。

## 15. 首个实施任务

首个任务必须是无行为变化的基础设施提交：

```text
refactor(custom): 建立下游 Overlay 挂载骨架
```

范围仅包括：`custom` 注册表、后端/前端固定挂载调用、空壳 Overlay、白名单 diff 检查和基础测试。不得在此提交中移动活动、转账、娱乐大厅的现有业务实现，不得修改数据库结构，不得改变用户可见功能；上游文件只允许出现表 7.5 列出的极小挂载补丁。

完成该任务后，先按一个能力一个提交的方式迁出现有下游功能，再要求后续新功能只进入 `custom` Overlay。
