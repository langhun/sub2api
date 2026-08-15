<!-- beginner -->
## 步骤 1：确认客户端名称和版本

酒馆类客户端和第三方改版很多，先在“关于”或设置页记录完整名称与版本。本文使用通用字段映射，不把某个改版界面当作唯一标准。

![不同客户端中的 Base URL、API Key、模型和流式字段](asset:tavernai/field-map.png "TavernAI 兼容客户端通用界面示意")

## 步骤 2：创建客户端专用 Key

在本站 **API 密钥** 新建一个独立 Key，名称包含客户端和设备，例如 `tavernai-desktop`。选择包含目标模型的分组，先设置小额额度；首次成功前暂不设置 IP 限制。

## 步骤 3：确认 Key 可见的模型

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

保存返回的精确模型 ID。模型不可见时先检查 Key 分组、模型白名单和启用状态，不要在客户端里猜模型名称。

## 步骤 4：复制本站 Base URL

从 Key 的“使用”弹窗复制 `api_base_url`。是否包含 `/v1` 以弹窗为准；不要从浏览器地址栏复制后台页面 URL。

## 步骤 5：打开 Provider Settings

在客户端中查找 `Provider Settings`、`API Settings`、`Connections` 或 `Backend`。先关闭角色卡自动回复，避免保存后立即产生复杂请求。

## 步骤 6：选择可自定义地址的 Provider

现代兼容客户端选择 `Custom OpenAI-compatible`。TavernAI v1 legacy 必须选择 `Reverse Proxy`；选择 `OpenAI` 会锁定官方地址，无法接入本站。

## 步骤 7：按名称映射字段

| 可能的名称 | 应选择或填写 | 常见错误 |
| --- | --- | --- |
| Provider / API / Backend / Source | Custom OpenAI-compatible 或 Reverse Proxy | 选择锁定官方地址的 OpenAI |
| Base URL / API address / Endpoint / Proxy URL | 本站 `api_base_url` | 填了后台页面地址或完整端点 |
| API Key / Token / Password / Proxy password | 客户端专用 Key | 粘贴了遮罩值或多余空格 |
| Model / Model ID / Custom model | `/models` 返回的精确 ID | 按显示名称猜测模型 |
| Streaming / Stream response | 初次保持默认，排障时先关闭 | 首次测试同时改多个开关 |
| Connect / Test | 连接并发送最小请求 | 未连接就导入长预设 |

TavernAI v1 的准确字段是 `API Proxy url adress`、`API key or Prpxy password` 和 `Custom proxy model`；拼写来自旧版界面，不是本文笔误。

## 步骤 8：检查最终请求路径

Base URL 不追加 `/chat/completions`。大多数兼容客户端会自动请求 `{base}/models` 和 `{base}/chat/completions`。出现 `404` 时从开发者工具或日志确认最终 URL，重点排查重复 `/v1` 和重复端点。

## 步骤 9：发送最小测试消息

先关闭工具、世界书和复杂预设，关闭流式后发送 `Reply only OK`。返回正常 assistant 内容且本站 **用量** 页面出现记录，即为成功。

## 步骤 10：逐项启用角色功能

依次恢复流式、角色卡、世界书、工具和长上下文，每次测试一项。如果 Grok 分组实际不支持 Chat Completions，应换支持 Responses 的客户端或选择具备 Chat 兼容能力的分组。

## 失败时先检查三项

1. Base URL 是否来自本站弹窗，且没有追加端点或重复 `/v1`。
2. Key 是否启用、已分组，`/models` 是否包含所填模型。
3. Provider 是否允许自定义地址，客户端协议是否确实为 Chat Completions。

仍失败时提供客户端完整名称和版本、时间和时区、最终请求 URL、模型、流式开关、HTTP 状态码、`X-Request-ID` 和脱敏错误体。不要提交 Key、Cookie、角色卡或聊天内容。

<!-- reference -->
## 版本字段对照

| 客户端 | Provider | 地址字段 | 模型字段 |
| --- | --- | --- | --- |
| SillyTavern 1.18 | Custom (OpenAI-compatible) | Custom Endpoint (Base URL) | Enter a Model ID |
| TavernAI v1 legacy | Reverse Proxy | API Proxy url adress | Model / Custom proxy model |
| TavernAI 2 或第三方改版 | 以版本界面为准 | Base URL / Endpoint / Proxy URL | Model / Model ID |

## 通用最小请求

```bash
curl "<api_base_url>/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}],"stream":false}'
```

按 401、403、404、429、5xx 顺序参考错误码文档。同名字段在第三方改版中可能采用不同的路径拼接规则，因此诊断时必须记录最终请求 URL。
