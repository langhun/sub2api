<!-- beginner -->
## Step 1: Prepare the client and config directory

Install and confirm that at least one client starts: Grok Build CLI, Codex, Claude Code, or OpenCode. Record the client version and locate its configuration file:

| Client | Windows | macOS / Linux |
| --- | --- | --- |
| Grok Build CLI | `%USERPROFILE%\.grok\config.toml` | `~/.grok/config.toml` |
| Codex | `%USERPROFILE%\.codex\config.toml` | `~/.codex/config.toml` |
| Claude Code | `%USERPROFILE%\.claude\settings.json` | `~/.claude/settings.json` |
| OpenCode | `~/.config/opencode/opencode.json` | `~/.config/opencode/opencode.json` |

Do not commit these user files or real keys to a project repository. The only sample key below is `sk-demo-replace-me`.

## Step 2: Create a client-specific key

Create a separate key under **API Keys** and name it after the client, such as "Grok-Codex-desktop." Start with a small quota and leave IP allowlists off for the first test. Give each device or client its own key so that it can be revoked and diagnosed independently.

## Step 3: Confirm the Grok group and model

The key must be assigned to a Grok group. Query models with this exact key; do not use models visible only to an administrator or fixed names from an old tutorial:

```bash
curl "https://api.daigua.icu/v1/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Select one text model from `data[].id`. Codex and Grok CLI require Responses support from the group. Claude Code requires the site's Messages bridge. Use OpenCode only with a group that has passed a minimal Chat Completions request.

## Step 4: Copy api_base_url and distinguish site-root

Copy `api_base_url` from the key's **Use key** dialog rather than guessing from the domain. The current public DaiGua example is `https://api.daigua.icu/v1`:

- Grok CLI, Codex, and OpenCode use the full `<api_base_url>`;
- Claude Code uses `<site-root>` for `ANTHROPIC_BASE_URL`, which is `https://api.daigua.icu` after removing the final `/v1`;
- do not append `/responses`, `/messages`, or `/chat/completions` to a Base URL field.

If the dialog shows a different address later, use that value. The address above is the current recognizable site example.

## Step 5: Fill in fields for the selected client

![Grok CLI Responses backend configuration](asset:grok/grok-cli-config.png "Generic Grok CLI configuration")

Field mapping:

| Client | Base URL field | Key field | Protocol field |
| --- | --- | --- | --- |
| Grok CLI | `base_url = "https://api.daigua.icu/v1"` | `env_key = "XAI_API_KEY"` | `api_backend = "responses"` |
| Codex | `base_url = "https://api.daigua.icu/v1"` | `env_key = "SUB2API_API_KEY"` | `wire_api = "responses"` |
| Claude Code | `ANTHROPIC_BASE_URL=https://api.daigua.icu` | `ANTHROPIC_API_KEY=sk-demo-replace-me` | Always uses Messages |
| OpenCode | `options.baseURL = "https://api.daigua.icu/v1"` | `options.apiKey = "{env:SUB2API_API_KEY}"` | Only verified Chat Completions |

### Grok Build CLI

Add this to the user-level `config.toml`:

```toml
[model."site-grok"]
model = "<grok-model-id>"
base_url = "https://api.daigua.icu/v1"
name = "Site Grok"
env_key = "XAI_API_KEY"
api_backend = "responses"

[models]
default = "site-grok"
```

PowerShell:

```powershell
$env:XAI_API_KEY="sk-demo-replace-me"
```

macOS / Linux:

```bash
export XAI_API_KEY="sk-demo-replace-me"
```

Keep `api_backend = "responses"` explicit. This guide uses the currently documented per-model `base_url` instead of private or version-specific `[endpoints]` fields from older configurations.

### Codex

![Codex custom Responses provider fields](asset:grok/codex-provider.png "Generic Codex custom provider configuration")

Add this to the user-level `config.toml`:

```toml
model_provider = "site-grok"
model = "<grok-model-id>"

[model_providers.site-grok]
name = "Site Grok"
base_url = "https://api.daigua.icu/v1"
env_key = "SUB2API_API_KEY"
wire_api = "responses"
requires_openai_auth = false
supports_websockets = false
```

Set `SUB2API_API_KEY`:

```powershell
$env:SUB2API_API_KEY="sk-demo-replace-me"
```

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

A Codex custom provider supports Responses only. Do not combine `env_key` with a plaintext `experimental_bearer_token` on disk. Change `supports_websockets` only when an administrator has explicitly verified Responses WebSocket support for this route.

### Claude Code

![Redacted DaiGua configuration showing the Claude Code site root, API key, and model mappings](asset:grok/claude-code-env.png "DaiGua redacted configuration: Claude Code Messages gateway")

Claude Code requests `/v1/messages` after `<site-root>`, so do not enter an `api_base_url` ending in `/v1` here:

```powershell
$env:ANTHROPIC_BASE_URL="https://api.daigua.icu"
$env:ANTHROPIC_API_KEY="sk-demo-replace-me"
$env:ANTHROPIC_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_OPUS_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_SONNET_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_HAIKU_MODEL="<grok-model-id>"
$env:ANTHROPIC_DEFAULT_FABLE_MODEL="<grok-model-id>"
$env:CLAUDE_CODE_SUBAGENT_MODEL="<grok-model-id>"
```

```bash
export ANTHROPIC_BASE_URL="https://api.daigua.icu"
export ANTHROPIC_API_KEY="sk-demo-replace-me"
export ANTHROPIC_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_OPUS_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_SONNET_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_HAIKU_MODEL="<grok-model-id>"
export ANTHROPIC_DEFAULT_FABLE_MODEL="<grok-model-id>"
export CLAUDE_CODE_SUBAGENT_MODEL="<grok-model-id>"
```

Map every model family and the subagent model so background work does not fall back to a Claude model absent from the group. Use `ANTHROPIC_API_KEY` by default and do not set `ANTHROPIC_AUTH_TOKEN` at the same time. Switch to the token only when an administrator explicitly says that this gateway accepts Bearer only, and remove the API-key variable first. Set `CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1` when nonessential outbound traffic must be blocked, but note that it also disables gateway model discovery, so explicit mappings remain necessary.

### OpenCode

![Redacted DaiGua OpenCode Chat Completions provider configuration with address, environment key, and model ID](asset:grok/opencode-provider.png "DaiGua redacted configuration: OpenCode Chat Completions provider")

First set the environment variable that OpenCode reads in the same terminal:

```powershell
$env:SUB2API_API_KEY="sk-demo-replace-me"
```

```bash
export SUB2API_API_KEY="sk-demo-replace-me"
```

Then pass a minimal Chat Completions request with the same key before using OpenCode's official OpenAI-compatible provider:

```bash
curl "https://api.daigua.icu/v1/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<grok-model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

On Windows PowerShell, write the command as `curl.exe`. Save the OpenCode configuration below only after it returns `OK`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "site/<grok-model-id>",
  "provider": {
    "site": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "DaiGua Chat Completions",
      "options": {
        "baseURL": "https://api.daigua.icu/v1",
        "apiKey": "{env:SUB2API_API_KEY}"
      },
      "models": {
        "<grok-model-id>": { "name": "Grok" }
      }
    }
  }
}
```

`@ai-sdk/openai-compatible` uses `/v1/chat/completions`, so it cannot connect to a Responses-only group. Use Grok Build CLI or Codex for a Responses-only group; do not assume that changing the npm package makes it compatible.

## Step 6: Select the model in the client

- Grok CLI: run `grok models`, select `site-grok`, or pass `-m site-grok` at startup.
- Codex: use `<grok-model-id>` from the config or pass `--model <grok-model-id>` explicitly.
- Claude Code: run `/model`; if the custom name is not listed, use `--model <grok-model-id>` or the environment variable above.
- OpenCode: run `/models` and select `site/<grok-model-id>`.

Display names are customizable, but the model ID sent to the site must exactly match a `data[].id` from step 3.

## Step 7: Send a minimal test message

Run the relevant commands in the same terminal where the environment variable is set:

```text
grok inspect
grok -p "Reply only OK" -m site-grok
```

```text
codex exec --model <grok-model-id> "Reply only OK"
```

```text
claude --model <grok-model-id> -p "Reply only OK"
```

```text
opencode debug config
opencode run --model "site/<grok-model-id>" "Reply only OK"
```

Do not enable MCP, images, tool plugins, long context, or automatic multi-agent work for the first request. Prove that protocol and authentication work before adding features one at a time.

## Step 8: Decide whether each test succeeded

- Grok CLI: `grok inspect` shows the user config and `site-grok`, and the test returns `OK`.
- Codex: the command exits with code `0`, does not request ChatGPT OAuth, and does not loop after a WebSocket failure.
- Claude Code: `/status` in an interactive session shows the site `Anthropic base URL` and `ANTHROPIC_API_KEY`, and the message returns `OK`.
- OpenCode: `opencode debug config` shows the `site` provider, `@ai-sdk/openai-compatible`, and the correct Base URL, and the test returns `OK`.

These are four independent acceptance checks. One working client does not prove another protocol works.

## Step 9: Check these three items first

1. **The environment variable reached the process:** variables may disappear after opening a new terminal; `env_key` in a config file is a variable name, not the key itself.
2. **Protocol and Base URL match:** Codex requires Responses; Claude Code requires Messages and `https://api.daigua.icu`; OpenCode connects only to a verified Chat Completions group.
3. **The model and group remain available:** query `/models` again, then check the key group, model allowlist, quota, and `Retry-After` on `429`.

## Step 10: Provide diagnostic information if it still fails

Provide the operating system, exact client version, config path, selected provider and protocol, redacted Base URL, model ID, timestamp and timezone, status, `X-Request-ID`, terminal error, and redacted response body. Include redacted `grok inspect` output for Grok CLI, the address and credential source from Claude Code `/status`, or redacted `opencode debug config` output for OpenCode. Never send a complete key, `auth.json`, cookie, or OAuth token.

<!-- reference -->
## API reference: Client protocol matrix

| Client | Site endpoint | Critical restriction |
| --- | --- | --- |
| Grok Build CLI | `/v1/responses` | Set `api_backend = "responses"` on every text model |
| Codex | `/v1/responses` | `wire_api` supports only `responses` |
| Claude Code | `/v1/messages` | Set `ANTHROPIC_BASE_URL` to the site root |
| OpenCode with `@ai-sdk/openai-compatible` | `/v1/chat/completions` | Only for a verified Chat Completions group; use Grok Build CLI or Codex for Responses-only access |

Grok CLI currently documents `XAI_API_KEY`, `GROK_MODELS_BASE_URL`, `GROK_MODELS_LIST_URL`, and `GROK_XAI_API_BASE_URL`. User config supports per-model `model`, `base_url`, `name`, `env_key`, and `api_backend`. This guide uses per-model `base_url` and does not depend on older `[endpoints]` settings that may change by version.

## Administrator: Checks before enabling CLI clients

1. Verify `/v1/models`, `/v1/responses`, and `/v1/messages` separately with a test key. Verify `/v1/chat/completions` only when Chat is required.
2. Ensure model mappings cover the main model and Claude Code background or subagent models, and that upstream processing does not incorrectly strip Responses tool fields.
3. Claude Code gateways receive Anthropic version and beta headers. The conversion layer must handle headers together with their request fields and preserve diagnostic errors. Changing only a model name does not create protocol compatibility.
4. Validate Codex over HTTP/SSE by default. Tell users to enable a Responses WebSocket option only after that deployment path is verified.
5. Before publishing a config, test with a normal non-administrator key and verify that logs, screenshots, and examples contain only `sk-demo-...`, with no real credential.
