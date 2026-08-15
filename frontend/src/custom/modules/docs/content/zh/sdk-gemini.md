<!-- beginner -->
## 步骤 1：确认 Gemini 兼容路径

本站 Gemini 兼容接口使用 `<site-root>/v1beta/models/<model-id>:generateContent`。先创建独立 Key，并从 `<api_base_url>/models` 复制当前 Key 真正可见的模型 ID。

## 步骤 2：先发送 REST 测试

```bash
curl "<site-root>/v1beta/models/<model-id>:generateContent" \
  -H "x-goog-api-key: sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"contents":[{"role":"user","parts":[{"text":"Reply only OK"}]}]}'
```

响应的 `candidates[0].content.parts` 中出现 `OK` 即为成功。REST 不通时不要先调 SDK 参数。

## 步骤 3：安装 Google Gen AI Python SDK

```text
python -m pip install google-genai
```

把 Key 放入环境变量：

```powershell
$env:SUB2API_API_KEY = "sk-demo-replace-me"
```

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

## 步骤 4：配置 SDK 的 Base URL

```python
import os
from google import genai
from google.genai import types

client = genai.Client(
    api_key=os.environ["SUB2API_API_KEY"],
    http_options=types.HttpOptions(
        base_url="<site-root>",
        api_version="v1beta",
    ),
)

response = client.models.generate_content(
    model="<model-id>",
    contents="Reply only OK",
)
print(response.text)
```

这里填 `<site-root>`，不要填已含 `/v1` 的 OpenAI `api_base_url`，也不要把 `:generateContent` 写进 Base URL。

## 步骤 5：再评估 Gemini CLI

Gemini CLI 默认面向 Google OAuth 或 `GEMINI_API_KEY`。自定义 Base URL 的变量和行为可能随 CLI 版本变化；只有当前安装版本的官方文档明确支持，并且 REST 与 SDK 测试都成功时，才把本站 Key 用于 CLI。记录 `gemini --version` 和最终请求 URL。

## 失败时先检查三项

1. 请求是否使用 `/v1beta` 和 `x-goog-api-key`，没有套用 OpenAI 路径。
2. Key 分组与 `/models` 是否包含精确模型 ID。
3. SDK 的 `base_url` 是否为 `<site-root>`，`api_version` 是否为 `v1beta`。

仍失败时提供 SDK/CLI 版本、模型、最终 URL、状态码、`X-Request-ID` 和脱敏错误体，不要提供完整 Key。

<!-- reference -->
## 流式端点

流式生成使用 `<site-root>/v1beta/models/<model-id>:streamGenerateContent?alt=sse`。响应结构与 OpenAI SSE 不同，不能按 `data: [DONE]` 的 Chat Completions 规则解析。

`HttpOptions` 属于 SDK 版本接口。升级 `google-genai` 后应重新运行最小请求；若 SDK 行为与 REST 不一致，先保留 REST 作为协议基线并记录 SDK 完整版本。
