<!-- beginner -->
## 步骤 1：创建专用 Key 并获取模型

在本站创建 API Key，确认已分配分组，然后获取该 Key 当前可见的模型：

```bash
curl "https://api.daigua.icu/v1/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

后续所有 `<model-id>` 都替换为 `data[].id` 中的精确值。不要使用教程中的固定模型名。

## 步骤 2：先用 curl 测试 Responses

![OpenAI SDK 与 curl 使用 DaiGua Base URL、Key、模型和 Responses 端点的中文参考图](asset:sdk/openai-curl.png "OpenAI SDK：先用最小 curl 请求确认 Responses 协议")

```bash
curl "https://api.daigua.icu/v1/responses" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","input":"Reply only OK"}'
```

返回 `200` 且输出文本包含 `OK`，说明 Key、分组、模型、Base URL 和 Responses 基本可用。

## 步骤 3：需要旧客户端时测试 Chat Completions

```bash
curl "https://api.daigua.icu/v1/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

Responses 成功不代表 Chat Completions 一定成功，反之亦然。分组只开放其中一种协议时，请选择匹配的 SDK 调用或客户端。

## 步骤 4：Python SDK 使用 Responses

```powershell
$env:SUB2API_API_KEY="sk-demo-replace-me"
python -m pip install --upgrade openai
```

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["SUB2API_API_KEY"],
    base_url="https://api.daigua.icu/v1",
)

response = client.responses.create(
    model="<model-id>",
    input="Reply only OK",
)
print(response.output_text)
```

Python 参数名是 `base_url`。

## 步骤 5：Python SDK 使用 Chat Completions

```python
response = client.chat.completions.create(
    model="<model-id>",
    messages=[{"role": "user", "content": "Reply only OK"}],
)
print(response.choices[0].message.content)
```

只有步骤 3 的最小请求成功后才使用这段代码。

## 步骤 6：Node.js SDK

```bash
npm install openai
```

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.SUB2API_API_KEY,
  baseURL: "https://api.daigua.icu/v1",
});

const response = await client.responses.create({
  model: "<model-id>",
  input: "Reply only OK",
});

console.log(response.output_text);
```

Node.js 参数名是 `baseURL`，不要写成 Python 的 `base_url`。

## 步骤 7：确认成功并逐项恢复功能

SDK 返回 `OK`，本站用量页出现相同时间和模型的记录，即完成最小接入。随后按流式、system 指令、工具、图片、多轮上下文的顺序逐项开启，每次只增加一项。

## 步骤 8：失败时提交诊断信息

先检查环境变量、最终 URL 和 `/models`。仍失败时提供 SDK 名称与版本、运行时版本、协议、模型 ID、时间与时区、状态码、`X-Request-ID` 和脱敏错误体。不要提交完整 Key 或业务输入。

<!-- reference -->
## Responses 与 Chat Completions

新项目优先使用 `responses.create`；兼容旧框架时再使用 `chat.completions.create`。两者请求结构、流式事件和工具字段不同，不能只替换 URL。

| SDK | Base URL 参数 | 值 |
| --- | --- | --- |
| Python | `base_url` | `https://api.daigua.icu/v1` |
| Node.js | `baseURL` | `https://api.daigua.icu/v1` |
| curl | 完整请求 URL | `/v1/responses` 或 `/v1/chat/completions` |

Codex 配置、WebSocket 适用条件和其他编程客户端请查看 [编程客户端与 CLI 接入](/docs/grok-cli)。
