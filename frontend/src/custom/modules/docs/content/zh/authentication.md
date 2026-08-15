<!-- beginner -->
## 步骤 1：复制正确的 Base URL

从 Key 的使用弹窗复制 `api_base_url`。OpenAI 客户端通常需要包含 `/v1` 的地址；Anthropic SDK 通常填写站点根地址并由 SDK 追加 `/v1/messages`；Gemini 使用 `/v1beta` 路径。

## 步骤 2：选择鉴权头

| 协议 | 请求头 |
| --- | --- |
| OpenAI / Responses | `Authorization: Bearer sk-demo-replace-me` |
| Anthropic | `x-api-key: sk-demo-replace-me`，部分客户端可用 Bearer |
| Gemini | `x-goog-api-key: sk-demo-replace-me` 或兼容查询参数 |

不要同时填写多个不同 Key。浏览器前端也不要直接保存长期 Key；应由自己的服务端代理请求。

## 步骤 3：验证

先调用模型列表，再发送最小文本请求。出现 `401` 时检查 Key 内容和请求头；出现 `403 ACCESS_DENIED` 时检查账户状态、余额和 IP 限制。

<!-- reference -->
## Base URL 规则

本站同时兼容多套协议，客户端对 Base URL 的追加规则并不相同。排障时记录“最终请求 URL”，而不只记录配置框中的值。响应头 `X-Request-ID` 可用于定位请求；不同协议的错误 JSON 结构不同，不要依赖统一的 `request_id` 字段。
