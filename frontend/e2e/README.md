# Playwright E2E

这套 E2E 基础设施只放在 `frontend/` 下，避免侵入业务代码。

## 覆盖范围

- `auth.setup.js`: 通过真实登录页 + API mock 生成管理员 `storageState`
- `admin-login.spec.js`: 验证登录后进入管理台
- `admin-flow-smoke.spec.js`: 串联 dashboard -> settings(payment) -> accounts -> groups -> proxies，并验证 payment provider 状态在跨页返回后仍可见
- `admin-groups.spec.js`: 验证分组页创建分组成功流程
- `admin-groups-accounts-linkage.spec.js`: 验证分组创建后能在账户页分组筛选中出现、筛选生效、清空筛选后列表恢复
- `admin-accounts.spec.js`: 验证账户页搜索、高级筛选展开与清空筛选
- `admin-proxies.spec.js`: 验证代理页关键入口，并覆盖创建代理后立即可见、状态切换成功的回归流程
- `admin-settings.spec.js`: 验证设置页加载、切换 tab、保存成功
- `admin-settings-payment.spec.js`: 验证支付设置页启用支付、启用支付类型、创建服务商、列表展示与删除流程
- `admin-settings-admin-api-key.spec.js`: 验证安全页管理员 API Key 的创建、刷新后脱敏展示、重新生成与删除流程
- `mobile-smoke.spec.js`: 验证移动端可打开菜单并进入后台页面

## 设计约束

- 优先使用已有稳定选择器：`role`、输入 `id`、按钮文案、现有 `data-test`
- 不修改业务源码，不新增前端组件层 `data-testid`
- 通过 `page.route()` 统一 mock 管理后台依赖接口，避免用例依赖后端环境
- `support/mock-api.js` 当前已覆盖支付设置保存与支付服务商 CRUD 的最小状态化 mock，适合设置页 E2E happy path

## 运行

优先使用仓库约定的 pnpm：

```bash
pnpm run e2e
pnpm run e2e:list
pnpm run e2e:headed
```

如果本机没有 `pnpm` 在 PATH，需要先补齐 Node 包管理器环境，再安装 `@playwright/test`。

如只做快速语法检查，可先验证：

```bash
node --check playwright.config.js
node --check e2e/support/mock-api.js
node --check e2e/support/fixtures.js
```
