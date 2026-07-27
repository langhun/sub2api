# 下游 Overlay 迁移现状清单

> **2026-07-25 收口结论。** 本页较早的分模块“待迁移”描述只保留作历史债务背景；以下结论优先：brand-home、game-hall、activity 与 wallet-extension 的业务实现、页面、API、路由、设置片段和相邻测试均已迁入 `backend/internal/custom/**` 或 `frontend/src/custom/**`。核心层只保留经 `custom.Runtime`、`custom/registry.ts` 和 `handler/settingsext.Mount` 接入的通用挂载点。
>
> 不能移动的例外是已发布的 SQL 迁移、Ent schema/生成物，以及为保证单事务余额语义保留的最小数据访问适配器；它们不是模块业务实现。`default_homepage` 现由 brand-home 模块设置片段持有，默认仍回退到 `/home`。当前增量门禁以最后一次上游同步后的审核检查点 `c27d1237628cac3e41a8586c3fb5c9dcbd2bc19c` 为准；历史身份/管理路径库存仍可通过 `HistoricalAudit` 单独盘点，但不应误报为本轮模块迁移新增问题。

> 当前审计日期：2026-07-25。上游基线 `origin/main` 为 `333acde7d189b5a604854bc4fc5bb768c7ad58c9`，当前 Overlay 检查点为 `feature/activity-overlay` 的 `4d73b9d8de8e88939ee9824b9d540802574f0240`，共同祖先为 `cb24522dd53f8f363d008e3afdc8e4baf9788cab`。
>
> `origin/main...HEAD` 当前有 449 个路径：custom backend 69、custom frontend 86、固定集成 21、Ent/迁移历史或生成物 83、文档 3、白名单外共享路径 187。分支相对当前上游领先 93 个提交、落后 16 个提交。
>
> 本清单只描述当前已跟踪的下游差异，不包含未跟踪文件或工作区临时改动。旧的迁移计划保留为债务清理参考；本页的当前状态和门禁规则优先。

## 1. 使用方式与边界

当前差异共有 **449 个路径**。大量 backend/ent/** 是 schema 生成输出，不能按文件逐个迁移或手工解冲突；本清单按业务所有权和迁移动作聚合，列出的路径是每组关键入口和路径模式。

一次模块迁移的目标不是重构上游，而是：

1. 将本分支新增的业务实现移至 backend/internal/custom/modules/<module>/ 和 frontend/src/custom/modules/<module>/。
2. 同一提交删除该能力对上游 Handler、Service、Repository、routes/*.go、页面和布局的直接改动。
3. 只留下 custom 注册表、固定挂载点、追加式历史数据定义和重新生成的 Ent/Wire 输出。

brand-home、game-hall、activity 和 wallet-extension 的路由、前端入口和运行时挂载已完成 Overlay 收口。活动和钱包仍依赖共享 Handler、Service、Repository、设置、Ent schema 与通用前端文件，不能宣称全量业务实现已经完全模块化；后续只清理这些兼容债务，不得借迁移扩展娱乐大厅功能或改写已发布的数据结构。

| 标记 | 含义 | 迁移处理 |
| --- | --- | --- |
| M | 模块自有业务 | 移入模块 custom 目录，并迁移相邻测试。 |
| R | 需归还的上游路径 | 模块迁完后恢复上游行为；仅固定挂载调用留在白名单。 |
| H | 已发布历史数据定义 | 不重命名、不拆分、不改写；新变更才使用 custom_<module> 命名。 |
| G | 可再生输出 | 不手工迁移；修改 schema/Provider 后统一生成。 |
| S | 共享适配层 | 收口为 custom 契约或注册表，不能继续扩散业务规则。 |

## 2. 模块总览

| 模块 | 当前状态 | 当前能力 | 顺序 | 主要风险 |
| --- | --- | --- | --- | --- |
| brand-home | 主体已完成；设置片段待收口 | 根首页选择、/Dino、Chromium Dino 资源与运行器 | 已完成 | 保留 /home 默认行为和 /Dino 大小写路径。 |
| game-hall | 已完成 | DG 独立钱包、兑换、Slots、回合/流水、用户访问控制和模块管理入口 | 已完成 | 余额事务、幂等、历史迁移和用户级开关。 |
| activity | 路由/入口已迁入；实现和设置仍有共享债务 | 签到、幸运签到、盲盒、奖励投递、红包、排行榜和管理配置 | 1 | 后台任务、幂等、并发和跨模块余额写入。 |
| wallet-extension | 路由/入口已迁入；实现和设置仍有共享债务 | 站内余额转账、收款方搜索、限额/手续费、管理和转账榜 | 2 | 直接改动核心余额、事务、审计和用户查询；最后迁移。 |

activity 与 wallet-extension 共享 173 号历史迁移和部分设置/余额契约，但不能合为一个模块；两者只能经 custom/contract/ 的最小接口通信。

## 3. brand-home

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | frontend/src/custom/modules/brand-home/{DaiGuaView.vue,ChromiumDinoRunner.vue,RootHomeView.vue,chromiumDino/**,routes.ts} 与相邻测试 | 已迁入的 DaiGua 页面、运行器、根首页选择和 Chromium 离线 Dino 实现。 |
| M | frontend/public/custom/brand-home/** | 已迁入的精灵图、音效与 Chromium 许可证；保持 Vite 静态资源 URL。 |
| R | frontend/src/router/index.ts | 已删除具体 Dino 路由定义，仅展开一次 customRoutes。 |
| S | frontend/src/stores/app.ts；后端设置 DTO/公开设置链路 | default_homepage 仍在通用设置对象中，是待收口的品牌入口设置兼容债务。 |

当前完成情况：

1. 已由 frontend/src/custom/modules/brand-home/routes.ts 导出 /Dino 和根路径处理路由。
2. 页面、Runner、chromiumDino/**、资源、许可证和测试均已迁入模块目录。
3. 默认行为仍为 /home，只有公开设置明确为 dino 时才跳转 /Dino。
4. 仅剩 default_homepage 的通用设置链路，后续作为设置片段收口；不修改上游 /home 行为。

~~~powershell
Set-Location frontend
pnpm typecheck
pnpm test:run -- src/custom/modules/brand-home
pnpm build
~~~

手工验证：未加载公开设置时 / 仍落到 /home；设置为 dino 时 / 到 /Dino；刷新 /Dino 后资源可加载并可开始游戏。

## 4. game-hall

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | backend/internal/custom/modules/game-hall/{handler.go,module.go,repository.go,routes.go,service.go} 与相邻 *_test.go | 已迁入的用户端、管理端、交易/回合业务和数据访问。 |
| M | frontend/src/custom/modules/game-hall/{api.ts,api/admin.ts,store.ts,GameHallView.vue,admin/GameHallSettingsPanel.vue,routes.ts,navigation.ts,locales.ts} 与相邻测试 | 已迁入的游戏大厅 API、状态、页面、用户访问控制、路由、导航和翻译。 |
| R | backend/internal/server/routes/{user.go,admin.go} | 游戏大厅路由已由模块 RegisterRoutes 注册；这里不再直接注册 /game-hall 和 /admin/game-hall。 |
| R | backend/internal/{handler/wire.go,service/wire.go,repository/wire.go,handler/handler.go}；backend/cmd/server/wire.go | 主体装配已转入 custom.Runtime；全局 Wire/Handler 仅保留兼容或生成输出，不得恢复游戏业务。 |
| R | backend/internal/{handler/admin/user_handler.go,handler/dto/{mappers.go,types.go},service/{admin_user.go,user.go}}；backend/ent/schema/user.go；frontend/src/components/admin/user/UserEditModal.vue | `game_hall_disabled` 已从通用用户模型、DTO 和编辑页移除；上游用户能力不再承载游戏大厅业务。 |
| H | backend/migrations/175_add_game_hall_dedicated_tables.sql；176_backfill_game_hall_dedicated_balances.sql；178_add_game_hall_rounds.sql；180_add_user_game_hall_disabled.sql；187_move_game_hall_user_access.sql | 已有 game_hall 钱包、流水、奖池、回合与用户禁用数据；187 已回填访问决策到 `game_hall_user_access` 并移除旧用户列，所有迁移原样保留。 |
| H | backend/migrations/game_hall_migrations_regression_test.go；backend/internal/repository/game_hall_repo_integration_test.go | 保留为历史升级与事务回归门禁。 |
| S | backend/internal/service/setting_balance_features.go、公开设置、前端 feature flags/i18n | game_hall_enabled 与兑换/Slots 限制应收口为模块设置适配器。 |

当前游戏数据是原生 SQL 表，不是 Ent schema：game_hall_wallets、game_hall_wallet_transactions、game_hall_jackpots、game_hall_jackpot_transactions、game_hall_rounds、game_hall_main_balance_transactions。迁移服务代码时必须维持用户钱包、奖池与主余额在同一事务中的锁定与幂等语义。

当前完成情况与剩余债务：

1. Repository、Service、Handler、用户/管理路由、前端 API、Store、页面、导航、翻译和测试均已迁入模块目录。
2. 模块路由经 custom.Runtime 注册；不得再把游戏路由或 Handler 字段加回上游 routes/Wire。
3. 后续只处理最小 contract：主余额、用户身份、审计、设置和幂等能力不得继续直接扩散上游依赖。
4. 用户禁用已由模块专属 `game_hall_user_access` 承接；管理员仅经 `/api/v1/admin/game-hall/users/:user_id/access` 和游戏大厅设置面板操作，不能将开关加回通用用户 API 或 `UserEditModal`。

~~~powershell
Set-Location backend
go generate ./ent
go generate ./cmd/server
go test -tags=integration ./internal/custom/modules/game-hall/...
Set-Location ../frontend
pnpm typecheck
pnpm test:run -- src/custom/modules/game-hall
~~~

额外验证：兑换和游戏请求重放不重复扣款；用户禁用后用户路由、管理/公开设置和前端入口均受阻；迁移回归测试仍覆盖既有数据库。

## 5. activity

状态：**路由、前端入口和 Runtime 挂载已迁入；实现提取未完成**。`backend/internal/custom/modules/activity/**` 与 `frontend/src/custom/modules/activity/**` 已拥有模块入口、路由、导航、页面和测试；签到、盲盒、奖励投递、红包和排行榜仍通过共享 Service、Repository、Handler、Ent schema 与设置链路实现。

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | backend/internal/custom/modules/activity/**；frontend/src/custom/modules/activity/** | 当前模块入口、路由、导航、页面、组件、API、翻译和相邻测试。 |
| S | backend/internal/{handler/checkin_handler.go,handler/leaderboard_handler.go,handler/admin/blindbox_handler.go} | 签到、榜单、盲盒和管理 API 的剩余共享实现。 |
| S | backend/internal/service/{checkin_service.go,blindbox_service.go,leaderboard_service.go,reward_delivery*.go,red_packet_expiry_service.go} 与测试 | 签到、盲盒、榜单、奖励投递 Outbox 和红包到期后台任务的剩余共享实现。 |
| S | backend/internal/repository/{balance_redpacket_repo.go,reward_delivery_repo.go} 与测试 | 红包与可靠奖励投递持久化的剩余共享实现。 |
| S | frontend/src/{api,stores,views,components,utils}/** | 历史活动 UI 与辅助逻辑的剩余共享路径；后续仅可删除或迁入 custom。 |
| R | backend/internal/server/routes/{common.go,user.go}；frontend/src/components/layout/{AppHeader.vue,AppSidebar.vue}；frontend/src/router/index.ts | 已移除活动业务注册，只保留固定挂载。 |
| H | backend/migrations/173_port_balance_features.sql | 同时包含签到、盲盒、转账和红包的首批结构与设置；不可拆分或改名。 |
| H | backend/migrations/177_migrate_legacy_checkin_records.sql；179_repair_legacy_checkin_migration.sql；181_create_reward_deliveries.sql；185_widen_large_balance_amount_columns.sql | 已有签到历史回填、修复、奖励 Outbox 和精度升级；必须原样保留。 |
| G | backend/ent/schema/{checkin.go,checkin_blindbox_record.go,checkin_prize_item.go} 和相关 backend/ent/{checkin*,checkinblindboxrecord*,checkinprizeitem*} | schema 改为 custom_activity_*.go 后生成；62 个 Ent 非 schema 差异路径均视为输出，不手工迁移。 |
| S | backend/ent/schema/{user.go,redeem_code.go}；backend/internal/service/{redeem_code.go,redeem_service.go,code_format*.go} | 当前活动扩展了上游 User/RedeemCode 与兑换码格式。迁移时优先模块 DTO 和 Core 适配接口；无法解除时列为兼容债务。 |

剩余迁移顺序：

1. 先拆 activity/contract：用户余额事务、订阅/兑换码、审计、通知和设置读取接口。
2. 迁入签到、盲盒、奖励投递和后台任务；把启动/停止责任从 cmd/server/wire.go 移至 custom.Runtime。
3. 迁入红包与排行榜，移除 routes/common.go、routes/user.go、routes/admin.go 的业务注册。
4. 迁入前端 API、Store、页面、管理面板、弹窗、翻译和测试；页头内容替换为模块导出的 Header Action。
5. 以新 schema 文件重新生成 Ent；历史 173/177/179/181/185 均只读保留，后续改表只追加 *_custom_activity_*.sql。

~~~powershell
Set-Location backend
go generate ./ent
go generate ./cmd/server
go test -tags=unit ./internal/custom/modules/activity/...
go test -tags=integration ./internal/custom/modules/activity/...
Set-Location ../frontend
pnpm typecheck
pnpm test:run -- src/custom/modules/activity
~~~

额外验证：模块/子开关关闭时，导航、直达页面、公共 API、管理 API、后台任务和 Service 都拒绝执行；签到与红包并发/重试保持幂等，奖励投递可重试且不重复发奖。

## 6. wallet-extension

状态：**路由、前端入口和 Runtime 挂载已迁入；实现提取未完成**。`backend/internal/custom/modules/wallet-extension/**` 与 `frontend/src/custom/modules/wallet-extension/**` 已拥有模块入口、路由、导航、页面和测试；转账、收款方搜索、限额和红包仍通过共享 Service、Repository、Handler、Ent schema 与设置链路实现。

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | backend/internal/custom/modules/wallet-extension/**；frontend/src/custom/modules/wallet-extension/** | 当前模块入口、路由、导航、页面、组件、API、翻译和相邻测试。 |
| S | backend/internal/{handler/balance_transfer_handler.go,handler/admin/transfer_admin_handler.go,service/balance_transfer_service.go,service/balance_transfer_types.go,repository/balance_transfer_repo.go} 与测试 | 用户转账、收款方搜索、限额/手续费、红包领取关联与管理端操作的剩余共享实现。 |
| S | frontend/src/{api,views,components}/** | 历史转账 UI、API 和管理页面的剩余共享路径；后续仅可删除或迁入 custom。 |
| R | backend/internal/server/routes/user.go；frontend/src/components/layout/AppSidebar.vue；frontend/src/router/index.ts | 已移除转账、红包和管理入口的业务注册，只保留固定挂载。 |
| H | backend/migrations/173_port_balance_features.sql | 转账和红包表的历史来源，与 activity 共用，保持原样。 |
| G | backend/ent/schema/{balance_transfer.go,balance_red_packet.go,balance_red_packet_claim.go} 和相关 backend/ent/balance* | 迁为 custom_wallet_extension_*.go 后统一生成；不要手工改输出。 |
| S | backend/internal/service/setting_balance_features.go；frontend/src/utils/featureFlags.ts；frontend/src/i18n/locales/*/balanceFeatures.ts | 转账、红包和榜单开关/翻译当前在共享入口；收口为模块设置片段和 i18n 入口。 |

剩余迁移顺序：

1. 先定义 wallet-extension/contract 的用户查询、核心余额原子增减、事务和审计接口；不要复制上游 User Repository。
2. 迁入转账 Repository、Service、接收方搜索和幂等测试，验证收费、每日次数/额度与余额不为负。
3. 迁入 Handler、管理接口、用户/管理前端 API 与页面；删除直接路由和全局 Handler 字段。
4. 将转账榜仅作为模块路由与导航项；它使用 activity 排行榜时只依赖公开 Contract。
5. 移除上游 user_repo.go 中仅为转账新增的方法；schema/Ent 统一生成，历史 173 不变。

~~~powershell
Set-Location backend
go generate ./ent
go generate ./cmd/server
go test -tags=unit ./internal/custom/modules/wallet-extension/...
go test -tags=integration ./internal/custom/modules/wallet-extension/...
Set-Location ../frontend
pnpm typecheck
pnpm test:run -- src/custom/modules/wallet-extension
~~~

额外验证：相同幂等键重放不重复扣款；并发转账不能将可用余额扣为负；手续费、每日额度、冻结/撤销、红包领取和审计记录均正确；关闭模块后所有写路径不可达。

## 7. 共享挂载、生成物与归还清单

### 7.1 固定挂载白名单

下列上游文件最终只保留一次导入和一次注册/聚合调用，不得包含模块业务规则：

| 路径 | 当前下游耦合 | 目标剩余内容 |
| --- | --- | --- |
| backend/cmd/server/wire.go | 奖励/红包后台任务清理与全局构造 | 创建 custom.Runtime。 |
| backend/internal/server/http.go | 将 Overlay Runtime 交给 Router | 仅传递 custom.Runtime，不含业务规则。 |
| backend/internal/server/router.go | 注册既有路由后接入 Overlay | 一次 custom.RegisterRoutes(...)。 |
| frontend/src/router/index.ts | Dino、根首页、活动、转账、游戏大厅、功能守卫 | 一次 ...customRoutes；通用守卫若支持模块元数据，只调用 registry。 |
| frontend/src/components/layout/AppSidebar.vue | 各模块导航与功能开关 | 一次 customNavigation 聚合。 |
| frontend/src/components/layout/AppHeader.vue | 快捷签到、弹窗 | 仅一个 CustomHeaderActions 挂载点。 |
| frontend/src/i18n/locales/{en,zh}/index.ts 和 admin index | balanceFeatures 合并 | 一次 custom 语言片段入口。 |

路径级门禁按五类维护：固定挂载、composition root、受限平台端口、验证测试和已发布不可变迁移。精确路径由 `tools/check-custom-overlay-boundaries.ps1` 中的独立集合维护；固定挂载只允许一次注册/聚合，composition root 只负责构造，平台端口只能是通用小接口且不得导入 `custom` 业务。测试不承载生产规则；173--181、185、187 号迁移一旦在当前基线之后被修改、删除或重命名即失败。

backend/internal/{handler,service,repository}/wire.go 和 handler/handler.go 不在最终白名单；它们应归还上游，所有 Overlay 装配放在 backend/internal/custom/。

### 7.2 可再生输出

backend/ent/** 中除 schema/ 外的 **62 个差异路径**，以及 backend/cmd/server/wire_gen.go、wire_gen_test.go，都是生成结果。修改 schema 或 Provider 后执行：

~~~powershell
Set-Location backend
go generate ./ent
go generate ./cmd/server
~~~

只审查源 schema、Provider 和生成后的 diff；不手工解决生成文件的业务冲突。

### 7.3 历史迁移和 schema 兼容债务

所有 H 标记的迁移及其回归测试都是已发布历史：**只保留原路径并参与回归，绝不移动到 custom 目录、拆分、改号、重命名或改写。** 模块化只允许追加新的模块命名迁移，并在新迁移中完成兼容回填。门禁将相对旧基线已存在的这些文件标为 `published-immutable-migration`，但任何当前修改、删除或重命名都标为失败的 `immutable-migration-change`。

| 现状 | 规则 |
| --- | --- |
| 173_port_balance_features.sql 同时服务 activity 和 wallet-extension | 永不拆分、改号或重命名；新迁移按模块追加。 |
| 175 到 181 与 185 | 已发布数据升级步骤必须原样保留，相关 regression/integration test 不删除。 |
| backend/ent/schema/user.go | 当前包含余额精度和 activity/wallet 反向 edge；`game_hall_disabled` 已由 187 号迁移移出，不能重新加入用户 schema。 |
| backend/ent/schema/redeem_code.go | 当前增加幸运签到下注/倍率字段并扩大精度；迁移时以兼容 DTO 或模块表承接，不能直接丢字段。 |
| backend/internal/service/setting_balance_features.go、DTO、公开设置和前端 flags | 是四个模块的共享开关债务。先保留兼容读取，再逐模块迁出并最终由 custom 设置注册表聚合。 |

## 8. 非模块业务改动的处置

当前 `origin/main...HEAD` 仍有 187 个白名单外共享路径，主要集中在 `backend/internal/service`（49）、`frontend/src/views`（29）、`backend/internal/repository`（27）、`backend/internal/handler`（23）、`frontend/src/components`（20）、`frontend/src/i18n`（16）和 `frontend/src/api`（10）。这些不属于固定挂载点，不能再新增业务；现有债务只能删除或迁入模块。

1. 能确定来自上游同步的改动，在下一次 sync/upstream-v* 复核并归还/保留，不与模块迁移混提交。
2. 仅被某模块使用的改动，例如 userDisplay、红包兑换码格式，迁入对应模块或模块 Contract。
3. 被两个以上模块依赖的能力，先在 backend/internal/custom/contract/ 或 frontend/src/custom/registry.ts 建最小适配；没有第二个消费者时不抽象。
4. 一次迁移提交不得顺手重构 frontend/src/views/admin/**、backend/internal/service/** 等无关路径；留给单独清理提交并说明上游归还策略。

## 9. 每个模块合并前的共同门禁

~~~powershell
$ErrorActionPreference = 'Stop'
$repoGit = 'C:\Users\L\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\git\cmd\git.exe'

# 在仓库根目录以 c27d12376 审核检查点为起点，阻止本轮新增共享业务实现。
.\tools\check-custom-overlay-boundaries.ps1

Set-Location backend
go generate ./ent
go generate ./cmd/server
go test ./...

Set-Location ../frontend
pnpm typecheck
pnpm test:run
pnpm build
~~~

审查时将 diff 按本清单的 M/R/H/G/S 分类核对。若出现新上游路径，提交说明必须写明：为什么无法放入 Overlay、预期上游冲突位置和回滚方式；没有该说明即不合并。
