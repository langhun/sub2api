<!-- beginner -->
## Step 1: Configure the Anthropic SDK

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

## Step 2: Configure Claude Code

For a Bearer gateway in PowerShell:

```powershell
$env:ANTHROPIC_BASE_URL = "<site-root>"
$env:ANTHROPIC_AUTH_TOKEN = "sk-demo-replace-me"
$env:ANTHROPIC_MODEL = "<model-id>"
```

On macOS/Linux replace `$env:NAME =` with `export NAME=`. Claude Code calls `/v1/messages`, so `ANTHROPIC_BASE_URL` normally excludes `/v1`.

## Step 3: Check model fallback

Run `/status` in Claude Code to inspect the Base URL and credential source, then send a minimal message. For non-Claude models, also map default Opus, Sonnet, Haiku, and subagent models so background work does not fall back to missing IDs.

<!-- reference -->
## Credential headers

`ANTHROPIC_AUTH_TOKEN` sends Bearer while `ANTHROPIC_API_KEY` sends `x-api-key`. Choose the path supported by this gateway and do not set conflicting values. Discovery can filter IDs that do not contain `claude` or `anthropic`, so explicitly set `ANTHROPIC_MODEL` for Grok and similar IDs.
