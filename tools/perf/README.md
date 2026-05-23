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
  check-threshold-report.mjs
  thresholds.mjs
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

### 本地 readiness 校验

如果你刚改过 `.github/workflows/perf-nightly.yml` 或 `.github/workflows/perf-long-run.yml`，建议先在本地跑一次：

```bash
node tools/perf/verify-workflow-readiness.mjs
```

这个脚本不会执行 workflow，只会做只读校验，重点检查：

- `history_source` / `history_run_id` / `history_artifact_name` / `history_csv_path` 输入是否还在
- `fallback_reason` 是否仍通过 `GITHUB_OUTPUT` 向后续步骤暴露
- `--history-csv "$HISTORY_CSV_PATH"` 是否仍传给趋势导出脚本
- `perf-trend-latest.md` 是否仍会在存在时追加到 `GITHUB_STEP_SUMMARY`
- raw / trend artifact 命名是否仍保持 `perf-nightly-*` / `perf-long-run-*` 约定

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
  - `perf-trend-history.csv`（启用历史对比时）
  - `perf-trend-latest.md`（启用历史对比时）
  - `perf-thresholds.md`
  - `perf-threshold-report.json`
  - `perf-stability-summary.md`
  - `perf-stability-summary.json`

#### 手动触发覆盖参数

在 GitHub Actions 页面手动运行时，可覆盖：

- `base_url`
- `k6_vus`
- `k6_duration`
- `k6_timeout`
- `k6_rps`
- `history_source`
- `history_run_id`
- `history_artifact_name`
- `history_csv_path`

适合在同一套 workflow 下临时放大或缩小基线参数，而不改仓库文件。

#### nightly 历史对比

`Perf Nightly` 现在也支持把历史 CSV 传给 `export-k6-trend.mjs --history-csv`，用于和上一轮或指定基线做连续对比。若选择 `previous-run-artifact` 或 `artifact`，历史 artifact 的解析与下载由 workflow 在 GitHub Actions runner 内通过 GitHub REST API 自动完成；本地查看文档或手动触发 workflow 不要求预装 `gh` CLI。默认策略保持安全兼容：

- `schedule` 触发时固定按 `history_source=previous-run-artifact`
- 手动 `workflow_dispatch` 时可显式选择历史源
- 历史不可用时只发 warning，并安全降级为当前运行单独输出

可选历史源与 `Perf Long Run` 对齐：

- `previous-run-artifact`
  - 默认值
  - 自动查找当前 workflow 最近一次成功运行的 `perf-nightly-trend-*` artifact（runner 内自动处理）
- `artifact`
  - 手动指定某个 nightly 历史 run 的 artifact（runner 内自动处理）
- `path`
  - 使用工作区中已有的 CSV 路径
- `none`
  - 明确禁用历史对比

输入组合非法时，workflow 会给 warning 但不会失败：

- `history_source=previous-run-artifact`
  - 忽略 `history_run_id`
  - 忽略 `history_artifact_name`
  - 忽略 `history_csv_path`
- `history_source=artifact`
  - 忽略 `history_csv_path`
- `history_source=path`
  - 忽略 `history_run_id`
  - 忽略 `history_artifact_name`
- `history_source=none`
  - 忽略任意其他 `history_*`

历史相关 summary 会额外输出：

- 最终 `history_source`
- 选中的 history run / branch / event / artifact
- 最终使用的 history CSV path
- 最终 fallback reason

这样即使历史下载失败、artifact 过期、路径不存在，nightly 仍会继续产出：

- `perf-trend.md`
- `perf-trend.csv`
- `perf-thresholds.md`
- `perf-stability-summary.md`

### GitHub Actions longer-run

仓库还提供独立的 `.github/workflows/perf-long-run.yml`，用于和 nightly 解耦的长时压测守护。默认能力如下：

- 每周 UTC `19:00` 定时触发，换算为北京时间周一 `03:00`
- 支持手动 `workflow_dispatch`
- 固定跑同一组 4 个场景：`health` / `pricing` / `monitoring-summary` / `mixed`
- 默认参数更偏向长稳态：
  - `K6_VUS=25`
  - `K6_DURATION=15m`
  - `K6_TIMEOUT=5s`
  - `K6_RPS=0`
- 复用与 nightly 相同的趋势导出、阈值评估、稳定性总结逻辑
- 使用独立 artifact 前缀和输出目录，不会和 nightly 混在一起
- 默认非阻塞：
  - workflow 级别 `continue-on-error` 会在手动输入 `enforce_thresholds=false` 时生效
  - 阈值检查默认也按 warning 模式运行，除非显式开启强制模式

#### longer-run 变量与默认值

long-run 优先读取以下仓库变量：

| 类型 | 名称 | 默认值 | 说明 |
| --- | --- | --- | --- |
| Variable | `PERF_LONGRUN_BASE_URL` | 回退到 `PERF_BASE_URL` | long-run 默认目标地址 |
| Variable | `PERF_LONGRUN_K6_VUS` | `25` | long-run 默认并发 |
| Variable | `PERF_LONGRUN_K6_DURATION` | `15m` | long-run 默认时长 |
| Variable | `PERF_LONGRUN_K6_TIMEOUT` | `5s` | 单请求超时 |
| Variable | `PERF_LONGRUN_K6_RPS` | `0` | 限速，`0` 表示不限速 |
| Variable | `PERF_LONGRUN_THRESHOLDS` | 空 | long-run 专用阈值 JSON |
| Variable | `PERF_LONGRUN_ENFORCE` | `false` | 是否把阈值失败升级为阻塞失败 |

说明：

- 如果 `PERF_LONGRUN_BASE_URL` 未配置，会自动回退到 `PERF_BASE_URL`
- 鉴权相关变量与 Secret 仍复用 nightly 那套：
  - `PERF_AUTH_TOKEN`
  - `PERF_AUTH_HEADER`
  - `PERF_AUTH_SCHEME`
  - `PERF_EXTRA_HEADERS`
  - `PERF_EXPECTED_STATUS`
  - `PERF_K6_INSECURE_SKIP_TLS_VERIFY`
- 如果 weekly schedule 触发时目标地址仍为空，workflow 会写明原因并跳过

#### 手动触发 longer-run

在 GitHub Actions 页面选择 `Perf Long Run` 后，可覆盖：

- `base_url`
- `k6_vus`
- `k6_duration`
- `k6_timeout`
- `k6_rps`
- `enforce_thresholds`

推荐用法：

- 常规周守护：直接等待 schedule，使用仓库变量默认值
- 临时放大压测：手动把 `k6_vus`、`k6_duration` 提高
- 想让本次结果真正阻塞：手动把 `enforce_thresholds` 设为 `true`

#### longer-run artifact 内容

long-run 运行后会上传两类独立 artifact：

- `perf-long-run-raw-<timestamp>-<sha12>`
  - `run-metadata.json`
  - `*.summary.json`
  - `*.log`
- `perf-long-run-trend-<timestamp>-<sha12>`
  - `perf-trend.md`
  - `perf-trend.csv`
  - `perf-trend-history.csv`（启用历史对比时）
  - `perf-trend-latest.md`（启用历史对比时）
  - `perf-thresholds.md`
  - `perf-threshold-report.json`
  - `perf-stability-summary.md`
  - `perf-stability-summary.json`

#### 如何启用历史对比

`Perf Long Run` 已支持把历史 CSV 输入接到 `export-k6-trend.mjs --history-csv`，用于跨运行对比与更连续的稳定性分析。若选择 `previous-run-artifact` 或 `artifact`，历史 artifact 的解析与下载由 workflow 在 GitHub Actions runner 内通过 GitHub REST API 自动完成；本地阅读文档或手动触发 workflow 不要求预装 `gh` CLI。默认策略是“尽量使用历史，但拿不到时安全降级为当前运行单独输出”。

可选历史源有 4 种：

- `previous-run-artifact`
  - 默认值
  - 自动查找当前 workflow 最近一次成功运行的 `perf-long-run-trend-*` artifact（runner 内自动处理）
- `artifact`
  - 手动指定某个历史 workflow run 的 artifact（runner 内自动处理）
- `path`
  - 使用仓库工作区中已有的 CSV 路径
- `none`
  - 明确禁用历史对比

##### 方案 1：默认自动接上一轮 long-run artifact

这是最省心的方式：

1. 保持 `history_source=previous-run-artifact`
2. 先至少成功跑过一次 `Perf Long Run`
3. 从第二次开始，workflow 会自动尝试下载上一轮成功运行的 `perf-long-run-trend-*`
4. 自动选择历史 run 时会优先按下面的顺序过滤：
   - 同分支 + 同事件类型
   - 同分支
   - 最近一次成功运行
4. 如果下载到：
   - 优先使用 `perf-trend-history.csv`
   - 否则回退使用 `perf-trend.csv`
5. 如果上一轮 artifact 不存在、已过期或下载失败：
   - workflow 会写 warning
   - 但仍继续产出本轮 `perf-trend.md` / `perf-trend.csv`
6. `GITHUB_STEP_SUMMARY` 会额外写出：
   - 选中的 history run id
   - 选中的 history branch
   - 选中的 history event
   - `Selection basis`
   - 最终使用的 history CSV path
   - 最终 fallback reason

##### 方案 2：手动指定历史 artifact

适合你想和某个固定 run 做对比时使用：

1. 在 GitHub Actions 页面手动运行 `Perf Long Run`
2. 设置：
   - `history_source=artifact`
   - `history_run_id=<某次历史 run id>`
3. 可选再填：
   - `history_artifact_name=<精确 artifact 名>`
4. 如果 `history_artifact_name` 留空，workflow 会自动在该 run 中查找最新的 `perf-long-run-trend-*`
5. 若 run id 或 artifact 找不到：
   - workflow 会安全降级
   - 不会因为缺历史而阻断整次压测
6. 如果同时传了 `history_csv_path`：
   - workflow 会提前给 warning
   - 但仍按 `artifact` 模式继续

##### 方案 3：手动指定 CSV 路径

适合你把历史 CSV 先放到工作区后再运行，例如自定义调试或从别处复制来的基线：

1. 手动运行 `Perf Long Run`
2. 设置：
   - `history_source=path`
   - `history_csv_path=<工作区相对路径或绝对路径>`
3. 推荐传入：
   - 某次导出的 `perf-trend-history.csv`
   - 或单次 `perf-trend.csv`
4. 如果该路径不存在：
   - workflow 会写 warning
   - 然后自动回退到 current-only 模式
5. 如果同时传了 `history_run_id` 或 `history_artifact_name`：
   - workflow 会提前给 warning
   - 但仍按 `path` 模式继续

##### 方案 4：显式关闭历史对比

如果你只想看本轮数据：

1. 手动运行 `Perf Long Run`
2. 设置 `history_source=none`
3. workflow 不会尝试下载或读取任何历史 CSV
4. 如果此时仍填写其他 `history_*` 输入：
   - workflow 会提前给 warning
   - 但仍按禁用历史处理

##### 输入校验与安全降级说明

为了避免误配，workflow 会对以下组合提前给 warning，但不会直接失败：

- `history_source=previous-run-artifact`
  - 同时填写 `history_run_id`
  - 同时填写 `history_artifact_name`
  - 同时填写 `history_csv_path`
- `history_source=artifact`
  - 同时填写 `history_csv_path`
- `history_source=path`
  - 同时填写 `history_run_id`
  - 同时填写 `history_artifact_name`
- `history_source=none`
  - 同时填写任意其他 `history_*`

这类 warning 只用于提示“哪些输入会被忽略”，不会阻断当前 long-run。

##### 启用历史对比后会多出什么产物

当 `--history-csv` 实际传入后，trend artifact 中会额外出现：

- `perf-trend-history.csv`
  - 历史与当前合并后的完整趋势 CSV
- `perf-trend-latest.md`
  - 当前样本与上一个同场景样本的 delta 对比

同时，稳定性汇总步骤会优先使用：

- `perf-trend-history.csv`

如果历史未启用或未获取成功，才回退使用：

- `perf-trend.csv`

##### summary 中会看到什么

历史相关的 `GITHUB_STEP_SUMMARY` 现在会明确输出：

- 当前 branch
- 当前 event type
- 选中的 history run id
- 选中的 history branch
- 选中的 history event
- `Selection basis`
- 选中的 history artifact 名称
- 最终使用的 history CSV path
- 最终 `fallback reason`

这样即使历史不可用，也能快速看出：

- 是没有同分支历史
- 还是没有 matching artifact
- 还是 artifact 下载失败
- 还是路径输入不存在
- 还是明确被 `none` 禁用

#### 如何读取 longer-run 产物

建议按以下顺序看：

1. `perf-trend.md`
   - 先看 4 个场景的 `requests`、`rps`、`error_rate`、`timeout_rate`、`p95`、`p99`
2. `perf-trend-latest.md`
   - 如果启用了历史对比，优先看它给出的当前值与上一轮同场景 delta
3. `perf-thresholds.md`
   - 看当前是否命中阈值，以及哪些指标偏离
4. `perf-stability-summary.md`
   - 看最近多次运行的稳定性汇总，判断是否是偶发抖动
5. `run-metadata.json`
   - 确认本次 `base_url`、`vus`、`duration`、`timeout`、`rps`、commit 和 run id
6. 单场景 `*.log`
   - 当某个场景异常时，再下钻看对应原始日志

如果你要做长期观察，最适合消费的是：

- `perf-trend.csv`：适合导入表格、BI 或外部报表
- `perf-trend-history.csv`：适合保存连续多轮 long-run 趋势
- `perf-threshold-report.json`：适合做自动判定或外部通知
- `perf-stability-summary.json`：适合做“最近 N 次”波动分析

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
- `threshold_status`
- `threshold_failed_metrics`
- 各阈值对应的 `*_threshold` / `*_check`

如果提供阈值配置，它还会额外生成：

- `perf-thresholds.md`
- `perf-threshold-report.json`

### 阈值配置

nightly workflow 与本地脚本共用同一套阈值模型，支持 4 个指标：

- `error_rate`
- `timeout_rate`
- `p95`
- `p99`

其中：

- `error_rate` / `timeout_rate` 使用小数比例，例如 `0.01` 代表 `1%`
- `p95` / `p99` 使用毫秒值，例如 `350`
- 所有阈值语义均为“实际值必须小于等于阈值”
- 可以先定义默认阈值，再按 scenario 覆盖

#### 推荐阈值模板

仓库已提供可直接复用的模板文件：

- `tools/perf/nightly-thresholds.example.json`

推荐用法：

1. 先复制模板为你自己的基线版本，例如 `tools/perf/output/raw/thresholds.json` 或本地任意不提交路径
2. 先按最近一轮稳定 nightly / 预发基线填入默认值
3. 再只对波动明显不同的场景单独覆盖，例如 `health` 更严格、`mixed` 更宽松
4. 先以 `PERF_NIGHTLY_ENFORCE=false` 跑几轮观察，再决定是否切到强制模式

这个模板覆盖了当前 nightly 固定执行的 4 个 scenario：

- `health`
- `pricing`
- `monitoring-summary`
- `mixed`

示例：

```json
{
  "default": {
    "error_rate": 0.01,
    "timeout_rate": 0.005,
    "p95": 400,
    "p99": 800
  },
  "scenarios": {
    "health": {
      "p95": 150,
      "p99": 300
    },
    "mixed": {
      "error_rate": 0.02,
      "p95": 600,
      "p99": 1200
    }
  }
}
```

本地可通过以下方式传入：

- `--thresholds <path-to-json>`
- `--thresholds-json '<json-string>'`
- 环境变量 `PERF_NIGHTLY_THRESHOLDS`

示例：

```bash
node tools/perf/export-k6-trend.mjs \
  --input-dir tools/perf/output/raw \
  --output-dir tools/perf/output/trend \
  --metadata tools/perf/output/raw/run-metadata.json \
  --thresholds tools/perf/output/raw/thresholds.json
```

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

### nightly 阈值守护

`.github/workflows/perf-nightly.yml` 默认会在导出趋势后执行阈值检查，并把结果追加到 `GITHUB_STEP_SUMMARY`。

相关仓库变量：

| 类型 | 名称 | 默认值 | 说明 |
| --- | --- | --- | --- |
| Variable | `PERF_NIGHTLY_THRESHOLDS` | 空 | 阈值 JSON 字符串，格式同上 |
| Variable | `PERF_NIGHTLY_ENFORCE` | `false` | 是否把阈值失败升级为阻塞失败 |

#### 如何落地到仓库变量 `PERF_NIGHTLY_THRESHOLDS`

推荐步骤：

1. 打开 `tools/perf/nightly-thresholds.example.json`
2. 按你的服务基线调整默认值与各 scenario 覆盖值
3. 将文件内容压缩成单行合法 JSON
4. 进入 GitHub 仓库 `Settings -> Secrets and variables -> Actions -> Variables`
5. 新建或更新变量 `PERF_NIGHTLY_THRESHOLDS`
6. 把单行 JSON 粘贴进去保存
7. 初次启用时，建议同时确认 `PERF_NIGHTLY_ENFORCE=false`
8. 手动触发一次 `Perf Nightly`，在 `GITHUB_STEP_SUMMARY` 中确认：
   - Threshold source 显示为 `env:PERF_NIGHTLY_THRESHOLDS`
   - Status 不是 `not_configured`
   - 各 scenario 的阈值判定符合预期

如果你想在本地先压缩 JSON，可直接运行：

```bash
node -e "process.stdout.write(JSON.stringify(JSON.parse(require('node:fs').readFileSync('tools/perf/nightly-thresholds.example.json','utf8'))))"
```

也可以先在本地验证导出逻辑是否能正确读取：

```bash
node tools/perf/export-k6-trend.mjs \
  --input-dir tools/perf/output/raw \
  --output-dir tools/perf/output/trend \
  --metadata tools/perf/output/raw/run-metadata.json \
  --thresholds tools/perf/nightly-thresholds.example.json
```

行为说明：

- 未配置 `PERF_NIGHTLY_THRESHOLDS` 时，workflow 会生成 `not_configured` 结果，不会误判失败
- 已配置阈值但 `PERF_NIGHTLY_ENFORCE=false` 时，阈值失败只发 `warning` annotation，不阻塞 nightly
- 已配置阈值且 `PERF_NIGHTLY_ENFORCE=true` 时，任一 scenario 触发阈值失败会让该步骤退出非零
- `perf-threshold-report.json` 可供后续脚本或外部报表继续消费

### longer-run 阈值守护

`.github/workflows/perf-long-run.yml` 与 nightly 使用同一套阈值模型，但变量名独立，默认面向更长时长与更高并发的压测结果。

相关仓库变量：

| 类型 | 名称 | 默认值 | 说明 |
| --- | --- | --- | --- |
| Variable | `PERF_LONGRUN_THRESHOLDS` | 空 | long-run 阈值 JSON 字符串 |
| Variable | `PERF_LONGRUN_ENFORCE` | `false` | 是否把阈值失败升级为阻塞失败 |

行为说明：

- 未配置 `PERF_LONGRUN_THRESHOLDS` 时，会生成 `not_configured` 结果，不阻塞 workflow
- 已配置阈值但 `PERF_LONGRUN_ENFORCE=false` 时，阈值失败只发 warning
- 已配置阈值且 `PERF_LONGRUN_ENFORCE=true` 时，阈值失败会让阈值步骤返回非零
- 手动触发时如果输入 `enforce_thresholds=true`，会同时：
  - 让阈值步骤按强制模式执行
  - 关闭 job 级默认非阻塞语义，使整个 workflow 对失败敏感

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
