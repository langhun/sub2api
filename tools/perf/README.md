# Sub2API Performance Baseline

## 目标

`tools/perf` 提供一套面向 Sub2API 的可复用性能基线脚本，优先使用 [k6](https://grafana.com/docs/k6/latest/)；当团队已经安装 Vegeta 时，也可使用附带的 PowerShell 辅助脚本快速生成同类请求压测。

当前覆盖的基线场景：

- `health`: `GET /health`
- `pricing`: `GET /api/v1/public/pricing`
- `monitoring-summary`: `GET /api/v1/monitoring/summary`
- `mixed`: 按权重混合请求 `health + pricing + monitoring-summary`
- `all`: 统一入口，通过 `SCENARIO` 选择具体场景

这套脚本的定位是“建立基线”和“做回归对比”，不是直接替代完整容量规划。

## 目录结构

```text
tools/perf/
  README.md
  export-k6-trend.mjs
  k6/
    all.js
    health.js
    mixed.js
    pricing.js
    monitoring-summary.js
    lib/
      config.js
      helpers.js
      mixed.js
      scenarios.js
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
| `SCENARIO` | `health` | `all.js` 使用，支持 `health` / `pricing` / `monitoring-summary` / `mixed` |
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
| `MIXED_WEIGHT_HEALTH` | `60` | `mixed` 场景中 `health` 的权重 |
| `MIXED_WEIGHT_PRICING` | `30` | `mixed` 场景中 `pricing` 的权重 |
| `MIXED_WEIGHT_MONITORING_SUMMARY` | `10` | `mixed` 场景中 `monitoring-summary` 的权重 |
| `MIXED_PACE_MS` | 空 | `mixed` 场景固定节奏，单位毫秒；设置后优先于范围配置 |
| `MIXED_PACE_MIN_MS` | `0` | `mixed` 场景节奏下界，单位毫秒 |
| `MIXED_PACE_MAX_MS` | `0` | `mixed` 场景节奏上界，单位毫秒 |

说明：

- `monitoring-summary` 默认按公开接口处理，不强制要求鉴权。
- 如果后续环境把该接口挂到需要鉴权的入口，只需要设置 `AUTH_TOKEN` 或覆写 `AUTH_HEADER` / `AUTH_SCHEME`，不用改脚本。
- `EXTRA_HEADERS` 适合放灰度标记、租户头、追踪头等临时字段。
- `mixed` 场景每次迭代只发 1 个请求，请求类型由权重随机选出；若设置 `MIXED_PACE_MS` 或 `MIXED_PACE_MIN_MS/MAX_MS`，会在每次迭代后追加 sleep，用于模拟更稳定或更离散的请求节奏。

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

### 5. 混合基线场景

按权重混合三个只读端点：

```bash
k6 run tools/perf/k6/mixed.js
```

调整权重与节奏：

```bash
BASE_URL=http://127.0.0.1:18808 \
MIXED_WEIGHT_HEALTH=50 \
MIXED_WEIGHT_PRICING=35 \
MIXED_WEIGHT_MONITORING_SUMMARY=15 \
MIXED_PACE_MIN_MS=100 \
MIXED_PACE_MAX_MS=300 \
k6 run tools/perf/k6/mixed.js
```

PowerShell 示例：

```powershell
$env:BASE_URL='http://127.0.0.1:18808'
$env:MIXED_WEIGHT_HEALTH='50'
$env:MIXED_WEIGHT_PRICING='35'
$env:MIXED_WEIGHT_MONITORING_SUMMARY='15'
$env:MIXED_PACE_MIN_MS='100'
$env:MIXED_PACE_MAX_MS='300'
k6 run tools/perf/k6/mixed.js
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
- `mixed` 场景会在同一摘要后额外补充本次权重和节奏配置，方便横向对比

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

### GitHub Actions nightly

仓库已提供 `.github/workflows/perf-nightly.yml`，默认能力如下：

- 每天 UTC `18:30` 定时触发，换算为北京时间 `02:30`
- 支持手动 `workflow_dispatch`
- 固定跑 4 个场景：`health` / `pricing` / `monitoring-summary` / `mixed`
- 每个场景都会保留：
  - `*.summary.json`
  - `*.log`
- 汇总生成：
  - `perf-trend.md`
  - `perf-trend.csv`

#### 需要配置的仓库变量 / Secret

必填：

| 类型 | 名称 | 说明 |
| --- | --- | --- |
| Variable | `PERF_BASE_URL` | nightly 压测目标，例如 `https://perf.example.com` |

可选：

| 类型 | 名称 | 默认值 | 说明 |
| --- | --- | --- | --- |
| Variable | `PERF_K6_VUS` | `10` | nightly 默认并发 |
| Variable | `PERF_K6_DURATION` | `5m` | nightly 默认时长 |
| Variable | `PERF_K6_TIMEOUT` | `5s` | 单请求超时 |
| Variable | `PERF_K6_RPS` | `0` | 限速，`0` 表示不限速 |
| Variable | `PERF_EXPECTED_STATUS` | `200` | 期望 HTTP 状态码 |
| Variable | `PERF_AUTH_HEADER` | `Authorization` | 认证头名 |
| Variable | `PERF_AUTH_SCHEME` | `Bearer` | 认证前缀 |
| Variable | `PERF_EXTRA_HEADERS` | 空 | 额外请求头 JSON 字符串 |
| Variable | `PERF_K6_INSECURE_SKIP_TLS_VERIFY` | `false` | 是否跳过 TLS 校验 |
| Secret | `PERF_AUTH_TOKEN` | 空 | 需要鉴权时使用 |

说明：

- 如果 `schedule` 触发时没有配置 `PERF_BASE_URL`，workflow 会写明原因并跳过，不会误报“压测通过”。
- 如果手动触发且未提供 `base_url` 输入，也没有仓库级 `PERF_BASE_URL`，workflow 会直接报错。

#### Artifact 内容

nightly 运行后会上传两类 artifact：

- `perf-nightly-raw-<timestamp>-<sha12>`
  - `run-metadata.json`
  - `*.summary.json`
  - `*.log`
- `perf-nightly-trend-<timestamp>-<sha12>`
  - `perf-trend.md`
  - `perf-trend.csv`

#### 手动触发覆盖参数

在 GitHub Actions 页面手动运行时，可覆盖：

- `base_url`
- `k6_vus`
- `k6_duration`
- `k6_timeout`
- `k6_rps`

适合在同一套 workflow 下临时放大或缩小基线参数，而不改仓库文件。

### 趋势提取脚本

`tools/perf/export-k6-trend.mjs` 会从 `k6 --summary-export` 产出的 JSON 中提取以下字段：

- `scenario`
- `requests`
- `rps`
- `error_rate`
- `timeout_rate`
- `p95`
- `p99`
- `commit`
- `date`

本地手动生成趋势文件示例：

```bash
mkdir -p tools/perf/output/raw tools/perf/output/trend

k6 run tools/perf/k6/health.js \
  --summary-export tools/perf/output/raw/health.summary.json \
  > tools/perf/output/raw/health.log

node tools/perf/export-k6-trend.mjs \
  --input-dir tools/perf/output/raw \
  --output-dir tools/perf/output/trend \
  --metadata tools/perf/output/raw/run-metadata.json
```

如果没有 `run-metadata.json`，也可以只传 `--input-dir` 和 `--output-dir`；这时 `commit` / `date` 会回退到环境变量或留空。

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
