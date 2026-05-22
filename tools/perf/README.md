# Sub2API Performance Baseline

## 目标

`tools/perf` 提供一套面向 Sub2API 的可复用性能基线脚本，优先使用 [k6](https://grafana.com/docs/k6/latest/)；当团队已经安装 Vegeta 时，也可使用附带的 PowerShell 辅助脚本快速生成同类请求压测。

当前覆盖的基线场景：

- `health`: `GET /health`
- `pricing`: `GET /api/v1/public/pricing`
- `monitoring-summary`: `GET /api/v1/monitoring/summary`
- `all`: 统一入口，通过 `SCENARIO` 选择具体场景

这套脚本的定位是“建立基线”和“做回归对比”，不是直接替代完整容量规划。

## 目录结构

```text
tools/perf/
  README.md
  k6/
    all.js
    health.js
    pricing.js
    monitoring-summary.js
    lib/
      config.js
      helpers.js
      summary.js
  vegeta/
    run-baseline.ps1
```

## 前置依赖

### k6

推荐安装官方 k6：

- macOS: `brew install k6`
- Windows: `choco install k6` 或 `winget install k6 --source winget`
- Linux: 参考官方文档安装

安装完成后确认：

```bash
k6 version
```

### Vegeta（可选）

仅在你们已经使用 Vegeta 时作为补充工具：

```bash
vegeta -version
```

## 环境变量

所有脚本统一读取以下环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `BASE_URL` | `http://127.0.0.1:18808` | 压测目标根地址，建议先指向隔离实例 |
| `SCENARIO` | `health` | `all.js` 使用，支持 `health` / `pricing` / `monitoring-summary` |
| `K6_VUS` | `5` | 并发虚拟用户数 |
| `K6_DURATION` | `30s` | 持续时间，例如 `30s` / `2m` |
| `K6_TIMEOUT` | `5s` | 单请求超时 |
| `K6_RPS` | `0` | 限速，`0` 表示不限速 |
| `K6_INSECURE_SKIP_TLS_VERIFY` | `false` | 是否跳过 TLS 校验 |
| `AUTH_TOKEN` | 空 | 可选 Bearer Token |
| `AUTH_HEADER` | `Authorization` | 自定义认证头名 |
| `AUTH_SCHEME` | `Bearer` | 认证头前缀，留空则只发送 token |
| `EXTRA_HEADERS` | 空 | 额外请求头，JSON 字符串，例如 `{"X-Debug":"1"}` |
| `EXPECTED_STATUS` | `200` | 期望状态码 |
| `SUMMARY_TREND_STATS` | `avg,min,med,max,p(50),p(95),p(99)` | k6 趋势指标输出项 |

说明：

- `monitoring-summary` 默认按公开接口处理，不强制要求鉴权。
- 如果后续环境把该接口挂到需要鉴权的入口，只需要设置 `AUTH_TOKEN` 或覆写 `AUTH_HEADER` / `AUTH_SCHEME`，不用改脚本。
- `EXTRA_HEADERS` 适合放灰度标记、租户头、追踪头等临时字段。

## 运行命令

### 1. 健康检查基线

```bash
k6 run tools/perf/k6/health.js
```

自定义并发和时长：

```bash
BASE_URL=http://127.0.0.1:18808 K6_VUS=20 K6_DURATION=1m k6 run tools/perf/k6/health.js
```

PowerShell 示例：

```powershell
$env:BASE_URL='http://127.0.0.1:18808'
$env:K6_VUS='20'
$env:K6_DURATION='1m'
k6 run tools/perf/k6/health.js
```

### 2. 公开定价接口基线

```bash
BASE_URL=http://127.0.0.1:18808 k6 run tools/perf/k6/pricing.js
```

### 3. 监控摘要接口基线

公开模式：

```bash
BASE_URL=http://127.0.0.1:18808 k6 run tools/perf/k6/monitoring-summary.js
```

带鉴权占位：

```bash
BASE_URL=http://127.0.0.1:18808 AUTH_TOKEN=replace-me k6 run tools/perf/k6/monitoring-summary.js
```

### 4. 统一入口

一次性运行某个场景，但统一使用 `all.js` 入口：

```bash
SCENARIO=pricing k6 run tools/perf/k6/all.js
```

## 指标说明

脚本重点关注以下统一指标：

- `P50`: 50% 请求在该响应时延内完成，反映常态体验
- `P95`: 95% 请求在该响应时延内完成，适合作为常用预警线
- `P99`: 99% 请求在该响应时延内完成，能更敏感地暴露尾延迟
- `错误率`: 非预期状态码、断言失败、网络异常占总请求比例
- `超时率`: 因超时、中断、传输层失败导致的失败比例
- `吞吐`: 单位时间完成的请求数，主要观察 `http_reqs` 和场景总量

### 脚本摘要会打印什么

每个 k6 脚本都会在运行结束时输出自定义 `PERF_BASELINE_SUMMARY`，包含：

- 场景名
- 目标 URL
- 请求总数
- 吞吐（requests/sec）
- 错误率
- 超时率
- P50 / P95 / P99

## 结果解读建议

### 健康检查

- 应接近“零业务负担”的最低成本接口
- 适合作为进程、网关、反向代理的最小延迟基线
- 如果 `health` 都出现明显尾延迟，优先排查网络、TLS、代理层或机器资源争用

### 公开定价接口

- 反映公开读接口在数据库 / 缓存 / 序列化链路上的基础表现
- 如果 P95/P99 抖动明显，但 `health` 正常，优先查业务查询与数据准备链路

### 监控摘要接口

- 适合作为“较重只读接口”的基线
- 若它明显慢于 `pricing`，通常是聚合查询、统计窗口或数据库扫描成本更高
- 建议与真实业务低峰 / 高峰分别建立基线，便于后续回归比较

## CI 与本地使用建议

### 本地

建议先做本地最小基线，再做变更前后对比：

1. 启动隔离实例，例如 `18808`
2. 固定同一组参数，例如 `K6_VUS=10 K6_DURATION=30s`
3. 记录变更前结果
4. 完成改动后重跑同一命令
5. 对比 P95/P99、错误率、超时率、吞吐

### CI

建议只在以下条件下接入：

- 有稳定、可重复的测试环境
- 有固定数据集或已知环境噪声范围
- 团队接受“性能回归阈值”是软门禁还是硬门禁

建议做法：

- PR 或 nightly job 只跑短时基线，例如 `K6_VUS=5 K6_DURATION=15s`
- 将摘要结果归档为 artifact
- 如果后续需要门禁，可在 CI 中比对 `error_rate`、`timeout_rate`、`p(95)` 的阈值

## Vegeta 辅助脚本

`tools/perf/vegeta/run-baseline.ps1` 适合已经装好 Vegeta 的 Windows 环境，用法：

```powershell
$env:BASE_URL='http://127.0.0.1:18808'
$env:SCENARIO='pricing'
$env:VEGETA_RATE='50'
$env:VEGETA_DURATION='30s'
powershell -ExecutionPolicy Bypass -File tools/perf/vegeta/run-baseline.ps1
```

脚本会：

- 根据场景拼接目标 URL
- 自动附加鉴权头和额外请求头
- 将结果写到 `tools/perf/vegeta/output/`
- 输出 `vegeta report` 摘要

## 建议的基线留档方式

每次留档至少记录：

- 日期与环境
- Git commit
- 场景名
- `BASE_URL`
- `K6_VUS` / `K6_DURATION` / `K6_TIMEOUT`
- P50 / P95 / P99
- 错误率 / 超时率 / 吞吐
- 特殊背景，例如“刚重启服务”“数据库在冷缓存状态”

这样后续看性能回归时，能快速判断是代码变化还是环境噪声。
