<!-- beginner -->
## 步骤 1：配置 Anthropic SDK

```python
import os
from anthropic import Anthropic

client = Anthropic(
    api_key=os.environ["SUB2API_API_KEY"],
    base_url="<site-root>",
)
message = client.messages.create(
    model="<model-id>", max_tokens=64,
    messages=[{"role": "user", "content": "Reply only OK"}],
)
print(message.content[0].text)
```

## 步骤 2：配置 Claude Code

Bearer 网关使用：

```powershell
$env:ANTHROPIC_BASE_URL = "<site-root>"
$env:ANTHROPIC_AUTH_TOKEN = "sk-demo-replace-me"
$env:ANTHROPIC_MODEL = "<model-id>"
```

macOS/Linux 将 `$env:NAME =` 改为 `export NAME=`。Claude Code 会调用 `/v1/messages`，所以 `ANTHROPIC_BASE_URL` 一般不包含 `/v1`。

## 步骤 3：验证模型回落

进入 Claude Code 后用 `/status` 检查 Base URL 和凭据来源，再发最小消息。使用非 Claude 模型时，同时设置默认 Opus、Sonnet、Haiku 与子代理模型，避免后台任务回落到不存在的模型 ID。

<!-- reference -->
## 凭据头

`ANTHROPIC_AUTH_TOKEN` 发送 Bearer；`ANTHROPIC_API_KEY` 发送 `x-api-key`。按本站兼容路径选择一种，不要同时设置不同值。模型自动发现可能过滤不含 `claude` 或 `anthropic` 的 ID，Grok 等模型应显式设置 `ANTHROPIC_MODEL`。
