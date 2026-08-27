<!-- beginner -->
## Step 1: Create a dedicated key and list models

Create an API key, assign a group, and list models visible to that key:

```bash
curl "https://api.daigua.icu/v1/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Replace every `<model-id>` below with an exact `data[].id` value. Do not use a fixed model name from a tutorial.

## Step 2: Test Responses with curl first

![Chinese annotated reference showing the DaiGua Base URL, key, model, and Responses endpoint for OpenAI SDK and curl](asset:sdk/openai-curl.png "OpenAI SDK: verify Responses with a minimal curl request first")

```bash
curl "https://api.daigua.icu/v1/responses" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","input":"Reply only OK"}'
```

A `200` response containing `OK` confirms basic key, group, model, Base URL, and Responses compatibility.

## Step 3: Test Chat Completions for legacy clients

```bash
curl "https://api.daigua.icu/v1/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

Responses success does not guarantee Chat Completions success, or the reverse. Use the protocol enabled for the selected group.

## Step 4: Use Responses with the Python SDK

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
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

The Python parameter is `base_url`.

## Step 5: Use Chat Completions with the Python SDK

```python
response = client.chat.completions.create(
    model="<model-id>",
    messages=[{"role": "user", "content": "Reply only OK"}],
)
print(response.choices[0].message.content)
```

Use this only after the step 3 request succeeds.

## Step 6: Use the Node.js SDK

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

The Node.js parameter is `baseURL`, not Python's `base_url`.

## Step 7: Confirm success and restore features gradually

The minimal integration succeeds when the SDK returns `OK` and the DaiGua usage page shows a request with the same timestamp and model. Restore streaming, system instructions, tools, images, and conversation history one feature at a time.

## Step 8: Provide diagnostics for failures

First check the environment variable, final URL, and `/models`. If it still fails, provide the SDK and version, runtime version, protocol, model ID, timestamp and timezone, status code, `X-Request-ID`, and a redacted error body. Never provide a full key or business input.

<!-- reference -->
## Responses and Chat Completions

Prefer `responses.create` for new projects and use `chat.completions.create` for compatible legacy frameworks. Their request bodies, streaming events, and tool fields differ; changing only the URL is insufficient.

| SDK | Base URL option | Value |
| --- | --- | --- |
| Python | `base_url` | `https://api.daigua.icu/v1` |
| Node.js | `baseURL` | `https://api.daigua.icu/v1` |
| curl | Full request URL | `/v1/responses` or `/v1/chat/completions` |

See [Coding clients and CLI integration](/docs/grok-cli) for Codex, WebSocket conditions, and other coding clients.
