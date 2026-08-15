<!-- beginner -->
## 步骤 1：准备客户端和配置目录

先安装并确认至少一个客户端可以启动：Grok Build CLI、Codex、Claude Code 或 OpenCode。记录客户端版本，并确定配置文件位置：

| 客户端 | Windows | macOS / Linux |
| --- | --- | --- |
| Grok Build CLI | `%USERPROFILE%\.grok\config.toml` | `~/.grok/config.toml` |
| Codex | `%USERPROFILE%\.codex\config.toml` | `~/.codex/config.toml` |
| Claude Code | `%USERPROFILE%\.claude\settings.json` | `~/.claude/settings.json` |
| OpenCode | `~/.config/opencode/opencode.json` | `~/.config/opencode/opencode.json` |

不要把这些用户配置或真实 Key 放进项目仓库。下文只使用 `sk-demo-replace-me` 作为示例 Key。

## 步骤 2：创建客户端专用 Key

在本站 **API 密钥** 新建独立 Key，名称写明客户端，例如“Grok-Codex-台式机”。首次测试先使用小额额度，并暂时不启用 IP 白名单。不同设备或客户端使用不同 Key，便于撤销和定位问题。

## 步骤 3：确认 Grok 分组和模型

Key 必须绑定 Grok 分组。用该 Key 查询模型，不能使用管理员账号看到的模型或旧教程中的固定名称：

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

从 `data[].id` 选择一个文本模型。Codex 和 Grok CLI 需要该分组支持 Responses；Claude Code 需要本站的 Messages 桥接；OpenCode 根据所选 npm provider 使用 Responses 或 Chat。

## 步骤 4：复制 api_base_url 并区分 site-root

从 Key 的 **使用方式** 弹窗复制 `api_base_url`，不要凭域名猜测。假设它是 `https://gateway.example/v1`：

- Grok CLI、Codex、OpenCode 使用完整 `<api_base_url>`；
- Claude Code 的 `ANTHROPIC_BASE_URL` 使用 `<site-root>`，即去掉最后一个 `/v1` 后的 `https://gateway.example`；
- 不要把 `/responses`、`/messages` 或 `/chat/completions` 拼进 Base URL 字段。

## 步骤 5：按客户端填写字段

![Grok CLI Responses 后端配置](asset:grok/grok-cli-config.png "Grok CLI 配置文件通用示意图")

字段对照：

| 客户端 | Base URL 字段 | Key 字段 | 协议字段 |
| --- | --- | --- | --- |
| Grok CLI | `base_url = "<api_base_url>"` | `env_key = "XAI_API_KEY"` | `api_backend = "responses"` |
| Codex | `base_url = "<api_base_url>"` | `env_key = "SUB2API_API_KEY"` | `wire_api = "responses"` |
| Claude Code | `ANTHROPIC_BASE_URL=<site-root>` | `ANTHROPIC_AUTH_TOKEN=sk-demo-replace-me` | 固定使用 Messages |
| OpenCode | `options.baseURL` | `options.apiKey = "{env:SUB2API_API_KEY}"` | npm provider 决定协议 |

### Grok Build CLI

在用户级 `config.toml` 中添加：

```toml
[model."site-grok"]
model = "<grok-model-id>"
base_url = "<api_base_url>"
name = "Site Grok"
env_key = "XAI_API_KEY"
api_backend = "responses"

[models]
default = "site-grok"
```

PowerShell：

```powershell
$env:XAI_API_KEY="sk-demo-replace-me"
```

macOS / Linux：

```bash
export XAI_API_KEY="sk-demo-replace-me"
```

必须显式保留 `api_backend = "responses"`。本文采用当前官方公开的每模型 `base_url`，不依赖旧配置中的私有或版本化 `[endpoints]` 字段。

### Codex

![Codex 自定义 Responses Provider 配置字段](asset:grok/codex-provider.png "Codex 自定义 Provider 通用配置示意图")

在用户级 `config.toml` 中添加：

```toml
model_provider = "site-grok"
model = "<grok-model-id>"

[model_providers.site-grok]
name = "Site Grok"
base_url = "<api_base_url>"
env_key = "SUB2API_API_KEY"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

设置 `SUB2API_API_KEY`：

```powershell
$env:SUB2API_API_KEY="sk-demo-replace-me"
```

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

Codex 自定义 provider 只支持 Responses。不要同时设置 `env_key` 和磁盘明文 `experimental_bearer_token`。只有管理员明确确认本站为该路径开放 Responses WebSocket 时，才修改 `supports_websockets`。

### Claude Code

![Claude Code 网关环境变量与最小测试命令](asset:grok/claude-code-env.png "Claude Code Messages 网关通用配置示意图")

Claude Code 会在 `<site-root>` 后请求 `/v1/messages`，因此这里不能填带 `/v1` 的 `api_base_url`：

```powershell
$env:ANTHROPIC_BASE_URL="<site-root>"
$env:ANTHROPIC_AUTH_TOKEN="sk-demo-replace-me"
$env:ANTHROPIC_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_OPUS_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_SONNET_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_HAIKU_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_FABLE_MODEL="<grok-model-id>"
$env:CLAUDE_CODE_SUBAGENT_MODEL="<grok-model-id>"
```

```bash
export ANTHROPIC_BASE_URL="<site-root>"
export ANTHROPIC_AUTH_TOKEN="sk-demo-replace-me"
export ANTHROPIC_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_OPUS_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_SONNET_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_FABLE_MODEL="<grok-model-id>"
export CLAUDE_CODE_SUBAGENT_MODEL="<grok-model-id>"
```

同时映射各模型族和子代理，避免 Claude Code 的后台任务回落到分组中不存在的 Claude 模型。需要阻止非必要外联时可设置 `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1`，但它也会关闭网关模型发现，因此仍要保留显式模型映射。

### OpenCode

![OpenCode Responses Provider、环境变量和模型字段](asset:grok/opencode-provider.png "OpenCode Responses Provider 通用配置示意图")

Responses-only 或首选 Responses 的 Grok 分组使用 `@ai-sdk/openai`：

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "site/<grok-model-id>",
  "provider": {
    "site": {
      "npm": "@ai-sdk/openai",
      "name": "Site Grok Responses",
      "options": {
        "baseURL": "<api_base_url>",
        "apiKey": "{env:SUB2API_API_KEY}"
      },
      "models": {
        "<grok-model-id>": { "name": "Grok" }
      }
    }
  }
}
```

只有已经用最小 Chat 请求验证成功的分组，才把 npm provider 改为 `@ai-sdk/openai-compatible`。它使用 `/v1/chat/completions`，不能替代 Responses-only 接入。

## 步骤 6：在客户端选择模型

- Grok CLI：运行 `grok models`，选择 `site-grok`，或启动时传 `-m site-grok`。
- Codex：使用配置中的 `<grok-model-id>`，或启动时显式传 `--model <grok-model-id>`。
- Claude Code：运行 `/model`；如果列表未展示自定义名称，直接使用 `--model <grok-model-id>` 或前面的环境变量。
- OpenCode：运行 `/models`，选择 `site/<grok-model-id>`。

显示名称可以自定义，但发给本站的模型 ID 必须与步骤 3 的 `data[].id` 完全一致。

## 步骤 7：发送最小测试消息

在设置环境变量的同一个终端中运行对应命令：

```text
grok inspect
grok -p "Reply only OK" -m site-grok
```

```text
codex exec --model <grok-model-id> "Reply only OK"
```

```text
claude --model <grok-model-id> -p "Reply only OK"
```

```text
opencode debug config
opencode run --model "site/<grok-model-id>" "Reply only OK"
```

第一次测试不要启用 MCP、图片、工具插件、长上下文或自动多代理。先证明基本协议和鉴权可用，再逐项恢复功能。

## 步骤 8：确认什么结果算成功

- Grok CLI：`grok inspect` 能看到用户配置和 `site-grok`，测试返回 `OK`。
- Codex：命令退出码为 `0`，没有要求 ChatGPT OAuth 登录，也没有 WebSocket 连接失败后循环重试。
- Claude Code：交互模式 `/status` 显示本站 `Anthropic base URL` 和 `ANTHROPIC_AUTH_TOKEN`，消息返回 `OK`。
- OpenCode：`opencode debug config` 显示 `site` provider、正确 npm 包和 Base URL，测试返回 `OK`。

四个客户端是独立验收项；一个成功不代表其他协议也成功。

## 步骤 9：失败时先检查三项

1. **环境变量是否进入当前进程**：重新打开终端后变量可能消失；配置文件中的 `env_key` 是变量名称，不是 Key 本身。
2. **协议与 Base URL 是否匹配**：Codex 必须 Responses；Claude Code 必须 Messages 且使用 `<site-root>`；OpenCode npm 包必须与 Responses/Chat 对应。
3. **模型和分组是否仍可用**：重新请求 `/models`，检查 Key 分组、模型白名单、额度，以及 `429` 的 `Retry-After`。

## 步骤 10：仍失败时提交诊断信息

提供操作系统、客户端名称与完整版本、配置文件路径、使用的 provider/协议、脱敏 Base URL、模型 ID、发生时间和时区、状态码、`X-Request-ID`、终端错误及脱敏响应体。Grok CLI 同时提供 `grok inspect` 的脱敏输出；Claude Code 提供 `/status` 中的地址和凭据来源；OpenCode 提供 `opencode debug config` 的脱敏输出。不要提交完整 Key、`auth.json`、Cookie 或 OAuth Token。

<!-- reference -->
## 接口参考：客户端协议矩阵

| 客户端 | 本站端点 | 关键限制 |
| --- | --- | --- |
| Grok Build CLI | `/v1/responses` | 每个文本模型设置 `api_backend = "responses"` |
| Codex | `/v1/responses` | `wire_api` 只支持 `responses` |
| Claude Code | `/v1/messages` | `ANTHROPIC_BASE_URL` 填站点根地址 |
| OpenCode + `@ai-sdk/openai` | `/v1/responses` | 推荐用于 Responses-only 分组 |
| OpenCode + `@ai-sdk/openai-compatible` | `/v1/chat/completions` | 仅用于已验证的 Chat 兼容分组 |

Grok CLI 当前官方列出 `XAI_API_KEY`、`GROK_MODELS_BASE_URL`、`GROK_MODELS_LIST_URL` 和 `GROK_XAI_API_BASE_URL`。用户配置支持每模型 `model`、`base_url`、`name`、`env_key`、`api_backend` 等字段。本文使用每模型 `base_url`，避免依赖版本可能变化的旧 `[endpoints]` 配置。

## 管理员：开放 CLI 客户端前检查

1. 为测试 Key 分别验证 `/v1/models`、`/v1/responses` 和 `/v1/messages`；需要 Chat 时再单独验证 `/v1/chat/completions`。
2. 确认模型映射覆盖主模型、Claude Code 的后台/子代理模型，并且 Responses 工具字段不会被上游错误过滤。
3. Claude Code 网关会发送 Anthropic 版本和 beta 头；转换层应成对处理头与请求字段，并原样返回可诊断的错误。不能只改模型名就假设协议兼容。
4. Codex 默认按 HTTP/SSE 验收。只有部署已验证 Responses WebSocket 时才允许用户开启对应选项。
5. 发布配置前使用没有管理员权限的普通 Key 验收，并确认日志、截图和示例中只出现 `sk-demo-...`，没有真实凭据。
