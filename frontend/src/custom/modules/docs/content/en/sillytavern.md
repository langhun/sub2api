<!-- beginner -->
## Step 1: Prepare SillyTavern and your account

Confirm the SillyTavern version. This guide follows the generic `1.18.0` interface. Sign in to this site and confirm that the account has balance or an active subscription. Do not import characters, lorebooks, or complex presets before the first connection works.

## Step 2: Create a dedicated API key

Open **API Keys** and create a key named `sillytavern`. Select a group containing the target model and use a small test quota. Leave IP restrictions empty until the first request succeeds.

## Step 3: Confirm the group and model

List models with that key:

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Copy an exact model ID from the response. If the model is absent, fix the key group or model allowlist before retrying in SillyTavern.

## Step 4: Copy the Base URL

Copy `api_base_url` from the site's key usage dialog. It normally ends in `/v1`, but the dialog is authoritative. Do not construct the host yourself and do not append `/chat/completions`.

## Step 5: Open API Connections

Open **API Connections** from the SillyTavern header or sidebar. Themes may move the icon, but the field names remain the same.

![SillyTavern Custom OpenAI-compatible fields and test button](asset:sillytavern/custom-openai.png "Generic SillyTavern 1.18 connection screen")

## Step 6: Select the connection type

Choose in this order:

1. `API Type = Chat Completion`
2. `Chat Completion Source = Custom (OpenAI-compatible)`

Do not choose a preset that locks the official OpenAI host. This integration requires a source with an editable Base URL.

## Step 7: Fill every field

| SillyTavern field | What to enter | Common mistake |
| --- | --- | --- |
| Custom Endpoint (Base URL) | The site's `api_base_url` | Appending `/chat/completions` or duplicating `/v1` |
| Custom API Key | The dedicated SillyTavern key | Copying a masked value or using a disabled key |
| Enter a Model ID | Exact ID returned by `/models` | Using a tutorial name or trailing spaces |
| Available Models | Select the same model after loading | Guessing a model while the list is empty |

SillyTavern automatically requests `{base}/models` and `{base}/chat/completions`.

## Step 8: Keep compatibility defaults

Set `Prompt Post-Processing` to `None` for the first test and keep the default streaming setting. Do not choose a `no tools` variant when tools are needed, and do not treat strict mode as a generally safer default.

## Step 9: Connect and send a Test Message

Click `Connect` and confirm that no authentication error appears, then click `Test Message`. The test uses a small amount of quota. A normal assistant reply plus a matching entry under **Usage** means the connection succeeded.

## Step 10: Restore chat features gradually

Enable streaming, characters, lorebooks, tools, and long context one at a time. If a problem appears, this identifies the setting that introduced it. Do not enable every extension during the first test.

## Check these three items first

1. The Base URL was copied exactly, with no endpoint suffix or duplicate `/v1`.
2. The key is valid and assigned to a group, and `/models` contains the selected model.
3. The group supports Chat Completions; a Responses-only group needs another client or group.

For further diagnosis provide the full SillyTavern version, time and timezone, Base URL, model, streaming setting, Post-Processing mode, HTTP status, `X-Request-ID`, and a redacted screenshot. Never include a key, cookie, character card, or private conversation.

<!-- reference -->
## Prompt Post-Processing

- `None`: applies no explicit prompt processing unless the selected API requires it.
- `Merge consecutive roles`: merges adjacent messages with the same role.
- `Semi-strict`: merges adjacent roles and allows only one optional system message.
- `Strict`: builds on Semi-strict and requires the first conversational message to use the user role. If no user message exists before the first assistant message, SillyTavern inserts `promptPlaceholder` from `config.yaml`; its default value is `[Start a new chat]`.
- `Single user message (no tools)`: merges every message from every role into one user message. Use it only as a last compatibility option for an upstream that cannot accept multi-role messages.

`Merge consecutive roles`, `Semi-strict`, and `Strict` each have `with tools` and `no tools` variants: `with tools` preserves tool calls, while `no tools` removes them. `Single user message` is available only as `no tools` and also removes tool calls. Any `no tools` option is unsuitable for function calling, tool-based extensions, or conversations that depend on tool history.

Only move through `Merge`, `Semi-strict`, and `Strict` when the upstream explicitly rejects consecutive roles, multiple system messages, or role order. Some older versions expose `Bypass API status check`; it is not a stable required field and should be considered only when real requests work but status detection is wrong.

## Protocol boundary

SillyTavern's Custom OpenAI-compatible connection uses Chat Completions. When a Grok group accepts only Responses, Prompt Post-Processing cannot repair the protocol mismatch. Use a Responses-capable client or a group with Chat compatibility.
