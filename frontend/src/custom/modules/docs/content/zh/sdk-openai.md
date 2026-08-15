<!-- beginner -->
## 步骤 1：设置环境变量

PowerShell：

```powershell
$env:SUB2API_API_KEY = "sk-demo-replace-me"
```

macOS/Linux：

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

## 步骤 2：使用 Python SDK

```python
import os
from openai import OpenAI

client = OpenAI(api_key=os.environ["SUB2API_API_KEY"], base_url="<api_base_url>")
response = client.responses.create(model="<model-id>", input="Reply only OK")
print(response.output_text)
```

返回 `OK` 后再增加流式、工具或多轮上下文。

## 步骤 3：使用 Node.js SDK

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.SUB2API_API_KEY,
  baseURL: "<api_base_url>",
});

const response = await client.responses.create({
  model: "<model-id>",
  input: "Reply only OK",
});

console.log(response.output_text);
```

Node.js 的配置名是 `baseURL`，Python 是 `base_url`；不要照抄成同一个大小写。

## 步骤 4：配置 Codex

在 `~/.codex/config.toml`；Windows 为 `%USERPROFILE%\.codex\config.toml`：

```toml
model_provider = "sub2api"
model = "<model-id>"

[model_providers.sub2api]
name = "Sub2API"
base_url = "<api_base_url>"
env_key = "SUB2API_API_KEY"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

运行 `codex exec --model <model-id> "Reply only OK"`。若分组只支持 Chat Completions，Codex 的 Responses 连接不适用。

## 成功与排障

SDK 返回 `output_text` 且 Codex 退出码为 `0` 即成功。失败先检查环境变量是否进入当前进程、`api_base_url` 是否重复 `/v1`、模型是否仍在当前 Key 的 `/models` 中。提交诊断时附客户端版本、模型、端点、状态码和 `X-Request-ID`，不要附完整 Key。

<!-- reference -->
## 协议选择

新项目优先使用 `responses.create`。需要兼容旧客户端时才使用 `chat.completions.create`。Codex 当前 `wire_api` 使用 Responses；Key 通过 `env_key` 读取，不要写进 TOML。
