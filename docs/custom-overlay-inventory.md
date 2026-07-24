# 下游 Overlay 迁移现状清单

> 基线：product/main 的 e19d49fad；代码冻结标签 baseline/v0.1.164-before-overlay 指向 1f9aab7c4。
>
> 比对：origin/main...e19d49fad。本清单只描述当前已跟踪的下游差异，不包含未跟踪文件或工作区临时改动。

## 1. 使用方式与边界

当前差异共有 **352 个路径**：190 个新增、162 个修改。大量 backend/ent/** 是 schema 生成输出，不能按文件逐个迁移或手工解冲突；本清单按业务所有权和迁移动作聚合，列出的路径是每组关键入口和路径模式。

一次模块迁移的目标不是重构上游，而是：

1. 将本分支新增的业务实现移至 backend/internal/custom/modules/<module>/ 和 frontend/src/custom/modules/<module>/。
2. 同一提交删除该能力对上游 Handler、Service、Repository、routes/*.go、页面和布局的直接改动。
3. 只留下 custom 注册表、固定挂载点、追加式历史数据定义和重新生成的 Ent/Wire 输出。

模块化前先完成 feature/custom-overlay-bootstrap。它只能创建空注册表和挂载调用，不能移动下列任何业务实现或改数据库结构。

| 标记 | 含义 | 迁移处理 |
| --- | --- | --- |
| M | 模块自有业务 | 移入模块 custom 目录，并迁移相邻测试。 |
| R | 需归还的上游路径 | 模块迁完后恢复上游行为；仅固定挂载调用留在白名单。 |
| H | 已发布历史数据定义 | 不重命名、不拆分、不改写；新变更才使用 custom_<module> 命名。 |
| G | 可再生输出 | 不手工迁移；修改 schema/Provider 后统一生成。 |
| S | 共享适配层 | 收口为 custom 契约或注册表，不能继续扩散业务规则。 |

## 2. 模块总览

| 模块 | 当前能力 | 顺序 | 主要风险 |
| --- | --- | --- | --- |
| brand-home | 根首页选择、/Dino、Chromium Dino 资源与运行器 | 1 | 仅前端；保留 /home 默认行为和 /Dino 大小写路径。 |
| game-hall | DG 独立钱包、兑换、Slots、回合/流水、用户禁用和管理查询 | 2 | 余额事务、幂等、历史迁移和用户级开关。 |
| activity | 签到、幸运签到、盲盒、奖励投递、红包、排行榜和管理配置 | 3 | 后台任务、幂等、并发和跨模块余额写入。 |
| wallet-extension | 站内余额转账、收款方搜索、限额/手续费、管理和转账榜 | 4 | 直接改动核心余额、事务、审计和用户查询；最后迁移。 |

activity 与 wallet-extension 共享 173 号历史迁移和部分设置/余额契约，但不能合为一个模块；两者只能经 custom/contract/ 的最小接口通信。

## 3. brand-home

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | frontend/src/views/DaiGuaView.vue；frontend/src/components/game/ChromiumDinoRunner.vue | DaiGua 页面与运行器包装。 |
| M | frontend/src/games/chromiumDino/**（19 个 TypeScript 文件） | Chromium 离线 Dino 实现。 |
| M | frontend/public/dai-gua/**；frontend/src/assets/licenses/chromium-dino-LICENSE | 精灵图、音效与许可证；迁入后仍须按 Vite 静态资源规则引用。 |
| M | frontend/src/views/RootHomeView.vue | 根据 default_homepage 在 /home 和 /Dino 间跳转。 |
| R | frontend/src/router/index.ts | 当前直接定义 /Dino 和根路径 /；迁移后根路由只展开一次 customRoutes。 |
| S | frontend/src/stores/app.ts；后端设置 DTO/公开设置链路 | default_homepage 是品牌入口设置，须由 Overlay 设置片段提供，不能继续在通用设置对象扩散。 |

迁移顺序：

1. 在 frontend/src/custom/modules/brand-home/ 创建 routes.ts，先导出保持原值的 /Dino 和根路径处理路由。
2. 移入 DaiGua 页面、Runner、chromiumDino/**、资源和对应测试；保留静态资源 URL 与许可证。
3. 将根首页选择提炼为 Overlay 设置读取；默认值仍跳转 /home，只有明确为 dino 时跳转 /Dino。
4. 从根 router/index.ts 删除具体路由定义，只保留一次 ...customRoutes 挂载；不修改上游 /home 行为。

~~~powershell
Set-Location frontend
pnpm typecheck
pnpm test:run -- src/views/__tests__/RootHomeView.spec.ts
pnpm build
~~~

手工验证：未加载公开设置时 / 仍落到 /home；设置为 dino 时 / 到 /Dino；刷新 /Dino 后资源可加载并可开始游戏。

## 4. game-hall

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | backend/internal/{handler/game_hall_handler.go,handler/admin/game_hall_handler.go,service/game_hall_service.go,repository/game_hall_repo.go} 与相邻 *_test.go | 用户端、管理端、交易/回合业务和数据访问。 |
| M | frontend/src/{api/gameHall.ts,stores/gameHall.ts,views/user/GameHallView.vue} 与相邻测试 | 游戏大厅 API、状态和页面。 |
| M | frontend/src/components/admin/user/UserEditModal.vue；UserEditModal.gameHall.spec.ts | 用户级娱乐大厅禁用开关；应成为模块管理 UI。 |
| R | backend/internal/server/routes/{user.go,admin.go} | 当前直接注册 /game-hall 和 /admin/game-hall；迁移后由模块注册。 |
| R | backend/internal/{handler/wire.go,service/wire.go,repository/wire.go,handler/handler.go}；backend/cmd/server/wire.go | 当前将 GameHall 注入全局 Handlers 和清理流程；改为 custom.Runtime Provider 与模块生命周期。 |
| R | backend/internal/{handler/admin/user_handler.go,service/admin_user.go,repository/user_repo.go}；backend/ent/schema/user.go | game_hall_disabled 当前写入核心用户模型；迁移时改为 Overlay 专属禁用表/读取契约，或留下经评审的最小兼容适配。 |
| H | backend/migrations/175_add_game_hall_dedicated_tables.sql；176_backfill_game_hall_dedicated_balances.sql；178_add_game_hall_rounds.sql；180_add_user_game_hall_disabled.sql | 已有 game_hall 钱包、流水、奖池、回合与用户禁用数据；原样保留。 |
| H | backend/migrations/game_hall_migrations_regression_test.go；backend/internal/repository/game_hall_repo_integration_test.go | 保留为历史升级与事务回归门禁。 |
| S | backend/internal/service/setting_balance_features.go、公开设置、前端 feature flags/i18n | game_hall_enabled 与兑换/Slots 限制应收口为模块设置适配器。 |

当前游戏数据是原生 SQL 表，不是 Ent schema：game_hall_wallets、game_hall_wallet_transactions、game_hall_jackpots、game_hall_jackpot_transactions、game_hall_rounds、game_hall_main_balance_transactions。迁移服务代码时必须维持用户钱包、奖池与主余额在同一事务中的锁定与幂等语义。

迁移顺序：

1. 在 custom/modules/game-hall/contract 定义主余额读写、用户身份、审计和设置接口；模块不依赖上游具体 Repository。
2. 移入 Repository、Service 和事务/随机数测试，再由模块组装 Handler 与用户/管理路由。
3. 移入前端 API、Store、页面、用户禁用管理 UI、翻译和测试。
4. 从 routes/user.go、routes/admin.go、全局 Handlers 与 Wire Provider 删除游戏业务；只保留 custom.RegisterRoutes 和 custom.Runtime。
5. 历史迁移保留原文件名；以后需要独立禁用表或配置表时，才追加 *_custom_game_hall_*.sql。

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

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | backend/internal/{handler/checkin_handler.go,handler/leaderboard_handler.go,handler/admin/blindbox_handler.go} | 签到、榜单、盲盒和管理 API。 |
| M | backend/internal/service/{checkin_service.go,blindbox_service.go,leaderboard_service.go,reward_delivery*.go,red_packet_expiry_service.go} 与测试 | 签到、盲盒、榜单、奖励投递 Outbox 和红包到期后台任务。 |
| M | backend/internal/repository/{balance_redpacket_repo.go,reward_delivery_repo.go} 与测试 | 红包与可靠奖励投递持久化。 |
| M | frontend/src/{api/checkin.ts,api/leaderboard.ts,stores/checkin.ts,views/LeaderboardView.vue,views/user/CheckinView.vue,views/user/RedPacketView.vue} | 用户签到、排行榜和红包体验。 |
| M | frontend/src/components/{checkin/LuckyCheckinDialog.vue,user/profile/BlindboxModal.vue,admin/BlindboxPrizePoolCard.vue,admin/RewardDeliveryOpsPanel.vue} | 快捷签到、盲盒、奖池和奖励投递管理 UI。 |
| M | frontend/src/utils/{activityError.ts,checkinCalendar.ts} 与相关 __tests__ | 模块可携带的前端辅助逻辑与测试。 |
| R | backend/internal/server/routes/{common.go,user.go,admin.go} | 当前直接挂载公共榜单、签到、盲盒和奖励投递接口。 |
| R | frontend/src/components/layout/AppHeader.vue | 当前包含签到按钮、弹窗和 Store 调用；迁移后只留 CustomHeaderActions 挂载点。 |
| R | frontend/src/components/layout/AppSidebar.vue；frontend/src/router/index.ts | 当前直接添加签到、红包、榜单入口和路由；应由 registry 聚合。 |
| H | backend/migrations/173_port_balance_features.sql | 同时包含签到、盲盒、转账和红包的首批结构与设置；不可拆分或改名。 |
| H | backend/migrations/177_migrate_legacy_checkin_records.sql；179_repair_legacy_checkin_migration.sql；181_create_reward_deliveries.sql；185_widen_large_balance_amount_columns.sql | 已有签到历史回填、修复、奖励 Outbox 和精度升级；必须原样保留。 |
| G | backend/ent/schema/{checkin.go,checkin_blindbox_record.go,checkin_prize_item.go} 和相关 backend/ent/{checkin*,checkinblindboxrecord*,checkinprizeitem*} | schema 改为 custom_activity_*.go 后生成；62 个 Ent 非 schema 差异路径均视为输出，不手工迁移。 |
| S | backend/ent/schema/{user.go,redeem_code.go}；backend/internal/service/{redeem_code.go,redeem_service.go,code_format*.go} | 当前活动扩展了上游 User/RedeemCode 与兑换码格式。迁移时优先模块 DTO 和 Core 适配接口；无法解除时列为兼容债务。 |

迁移顺序：

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

| 标记 | 路径/模式 | 说明 |
| --- | --- | --- |
| M | backend/internal/{handler/balance_transfer_handler.go,handler/admin/transfer_admin_handler.go,service/balance_transfer_service.go,service/balance_transfer_types.go,repository/balance_transfer_repo.go} 与测试 | 用户转账、收款方搜索、限额/手续费、红包领取关联与管理端操作。 |
| M | frontend/src/{api/transfer.ts,api/admin/transfer.ts,views/user/TransferView.vue,views/user/TransferLeaderboardView.vue,views/admin/TransferManageView.vue} 与测试 | 用户端与管理端转账 UI。 |
| M | frontend/src/api/__tests__/transfer.idempotency.spec.ts | 前端幂等请求契约。 |
| R | backend/internal/server/routes/{user.go,admin.go} | 当前直接注册 /transfer、/redpacket 和 /admin/transfers。 |
| R | backend/internal/repository/user_repo.go；backend/internal/service/user.go；backend/internal/handler/dto/* | 当前为转账直接把收款人搜索和余额更新接口加入 core；迁移后经 wallet-extension/contract 调用最小用户/余额端口。 |
| R | frontend/src/components/layout/AppSidebar.vue；frontend/src/router/index.ts | 当前直接添加转账、红包和管理入口；应只由 customNavigation/customRoutes 提供。 |
| H | backend/migrations/173_port_balance_features.sql | 转账和红包表的历史来源，与 activity 共用，保持原样。 |
| G | backend/ent/schema/{balance_transfer.go,balance_red_packet.go,balance_red_packet_claim.go} 和相关 backend/ent/balance* | 迁为 custom_wallet_extension_*.go 后统一生成；不要手工改输出。 |
| S | backend/internal/service/setting_balance_features.go；frontend/src/utils/featureFlags.ts；frontend/src/i18n/locales/*/balanceFeatures.ts | 转账、红包和榜单开关/翻译当前在共享入口；收口为模块设置片段和 i18n 入口。 |

迁移顺序：

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

| 现状 | 规则 |
| --- | --- |
| 173_port_balance_features.sql 同时服务 activity 和 wallet-extension | 永不拆分、改号或重命名；新迁移按模块追加。 |
| 175 到 181 与 185 | 已发布数据升级步骤必须原样保留，相关 regression/integration test 不删除。 |
| backend/ent/schema/user.go | 当前包含余额精度、game_hall_disabled 和 activity/wallet 反向 edge；这是需要逐步归还的核心污染区。 |
| backend/ent/schema/redeem_code.go | 当前增加幸运签到下注/倍率字段并扩大精度；迁移时以兼容 DTO 或模块表承接，不能直接丢字段。 |
| backend/internal/service/setting_balance_features.go、DTO、公开设置和前端 flags | 是四个模块的共享开关债务。先保留兼容读取，再逐模块迁出并最终由 custom 设置注册表聚合。 |

## 8. 非模块业务改动的处置

origin/main...e19d49fad 还包含用户展示、审计搜索、兑换码格式、用量查询开关、支付/后台页面细节和多次上游同步带来的修改。这些不属于四个 Overlay 模块的直接实现，不能悄悄塞进任一模块。

1. 能确定来自上游同步的改动，在下一次 sync/upstream-v* 复核并归还/保留，不与模块迁移混提交。
2. 仅被某模块使用的改动，例如 userDisplay、红包兑换码格式，迁入对应模块或模块 Contract。
3. 被两个以上模块依赖的能力，先在 backend/internal/custom/contract/ 或 frontend/src/custom/registry.ts 建最小适配；没有第二个消费者时不抽象。
4. 一次迁移提交不得顺手重构 frontend/src/views/admin/**、backend/internal/service/** 等无关路径；留给单独清理提交并说明上游归还策略。

## 9. 每个模块合并前的共同门禁

~~~powershell
$ErrorActionPreference = 'Stop'
$repoGit = 'C:\Users\L\.cache\codex-runtimes\codex-primary-runtime\dependencies\native\git\cmd\git.exe'

# 确认本模块没有继续污染白名单外的上游路径。
& $repoGit diff --name-only origin/main...HEAD

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
