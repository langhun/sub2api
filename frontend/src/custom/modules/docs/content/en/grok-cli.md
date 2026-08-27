<!-- beginner -->
## Step 1: Prepare the client and a separate configuration

This guide covers Grok Build CLI, Codex, Claude Code, OpenCode, OpenClaw, Hermes Agent, WorkBuddy, and ZCode. Confirm that the client starts and record its version. Do not enable MCP, long context, images, tools, or multi-agent workflows for the first connection.

| Client | Configuration entry or common path | First protocol to test |
| --- | --- | --- |
| Grok Build CLI | `~/.grok/config.toml` | Responses |
| Codex | Windows `%USERPROFILE%\.codex\config.toml`; macOS/Linux `~/.codex/config.toml` | Responses |
| Claude Code | Environment variables or `~/.claude/settings.json` | Messages |
| OpenCode | `~/.config/opencode/opencode.json` | Chat Completions |
| OpenClaw | Control UI → Settings → Model Providers | Chat Completions |
| Hermes Agent | Run `hermes model` | OpenAI-compatible |
| WorkBuddy | Settings → Models → Custom API | Chat Completions |
| ZCode | Settings → Model settings → Add provider | OpenAI-compatible |

Do not commit user configuration files or real keys. The only key shown below is `sk-demo-replace-me`.

## Step 2: Create a client-specific API key

Create a separate key under **API Keys** for each device or client, for example `codex-laptop`. Start with a small quota and no IP allowlist, then tighten the policy after the minimal test succeeds. Separate keys make revocation, usage review, and diagnostics practical.

## Step 3: Confirm the group, protocol, and model

List models with the new key:

```bash
curl "https://api.daigua.icu/v1/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Copy an ID from `data[].id`; do not reuse a fixed model name from an old guide. Codex and Grok CLI require Responses, Claude Code requires Messages, and the configurations below for OpenCode, OpenClaw, WorkBuddy, and ZCode require Chat Completions.

## Step 4: Copy the Base URL and identify appended paths

Prefer `api_base_url` from the key's **Use key** dialog. The current public DaiGua example is `https://api.daigua.icu/v1`.

| Client | Value to enter | Path appended by the client |
| --- | --- | --- |
| Grok CLI, Codex, OpenCode, OpenClaw, Hermes | `https://api.daigua.icu/v1` | `/responses` or `/chat/completions` |
| Claude Code | `https://api.daigua.icu` | `/v1/messages` |
| WorkBuddy | `https://api.daigua.icu/v1` | Appends `/chat/completions` by default |
| ZCode with OpenAI protocol | `https://api.daigua.icu/v1` | OpenAI Chat path |

Do not add `/chat/completions`, `/responses`, or `/messages` to a Base URL field. WorkBuddy accepts a full endpoint only when Custom protocol is enabled; this guide keeps it disabled.

## Step 5: Configure the selected client

### Grok Build CLI

![Chinese annotated reference showing Grok Build CLI with the DaiGua Responses URL, environment variable, and dynamic model ID](asset:grok/grok-cli-config.png "Grok Build CLI: keep the Responses backend and read the key from an environment variable")

```toml
[model."daigua-grok"]
model = "<model-id>"
base_url = "https://api.daigua.icu/v1"
name = "DaiGua Grok"
env_key = "XAI_API_KEY"
api_backend = "responses"

[models]
default = "daigua-grok"
```

### Codex

![Chinese annotated verification layout for a Codex 0.147.0 custom Responses provider](asset:grok/codex-provider.png "Codex: Responses provider, DaiGua URL, and WebSocket applicability")

```toml
model_provider = "daigua"
model = "<model-id>"

[model_providers.daigua]
name = "DaiGua"
base_url = "https://api.daigua.icu/v1"
env_key = "SUB2API_API_KEY"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

Keep `wire_api = "responses"`. Grok and other compatibility routes default to HTTP/SSE, so keep WebSockets off. Set `supports_websockets = true` only for a verified OpenAI-native Responses group. `mode_router_v2_enabled` is unrelated and is not a global WebSocket switch.

### Claude Code

![Chinese annotated reference showing the Claude Code site root, API key, and model mappings](asset:grok/claude-code-env.png "Claude Code: omit /v1 from the root URL and map background models")

```bash
export ANTHROPIC_BASE_URL="https://api.daigua.icu"
export ANTHROPIC_API_KEY="sk-demo-replace-me"
export ANTHROPIC_MODEL="<model-id>"
export ANTHROPIC_DEFAULT_OPUS_MODEL="<model-id>"
export ANTHROPIC_DEFAULT_SONNET_MODEL="<model-id>"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="<model-id>"
export CLAUDE_CODE_SUBAGENT_MODEL="<model-id>"
```

### OpenCode

![Chinese annotated reference for an OpenCode Chat Completions provider](asset:grok/opencode-provider.png "OpenCode: use the openai-compatible provider and an environment-variable key")

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "daigua/<model-id>",
  "provider": {
    "daigua": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "DaiGua",
      "options": {
        "baseURL": "https://api.daigua.icu/v1",
        "apiKey": "{env:SUB2API_API_KEY}"
      },
      "models": {
        "<model-id>": { "name": "DaiGua Model" }
      }
    }
  }
}
```

### OpenClaw

![Chinese annotated field reference for an OpenClaw custom provider](asset:clients/openclaw-provider.png "OpenClaw: explicitly register the provider, model, and Chat Completions protocol")

Add a custom provider in Control UI or merge this configuration:

```json5
{
  agents: { defaults: { model: { primary: "daigua/<model-id>" } } },
  models: {
    mode: "merge",
    providers: {
      daigua: {
        baseUrl: "https://api.daigua.icu/v1",
        apiKey: "${DAIGUA_API_KEY}",
        api: "openai-completions",
        models: [{ id: "<model-id>", name: "DaiGua Model" }]
      }
    }
  }
}
```

Setting a default alone is insufficient. A custom model must also exist in `models.providers.daigua.models[]`.

### Hermes Agent

![Chinese annotated field reference for the Hermes Agent Custom Provider wizard](asset:clients/hermes-custom-provider.png "Hermes Agent: use hermes model to enter the custom URL, key, and model")

Run `hermes model`, choose **Custom / OpenAI-compatible**, and enter:

| Field | Value |
| --- | --- |
| Provider name | `daigua` |
| Base URL | `https://api.daigua.icu/v1` |
| API Key | `sk-demo-replace-me` |
| Model | `<model-id>` |

A custom provider has no fixed environment-variable name. Follow the current wizard's storage choice.

### WorkBuddy

![Chinese annotated field reference for WorkBuddy Custom API settings](asset:clients/workbuddy-custom-api.png "WorkBuddy: keep Custom protocol off and let the client append the Chat Completions path")

Open **Settings → Models → Custom API**:

| Field | Value |
| --- | --- |
| URL | `https://api.daigua.icu/v1` |
| API Key | `sk-demo-replace-me` |
| Model name | `<model-id>` |
| Custom protocol | Off |

With Custom protocol off, WorkBuddy validates and appends `/chat/completions`. Enable it only when entering a full endpoint. Verify image, tool, and reasoning capabilities independently before enabling them.

### ZCode

![Chinese annotated field reference for a ZCode custom OpenAI provider](asset:clients/zcode-custom-provider.png "ZCode: add an OpenAI-compatible provider and register the model manually")

Open **Settings → Model settings → Add provider**, then select OpenAI-compatible:

| Field | Value |
| --- | --- |
| Name | `DaiGua` |
| API Base URL | `https://api.daigua.icu/v1` |
| API Key | `sk-demo-replace-me` |
| Model ID | `<model-id>` |

Use the Anthropic protocol only after `/v1/messages` succeeds for the target group; its Base URL is `https://api.daigua.icu`.

## Step 6: Set key environment variables and select a model

PowerShell:

```powershell
$env:SUB2API_API_KEY="sk-demo-replace-me"
$env:XAI_API_KEY="sk-demo-replace-me"
$env:DAIGUA_API_KEY="sk-demo-replace-me"
```

macOS/Linux:

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
export XAI_API_KEY="sk-demo-replace-me"
export DAIGUA_API_KEY="sk-demo-replace-me"
```

Display names can differ, but the `<model-id>` sent to DaiGua must exactly match the current key's `/models` response.

## Step 7: Send a minimal test message

Verify the protocol with curl first, then send `Reply only OK` in the client:

```bash
curl "https://api.daigua.icu/v1/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

- Grok CLI: run `grok inspect`, then send a minimal prompt.
- Codex: `codex exec --model <model-id> "Reply only OK"`.
- Claude Code: `claude --model <model-id> -p "Reply only OK"`.
- OpenCode: run `opencode debug config`, then send a minimal prompt.
- OpenClaw: click **Test connection**, then send a message in Control UI.
- Hermes: start `hermes` and send the message in a new session.
- WorkBuddy and ZCode: select the new model in an empty conversation and send the message.

## Step 8: Confirm success

Success requires all of the following: the client displays the DaiGua provider, the model ID is correct, the prompt returns `OK`, and the DaiGua usage page shows the request without a protocol conversion error. Listing models does not prove that chat works, and one client succeeding does not validate another protocol.

## Step 9: Check these three items first

1. **Final URL:** eliminate duplicate `/v1` or `/chat/completions`, and ensure Claude Code did not receive `/v1` in its root URL.
2. **Protocol and capability:** test Responses, Messages, and Chat Completions separately; restore tools, images, and reasoning parameters one at a time.
3. **Key, group, and model:** confirm the environment variable reached the process, the key is enabled, the model remains in `/models`, and check quota plus `Retry-After`.

## Step 10: Provide diagnostics if it still fails

Provide the operating system, client and version, configuration entry, provider, redacted Base URL, model ID, protocol, timestamp and timezone, final request URL, status code, `X-Request-ID`, redacted error body, and a minimal reproduction. Never provide a full key, credential file, cookie, OAuth token, workspace source, or private conversation.

<!-- reference -->
## Reference: client protocol matrix

| Client | DaiGua endpoint | Constraint |
| --- | --- | --- |
| Grok Build CLI | `/v1/responses` | `api_backend = "responses"` |
| Codex | `/v1/responses` | `wire_api = "responses"`; WebSocket depends on the route |
| Claude Code | `/v1/messages` | Base URL is the site root |
| OpenCode | `/v1/chat/completions` | `@ai-sdk/openai-compatible` cannot use a Responses-only group |
| OpenClaw | `/v1/chat/completions` | Register custom models in the provider catalog |
| Hermes Agent | `/v1/chat/completions` | Use a Custom OpenAI-compatible provider |
| WorkBuddy | `/v1/chat/completions` | The client appends the endpoint by default |
| ZCode | `/v1/chat/completions` or `/v1/messages` | Select the protocol supported by the group |

## WebSocket and advanced capabilities

DaiGua exposes a `GET /v1/responses` WebSocket ingress, but client settings must still match the selected group and upstream. `supports_websockets = true` is not a universal speed switch and can cause reconnect loops when used on the wrong route. `mode_router_v2_enabled = false` only disables the new account-mode router; it does not disable WebSockets.

Do not infer context windows, images, tool calling, or reasoning levels from a model name. Start with a minimal text request and enable one capability at a time. Unsupported parameters generated by a client can produce `400` responses.
