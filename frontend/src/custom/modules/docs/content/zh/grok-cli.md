<!-- beginner -->
## 步骤 1：准备客户端和独立配置

本文覆盖 Grok Build CLI、Codex、Claude Code、OpenCode、OpenClaw、Hermes Agent、WorkBuddy 和 ZCode。先确认客户端能正常启动并记录版本；首次接入不要启用 MCP、长上下文、图片、工具或多代理。

| 客户端 | 配置入口或常用路径 | 首次推荐协议 |
| --- | --- | --- |
| Grok Build CLI | `~/.grok/config.toml` | Responses |
| Codex | Windows `%USERPROFILE%\.codex\config.toml`；macOS/Linux `~/.codex/config.toml` | Responses |
| Claude Code | 环境变量或 `~/.claude/settings.json` | Messages |
| OpenCode | `~/.config/opencode/opencode.json` | Chat Completions |
| OpenClaw | Control UI → Settings → Model Providers | Chat Completions |
| Hermes Agent | 终端运行 `hermes model` | OpenAI-compatible |
| WorkBuddy | 设置 → 模型 → 自定义 API | Chat Completions |
| ZCode | 设置 → 模型设置 → 添加供应商 | OpenAI-compatible |

配置文件和真实 Key 不要提交到项目仓库。本文只使用 `sk-demo-replace-me`。

## 步骤 2：为客户端创建专用 API Key

打开本站 **API 密钥**，按设备或客户端创建独立 Key，例如 `codex-laptop`。首次测试设置较小额度并暂不配置 IP 白名单，成功后再收紧限制。独立 Key 便于撤销、核对用量和定位问题。

## 步骤 3：确认分组、协议和模型

使用刚创建的 Key 获取模型列表：

```bash
curl "https://api.daigua.icu/v1/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

从 `data[].id` 复制一个模型 ID，不要照抄旧教程中的固定模型名。Codex 和 Grok CLI 需要分组支持 Responses；Claude Code 需要 Messages；OpenCode、OpenClaw、WorkBuddy 和本文的 ZCode OpenAI 配置需要 Chat Completions。

## 步骤 4：复制 Base URL，并确认客户端会追加什么

优先从 Key 的 **使用方式** 弹窗复制 `api_base_url`。本站当前公开示例为 `https://api.daigua.icu/v1`。

| 客户端 | 应填写的地址 | 客户端追加路径 |
| --- | --- | --- |
| Grok CLI、Codex、OpenCode、OpenClaw、Hermes | `https://api.daigua.icu/v1` | `/responses` 或 `/chat/completions` |
| Claude Code | `https://api.daigua.icu` | `/v1/messages` |
| WorkBuddy | `https://api.daigua.icu/v1` | 默认追加 `/chat/completions` |
| ZCode OpenAI 协议 | `https://api.daigua.icu/v1` | OpenAI Chat 路径 |

Base URL 字段不要再追加 `/chat/completions`、`/responses` 或 `/messages`。只有 WorkBuddy 开启“自定义协议”时才填写完整端点；本文默认关闭该开关。

## 步骤 5：按客户端填写配置

### Grok Build CLI

![Grok Build CLI 使用 DaiGua Responses 地址、环境变量和动态模型 ID 的中文参考图](asset:grok/grok-cli-config.png "Grok Build CLI：保留 Responses 后端并从环境变量读取 Key")

```toml
[model."daigua-grok"]
model = "<model-id>"
base_url = "https://api.daigua.icu/v1"
name = "DaiGua Grok"
env_key = "XAI_API_KEY"
api_backend = "responses"

[models]
default = "daigua-grok"
```

### Codex

![Codex 0.147.0 自定义 Responses Provider 的中文核验图](asset:grok/codex-provider.png "Codex：Responses Provider、DaiGua 地址和 WebSocket 适用条件")

```toml
model_provider = "daigua"
model = "<model-id>"

[model_providers.daigua]
name = "DaiGua"
base_url = "https://api.daigua.icu/v1"
env_key = "SUB2API_API_KEY"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

`wire_api` 保持 `responses`。Grok 或其他兼容路由默认使用 HTTP/SSE，因此 `supports_websockets = false`；只有已验证的 OpenAI 原生 Responses 分组才改成 `true`。这与 `mode_router_v2_enabled` 无关，后者不是 WebSocket 总开关。

### Claude Code

![Claude Code 使用站点根地址、API Key 和模型映射的中文参考图](asset:grok/claude-code-env.png "Claude Code：根地址不带 /v1，并映射后台模型")

```powershell
$env:ANTHROPIC_BASE_URL="https://api.daigua.icu"
$env:ANTHROPIC_API_KEY="sk-demo-replace-me"
$env:ANTHROPIC_MODEL="<model-id>"
$env:ANTHROPIC_DEFAULT_OPUS_MODEL="<model-id>"
$env:ANTHROPIC_DEFAULT_SONNET_MODEL="<model-id>"
$env:ANTHROPIC_DEFAULT_HAIKU_MODEL="<model-id>"
$env:CLAUDE_CODE_SUBAGENT_MODEL="<model-id>"
```

### OpenCode

![OpenCode Chat Completions Provider 的中文参考图](asset:grok/opencode-provider.png "OpenCode：使用 openai-compatible Provider 和环境变量 Key")

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "daigua/<model-id>",
  "provider": {
    "daigua": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "DaiGua",
      "options": {
        "baseURL": "https://api.daigua.icu/v1",
        "apiKey": "{env:SUB2API_API_KEY}"
      },
      "models": {
        "<model-id>": { "name": "DaiGua Model" }
      }
    }
  }
}
```

### OpenClaw

![OpenClaw 自定义 Provider 的中文字段参考图](asset:clients/openclaw-provider.png "OpenClaw：显式登记 Provider、模型和 Chat Completions 协议")

在 Control UI 添加自定义 Provider，或在配置中合并：

```json5
{
  agents: { defaults: { model: { primary: "daigua/<model-id>" } } },
  models: {
    mode: "merge",
    providers: {
      daigua: {
        baseUrl: "https://api.daigua.icu/v1",
        apiKey: "${DAIGUA_API_KEY}",
        api: "openai-completions",
        models: [{ id: "<model-id>", name: "DaiGua Model" }]
      }
    }
  }
}
```

不要只设置默认模型；自定义模型还必须出现在 `models.providers.daigua.models[]`。

### Hermes Agent

![Hermes Agent Custom Provider 向导的中文字段参考图](asset:clients/hermes-custom-provider.png "Hermes Agent：通过 hermes model 填写自定义地址、Key 和模型")

运行 `hermes model`，选择 **Custom / OpenAI-compatible**，依次填写：

| 字段 | 值 |
| --- | --- |
| Provider name | `daigua` |
| Base URL | `https://api.daigua.icu/v1` |
| API Key | `sk-demo-replace-me` |
| Model | `<model-id>` |

自定义 Provider 没有固定环境变量名，以当前向导保存的位置为准。

### WorkBuddy

![WorkBuddy 设置页自定义 API 的中文字段参考图](asset:clients/workbuddy-custom-api.png "WorkBuddy：关闭自定义协议并让客户端追加 Chat Completions 路径")

进入 **设置 → 模型 → 自定义 API**：

| 字段 | 值 |
| --- | --- |
| URL | `https://api.daigua.icu/v1` |
| API Key | `sk-demo-replace-me` |
| 模型名 | `<model-id>` |
| 自定义协议 | 关闭 |

默认关闭自定义协议时，WorkBuddy 会校验并追加 `/chat/completions`。只有明确填写完整端点时才打开该开关。图片、工具和推理能力不要仅凭模型名称勾选，先分别验证。

### ZCode

![ZCode 自定义 OpenAI 供应商的中文字段参考图](asset:clients/zcode-custom-provider.png "ZCode：添加 OpenAI-compatible 供应商并手动登记模型")

进入 **设置 → 模型设置 → 添加供应商**，选择 OpenAI-compatible：

| 字段 | 值 |
| --- | --- |
| 名称 | `DaiGua` |
| API 基础 URL | `https://api.daigua.icu/v1` |
| API Key | `sk-demo-replace-me` |
| 模型 ID | `<model-id>` |

仅当目标分组已经通过 `/v1/messages` 测试时才改用 Anthropic 协议，并将地址填为 `https://api.daigua.icu`。

## 步骤 6：设置 Key 环境变量并选择模型

PowerShell：

```powershell
$env:SUB2API_API_KEY="sk-demo-replace-me"
$env:XAI_API_KEY="sk-demo-replace-me"
$env:DAIGUA_API_KEY="sk-demo-replace-me"
```

macOS/Linux：

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
export XAI_API_KEY="sk-demo-replace-me"
export DAIGUA_API_KEY="sk-demo-replace-me"
```

模型显示名称可以自定义，但发给本站的 `<model-id>` 必须与当前 Key 的 `/models` 返回值完全一致。

## 步骤 7：发送最小测试消息

先用 curl 验证客户端所需协议，再在客户端发送 `Reply only OK`：

```bash
curl "https://api.daigua.icu/v1/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

- Grok CLI：`grok inspect` 后发送最小提示。
- Codex：`codex exec --model <model-id> "Reply only OK"`。
- Claude Code：`claude --model <model-id> -p "Reply only OK"`。
- OpenCode：先运行 `opencode debug config`，再发送最小提示。
- OpenClaw：先点 **Test connection**，再从 Control UI 发送消息。
- Hermes：启动 `hermes`，在新会话发送消息。
- WorkBuddy、ZCode：从新建空白对话选择刚添加的模型后发送消息。

## 步骤 8：确认成功结果

成功必须同时满足：客户端显示的是 DaiGua Provider、模型 ID 正确、测试返回 `OK`、本站用量页出现对应请求且没有协议转换错误。能列出模型不代表聊天端点可用；一个客户端成功也不代表另一个协议成功。

## 步骤 9：失败时先检查三项

1. **最终 URL**：排除重复 `/v1`、重复 `/chat/completions`，以及 Claude Code 错把 `/v1` 填进根地址。
2. **协议和能力**：Responses、Messages、Chat Completions 分别测试；工具、图片和推理参数随后单独恢复。
3. **Key、分组和模型**：确认环境变量进入当前进程，Key 已启用，模型仍在该 Key 的 `/models` 中，并检查额度与 `Retry-After`。

## 步骤 10：仍失败时提交诊断信息

提供操作系统、客户端名称和版本、配置入口、Provider、脱敏 Base URL、模型 ID、协议、发生时间与时区、最终请求 URL、状态码、`X-Request-ID`、脱敏错误体和最小复现步骤。不要提交完整 Key、配置凭据文件、Cookie、OAuth Token、工作区源码或私人对话。

<!-- reference -->
## 接口参考：客户端协议矩阵

| 客户端 | 本站端点 | 关键限制 |
| --- | --- | --- |
| Grok Build CLI | `/v1/responses` | `api_backend = "responses"` |
| Codex | `/v1/responses` | `wire_api = "responses"`；WebSocket 按路由能力设置 |
| Claude Code | `/v1/messages` | Base URL 使用站点根地址 |
| OpenCode | `/v1/chat/completions` | `@ai-sdk/openai-compatible` 不适用于 Responses-only 分组 |
| OpenClaw | `/v1/chat/completions` | 自定义模型必须登记到 Provider 模型列表 |
| Hermes Agent | `/v1/chat/completions` | 使用 Custom OpenAI-compatible Provider |
| WorkBuddy | `/v1/chat/completions` | 默认由客户端自动追加端点 |
| ZCode | `/v1/chat/completions` 或 `/v1/messages` | 选择与分组一致的协议 |

## WebSocket 与高级能力

本站提供 `GET /v1/responses` WebSocket 入口，但客户端配置仍需匹配目标分组和上游能力。`supports_websockets = true` 不是通用加速开关；错误开启可能导致客户端循环重连。`mode_router_v2_enabled = false` 只表示新账户模式路由器未启用，不表示 WebSocket 被关闭。

上下文窗口、图片、工具调用和推理档位不能从模型名称推断。先用纯文本最小请求验证，再逐项开启；第三方客户端生成的高级参数若上游不接受，可能返回 `400`。
