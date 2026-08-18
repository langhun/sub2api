<!-- beginner -->
## Step 1: Configure the Anthropic SDK

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

## Step 2: Configure Claude Code

Create a dedicated key on this site first. The current public Messages gateway root is `https://api.daigua.icu`. In PowerShell:

```powershell
$env:ANTHROPIC_BASE_URL = "https://api.daigua.icu"
$env:ANTHROPIC_API_KEY = "sk-demo-replace-me"
$env:ANTHROPIC_MODEL = "<model-id>"
```

On macOS/Linux use `export ANTHROPIC_BASE_URL="https://api.daigua.icu"` and `export ANTHROPIC_API_KEY="sk-demo-replace-me"`. Claude Code calls `/v1/messages`, so `ANTHROPIC_BASE_URL` excludes `/v1`. If the key dialog later shows a different address, use that value.

## Step 3: Check model fallback

Run `/status` in Claude Code to inspect the Base URL and credential source, then send a minimal message. For non-Claude models, also map default Opus, Sonnet, Haiku, and subagent models so background work does not fall back to missing IDs.

<!-- reference -->
## Credential headers

Use `ANTHROPIC_API_KEY` by default; it sends the credential through `x-api-key`. Switch to `ANTHROPIC_AUTH_TOKEN` only when an administrator explicitly says that a gateway accepts Bearer only, and remove `ANTHROPIC_API_KEY` first. Never set both. Discovery can filter IDs that do not contain `claude` or `anthropic`, so explicitly set `ANTHROPIC_MODEL` for Grok and similar IDs.
