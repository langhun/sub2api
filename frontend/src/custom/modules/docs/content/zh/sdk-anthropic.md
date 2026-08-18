<!-- beginner -->
## 步骤 1：配置 Anthropic SDK

```python
import os
from anthropic import Anthropic

client = Anthropic(
    api_key=os.environ["ANTHROPIC_API_KEY"],
    base_url="https://api.daigua.icu",
)
message = client.messages.create(
    model="<model-id>", max_tokens=64,
    messages=[{"role": "user", "content": "Reply only OK"}],
)
print(message.content[0].text)
```

## 步骤 2：配置 Claude Code

先在本站创建专用 Key；本站当前公开的 Messages 网关根地址为 `https://api.daigua.icu`。PowerShell 使用：

```powershell
$env:ANTHROPIC_BASE_URL = "https://api.daigua.icu"
$env:ANTHROPIC_API_KEY = "sk-demo-replace-me"
$env:ANTHROPIC_MODEL = "<model-id>"
```

macOS/Linux 使用 `export ANTHROPIC_BASE_URL="https://api.daigua.icu"` 和 `export ANTHROPIC_API_KEY="sk-demo-replace-me"`。Claude Code 会调用 `/v1/messages`，所以 `ANTHROPIC_BASE_URL` 不包含 `/v1`。将来 Key 弹窗显示的地址若有变化，以弹窗为准。

## 步骤 3：验证模型回落

进入 Claude Code 后用 `/status` 检查 Base URL 和凭据来源，再发最小消息。使用非 Claude 模型时，同时设置默认 Opus、Sonnet、Haiku 与子代理模型，避免后台任务回落到不存在的模型 ID。

<!-- reference -->
## 凭据头

默认使用 `ANTHROPIC_API_KEY`，它通过 `x-api-key` 发送凭据。若管理员明确说明某个网关只接受 Bearer，才改用 `ANTHROPIC_AUTH_TOKEN`，并先移除 `ANTHROPIC_API_KEY`；两者不能同时设置。模型自动发现可能过滤不含 `claude` 或 `anthropic` 的 ID，Grok 等模型应显式设置 `ANTHROPIC_MODEL`。
