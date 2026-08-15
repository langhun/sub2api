<!-- beginner -->
## Step 1: Confirm the Gemini-compatible path

This site's Gemini-compatible endpoint is `<site-root>/v1beta/models/<model-id>:generateContent`. Create a dedicated key and copy a model ID actually visible to that key through `<api_base_url>/models`.

## Step 2: Send a REST test first

```bash
curl "<site-root>/v1beta/models/<model-id>:generateContent" \
  -H "x-goog-api-key: sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"contents":[{"role":"user","parts":[{"text":"Reply only OK"}]}]}'
```

`OK` under `candidates[0].content.parts` means success. Do not debug SDK options before the REST request works.

## Step 3: Install the Google Gen AI Python SDK

```text
python -m pip install google-genai
```

Put the key in an environment variable:

```powershell
$env:SUB2API_API_KEY = "sk-demo-replace-me"
```

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

## Step 4: Configure the SDK Base URL

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

Use `<site-root>`, not an OpenAI `api_base_url` that already contains `/v1`, and never put `:generateContent` in the Base URL.

## Step 5: Evaluate Gemini CLI afterward

Gemini CLI defaults to Google OAuth or `GEMINI_API_KEY`. Custom Base URL variables and behavior can change by CLI release. Use the site key only when the current official documentation explicitly supports the override and both REST and SDK tests already work. Record `gemini --version` and the final request URL.

## Check these three items first

1. The request uses `/v1beta` and `x-goog-api-key`, not an OpenAI path.
2. The key group and `/models` contain the exact model ID.
3. SDK `base_url` is `<site-root>` and `api_version` is `v1beta`.

For further diagnosis provide SDK/CLI version, model, final URL, status, `X-Request-ID`, and a redacted error body. Never provide the full key.

<!-- reference -->
## Streaming endpoint

Streaming uses `<site-root>/v1beta/models/<model-id>:streamGenerateContent?alt=sse`. Its response differs from OpenAI SSE and must not be parsed with the Chat Completions `data: [DONE]` rule.

`HttpOptions` is versioned SDK surface. Rerun the minimal request after upgrading `google-genai`. When SDK behavior differs from REST, keep REST as the protocol baseline and record the exact SDK version.
