<!-- beginner -->
## 步骤 1：准备接入信息

开始前准备一个已登录的本站账号、可创建 API Key 的权限，以及能运行 `curl` 的终端。本文使用 `sk-demo-replace-me` 作为示例 Key，不要把真实 Key 放进截图、聊天记录或工单。

## 步骤 2：创建 Grok 专用 Key

打开 **API 密钥**，新建一个独立 Key。名称建议包含用途和客户端，例如“Grok-测试机”。第一次接入时先设置较小额度；IP 白名单等限制应在最小请求成功后再开启，以便区分配置错误和访问限制。

![Grok Base URL、API Key 和模型字段](asset:grok/grok-api-fields.png "Grok API 通用配置界面")

## 步骤 3：确认分组和可用模型

创建时选择 Grok 分组。保存后确认 Key 详情中已经显示分组，且管理员已给该分组配置可用账号、模型白名单和所需媒体权限。

先读取这个 Key 真正可见的模型：

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

返回 JSON 中的 `data[].id` 才是可以填写的模型 ID。不要根据旧教程猜测 `grok-4.5`、`grok-4.6` 或媒体模型名称。

## 步骤 4：复制本站 api_base_url

从 Key 的 **使用方式** 弹窗复制 `api_base_url`。它通常已经以 `/v1` 结尾，后面的示例会在它后面追加 `/models` 或 `/responses`。

为了让复制后的配置一眼可核验，下面用本站公开示例地址 `https://api.daigua.icu/v1`：

- Base URL 填 `https://api.daigua.icu/v1`；
- Responses 请求地址是 `https://api.daigua.icu/v1/responses`；
- 不要再手动加一次 `/v1`；
- 不要把 Base URL 填成 `/chat/completions` 或 `/responses` 的完整请求地址。

实际使用时始终以 Key 的“使用方式”弹窗显示的 `api_base_url` 为准；如果管理员配置了其他入口，只替换这两个示例中的主机名。

## 步骤 5：填写调用字段

| 字段 | 这里填什么 | 常见错误 |
| --- | --- | --- |
| Base URL | 本站显示的 `<api_base_url>` | 重复添加 `/v1` |
| API Key | `sk-demo-replace-me` 对应位置填真实 Key | 使用后台登录密码或上游 xAI Key |
| 协议 | 首选 `Responses` | 客户端只支持 Chat 却选择 Responses-only 分组 |
| 模型 | 步骤 3 返回的 `data[].id` | 复制展示名称而不是模型 ID |
| 存储 | 敏感场景设置 `store: false` | 误以为它会关闭本站必要日志和审计 |

鉴权头使用 `Authorization: Bearer <API Key>`。查询参数中的 Key 已弃用，不要把 Key 拼进 URL。

## 步骤 6：选择模型

从步骤 3 的 `/models` 结果中复制一个文本模型 ID。图片、视频和 Voice 使用不同的模型和端点，不能把媒体模型填进下面的文本请求。Grok 分组不提供 Embeddings，需要向量能力时请选择支持 Embeddings 的 OpenAI 分组。

## 步骤 7：发送最小测试消息

把 `<api_base_url>` 和 `<grok-model-id>` 替换为前面确认的值：

```bash
curl "<api_base_url>/responses" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<grok-model-id>","input":"Reply only OK","store":false}'
```

第一次测试只发送纯文本，不要同时开启图片、工具调用、长上下文或流式输出。最小请求成功后再逐项增加能力。

## 步骤 8：确认什么结果算成功

同时满足以下条件才算接入成功：

- HTTP 状态码为 `200`；
- 响应包含 `id` 和 `output`；
- `output` 的助手文本中能看到 `OK`；
- 响应模型与请求模型一致或符合管理员配置的模型映射；
- 没有出现登录页 HTML、反向代理错误页或重复 `/v1/v1` 路径。

## 步骤 9：失败时先检查三项

1. **Key 与分组**：Key 未过期、额度未耗尽，并且绑定的是 Grok 分组。
2. **Base URL 与协议**：`api_base_url` 没有重复 `/v1`，客户端实际发送的是 Responses、Chat 还是 Messages。
3. **模型与限流**：目标模型仍在 `/models` 中；遇到 `429` 时读取 `Retry-After`，不要连续快速重试。

## 步骤 10：仍失败时提交诊断信息

向支持人员提供：发生时间和时区、客户端名称与版本、请求端点、模型 ID、HTTP 状态码、`X-Request-ID`、是否流式、Key 所属分组，以及脱敏后的错误体。Key 只保留前后各 3 到 4 位，禁止提交完整 Key、Cookie、OAuth Token 或密码。

<!-- reference -->
## 接口参考：Responses 与 Chat

xAI 当前优先推荐 Responses。它支持 `previous_response_id`、原生工具和有类型的 `output`；Chat Completions 是兼容接口，需要客户端自行重发历史消息。

仅当客户端必须使用 Chat 时，发送最小兼容请求：

```bash
curl "<api_base_url>/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<grok-model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

成功内容位于 `choices[0].message.content`。Chat 是否可用取决于 Grok 分组和上游账号能力；Responses 成功不代表 Chat 必然成功，反之亦然。Responses 默认可能由 xAI 保存 30 天，`store: false` 可请求上游不保存该响应，但不改变本站依法或为安全、计费所需的日志策略。

## 接口参考：文本与媒体端点

| 能力 | 端点 | 说明 |
| --- | --- | --- |
| 模型列表 | `GET /v1/models` | 以当前 Key 可见结果为准 |
| 文本 | `POST /v1/responses` | 首选文本协议 |
| Chat 兼容 | `POST /v1/chat/completions` | 取决于分组和转换能力 |
| Messages | `POST /v1/messages` | Claude Code 等客户端使用 |
| 图片 | `POST /v1/images/generations`、`POST /v1/images/edits` | 需要媒体模型和图片权限 |
| 视频 | `POST /v1/videos/generations` 后轮询 `GET /v1/videos/{request_id}` | 异步任务，完成前不要当作失败 |
| 语音 | `POST /v1/tts`、`POST /v1/stt`、`/v1/custom-voices` | 返回音频或结构数据 |
| 实时语音 | `/v1/realtime` WebSocket | 与 HTTP 文本配置不同 |

本站允许透传 `x-ratelimit-limit-*`、`x-ratelimit-remaining-*`、`x-ratelimit-reset-*` 和 `Retry-After`。遇到 `429` 时按响应头等待，并采用有上限的指数退避。

## 管理员：配置 Grok 上游

1. 在管理员账号页选择 Grok OAuth 或 API Key 类型；优先使用 OAuth 授权或独立 API Key，不要收集用户密码、原始 SSO Cookie 或可复用登录凭据。
2. API Key 账号默认上游为 `https://api.x.ai/v1`。OAuth 的自定义 Base URL 只改写对话、媒体和探测流量，OAuth 授权与令牌刷新仍走官方端点。
3. 只有私有网关或区域代理明确要求时才设置自定义 Base URL，并验证 TLS、路径、鉴权头和各端点返回。认证头由系统管理，不应通过 Header Override 覆写。
4. 在分组与模型映射中把用户可见 ID 映射到实际上游模型。文本、图片、视频和 Voice 分别验证；文本映射不会自动改变媒体模型或媒体权限。
5. 检查 `x-ratelimit-*`、`Retry-After` 和本站限流日志。更新 OAuth/API Key 后，使用独立测试 Key 验证 `/models`、Responses、Chat 兼容和每项已开放的媒体能力，再提供给用户。

自定义 Base URL、模型映射和账号能力属于管理员配置。用户示例不会覆盖后台真实设置。
