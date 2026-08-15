<!-- beginner -->
## Step 1: Set an environment variable

PowerShell:

```powershell
$env:SUB2API_API_KEY = "sk-demo-replace-me"
```

macOS/Linux:

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

## Step 2: Use the Python SDK

```python
import os
from openai import OpenAI

client = OpenAI(api_key=os.environ["SUB2API_API_KEY"], base_url="<api_base_url>")
response = client.responses.create(model="<model-id>", input="Reply only OK")
print(response.output_text)
```

A returned `OK` proves the basic connection. Add streaming, tools, or conversation history afterward.

## Step 3: Use the Node.js SDK

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

Node.js uses `baseURL`, while Python uses `base_url`. Preserve the language-specific casing.

## Step 4: Configure Codex

Edit `~/.codex/config.toml`, or `%USERPROFILE%\.codex\config.toml` on Windows:

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

Run `codex exec --model <model-id> "Reply only OK"`. A group that supports Chat Completions only is not suitable for Codex's Responses connection.

## Success and troubleshooting

An SDK `output_text` plus Codex exit code `0` means success. Check that the environment variable reached the current process, `api_base_url` does not duplicate `/v1`, and the model remains visible to the key. For diagnosis provide client version, model, endpoint, status, and `X-Request-ID`, never the full key.

<!-- reference -->
## Protocol selection

Prefer `responses.create` for new applications and use `chat.completions.create` only for legacy compatibility. Codex currently uses Responses for `wire_api`. Read the key through `env_key`; never place it in TOML.
