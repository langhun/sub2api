<!-- beginner -->
## Step 1: Record the client name and version

Tavern clients and third-party forks vary widely. Record the exact product name and version from About or Settings. This guide uses a generic field map and does not treat one fork's interface as the only standard.

![Base URL, API key, model, and streaming fields in compatible clients](asset:tavernai/field-map.png "Generic TavernAI-compatible client screen")

## Step 2: Create a client-specific key

Create a separate key under **API Keys** with the client and device in its name, such as `tavernai-desktop`. Select a group containing the target model, set a small quota, and leave IP restrictions empty until the first request works.

## Step 3: Confirm models visible to the key

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Keep an exact model ID from the response. If it is absent, fix the key group, model allowlist, or enabled state instead of guessing a name in the client.

## Step 4: Copy the site Base URL

Copy `api_base_url` from the key usage dialog. Whether it includes `/v1` is determined by that dialog. Do not copy the administration page URL from the browser address bar.

## Step 5: Open Provider Settings

Look for `Provider Settings`, `API Settings`, `Connections`, or `Backend`. Disable automatic character replies first so saving the form does not immediately send a complex request.

## Step 6: Choose a provider with an editable address

Select `Custom OpenAI-compatible` in modern compatible clients. Legacy TavernAI v1 must use `Reverse Proxy`; selecting `OpenAI` locks the official host and cannot connect to this site.

## Step 7: Map field names

| Possible name | Select or enter | Common mistake |
| --- | --- | --- |
| Provider / API / Backend / Source | Custom OpenAI-compatible or Reverse Proxy | Selecting OpenAI with a locked host |
| Base URL / API address / Endpoint / Proxy URL | The site's `api_base_url` | Entering an admin page or full endpoint |
| API Key / Token / Password / Proxy password | The client-specific key | Pasting a masked value or spaces |
| Model / Model ID / Custom model | Exact ID returned by `/models` | Guessing from a display name |
| Streaming / Stream response | Keep the default first; disable for diagnosis | Changing several switches at once |
| Connect / Test | Connect and send a minimal request | Importing a long preset before testing |

The exact legacy TavernAI v1 fields are `API Proxy url adress`, `API key or Prpxy password`, and `Custom proxy model`. Those spellings come from the old interface.

## Step 8: Inspect the final request path

Do not append `/chat/completions` to the Base URL. Most compatible clients request `{base}/models` and `{base}/chat/completions`. For `404`, inspect the final URL in developer tools or logs, especially duplicate `/v1` and endpoint suffixes.

## Step 9: Send a minimal test message

Disable tools, lorebooks, and complex presets, turn streaming off, and send `Reply only OK`. A normal assistant response plus a matching **Usage** record means the connection works.

## Step 10: Enable character features one at a time

Restore streaming, characters, lorebooks, tools, and long context one by one. If a Grok group does not support Chat Completions, use a Responses-capable client or a group with Chat compatibility.

## Check these three items first

1. The Base URL came from the site dialog and has no endpoint suffix or duplicate `/v1`.
2. The key is enabled and assigned to a group, and `/models` contains the configured model.
3. The provider accepts a custom address and the client is actually using Chat Completions.

For further diagnosis provide the exact client name and version, time and timezone, final request URL, model, streaming setting, HTTP status, `X-Request-ID`, and a redacted error body. Never include a key, cookie, character card, or conversation.

<!-- reference -->
## Version field map

| Client | Provider | Address field | Model field |
| --- | --- | --- | --- |
| SillyTavern 1.18 | Custom (OpenAI-compatible) | Custom Endpoint (Base URL) | Enter a Model ID |
| TavernAI v1 legacy | Reverse Proxy | API Proxy url adress | Model / Custom proxy model |
| TavernAI 2 or a third-party fork | Follow that version's UI | Base URL / Endpoint / Proxy URL | Model / Model ID |

## Generic minimal request

```bash
curl "<api_base_url>/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}],"stream":false}'
```

Use the error-code topic for 401, 403, 404, 429, and 5xx. Similarly named fields in third-party forks may append paths differently, so diagnostics must record the final request URL.
