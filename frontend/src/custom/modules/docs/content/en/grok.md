<!-- beginner -->
## Step 1: Prepare the connection details

Before you begin, have a signed-in site account, permission to create an API key, and a terminal that can run `curl`. This guide uses `sk-demo-replace-me` as the sample key. Never put a real key in screenshots, chat messages, or support tickets.

## Step 2: Create a dedicated Grok key

Open **API Keys** and create a separate key. Include the purpose and client in its name, such as "Grok-test-machine." Start with a small quota. Add IP allowlists or similar restrictions only after the minimal request succeeds so that access restrictions are not confused with connection errors.

![Grok Base URL, API key, and model fields](asset:grok/grok-api-fields.png "Generic Grok API settings")

## Step 3: Confirm the group and available models

Select a Grok group while creating the key. After saving, confirm that the key details show the group and that an administrator has configured usable accounts, a model allowlist, and any required media permission for that group.

Read the models visible to this exact key:

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Only an ID in `data[].id` is safe to enter as the model. Do not guess `grok-4.5`, `grok-4.6`, or a media model from an older tutorial.

## Step 4: Copy the site's api_base_url

Copy `api_base_url` from the key's **Use key** dialog. It normally already ends in `/v1`; the examples append `/models` or `/responses` to it.

To make a copied configuration easy to verify, the examples below use the site's public example address `https://api.daigua.icu/v1`:

- enter `https://api.daigua.icu/v1` as the Base URL;
- the Responses request URL is `https://api.daigua.icu/v1/responses`;
- do not add a second `/v1`;
- do not enter a full `/chat/completions` or `/responses` request URL as the Base URL.

For a real request, always use the `api_base_url` shown in the key's **Use key** dialog. If an administrator configured another entry point, replace only the host in these examples.

## Step 5: Fill in each field

| Field | What to enter | Common mistake |
| --- | --- | --- |
| Base URL | The site's `<api_base_url>` | Adding `/v1` twice |
| API Key | Put the real key where `sk-demo-replace-me` appears | Using a panel password or upstream xAI key |
| Protocol | Prefer `Responses` | Selecting a Responses-only group for a Chat-only client |
| Model | A `data[].id` returned in step 3 | Copying a display label instead of the model ID |
| Storage | Set `store: false` for sensitive use cases | Assuming it disables required site logging and auditing |

Authenticate with `Authorization: Bearer <API Key>`. API keys in query parameters are deprecated; never append a key to the URL.

## Step 6: Select a model

Copy one text model ID from the `/models` response in step 3. Images, video, and voice use different models and endpoints; do not put a media model in the text request below. Grok groups do not provide embeddings. Use an OpenAI group with embedding support when vectors are required.

## Step 7: Send a minimal test message

Replace `<api_base_url>` and `<grok-model-id>` with the values already verified:

```bash
curl "<api_base_url>/responses" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<grok-model-id>","input":"Reply only OK","store":false}'
```

For the first test, send text only. Do not enable images, tools, long context, or streaming at the same time. Add those capabilities one at a time after the minimal request works.

## Step 8: Decide whether the test succeeded

Treat the connection as successful only when all of these are true:

- the HTTP status is `200`;
- the response contains `id` and `output`;
- assistant text in `output` contains `OK`;
- the response model matches the requested model or the administrator's documented mapping;
- the body is not a login page, reverse-proxy error page, or a request to a duplicated `/v1/v1` path.

## Step 9: Check these three items first

1. **Key and group:** the key has not expired or exhausted its quota and is assigned to a Grok group.
2. **Base URL and protocol:** `api_base_url` does not contain a duplicated `/v1`, and the client is actually sending Responses, Chat, or Messages as expected.
3. **Model and limits:** the model is still returned by `/models`; for `429`, read `Retry-After` instead of retrying rapidly.

## Step 10: Provide diagnostic information if it still fails

Send support the timestamp and timezone, client name and version, request endpoint, model ID, HTTP status, `X-Request-ID`, streaming mode, key group, and a redacted error body. Keep only the first and last 3 to 4 characters of a key. Never send a complete key, cookie, OAuth token, or password.

<!-- reference -->
## API reference: Responses and Chat

xAI currently recommends Responses. It supports `previous_response_id`, native tools, and typed `output`; Chat Completions is a compatibility API where clients resend conversation history.

Use this minimal compatibility request only when a client requires Chat:

```bash
curl "<api_base_url>/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<grok-model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

Successful text is in `choices[0].message.content`. Chat availability depends on the Grok group and upstream account. A working Responses request does not prove Chat works, or the reverse. xAI may store Responses for 30 days by default. `store: false` asks the upstream not to store that response, but does not change site logging required for security, billing, or legal obligations.

## API reference: Text and media endpoints

| Capability | Endpoint | Notes |
| --- | --- | --- |
| Model list | `GET /v1/models` | Use what the current key can see |
| Text | `POST /v1/responses` | Preferred text protocol |
| Chat compatibility | `POST /v1/chat/completions` | Depends on group and conversion support |
| Messages | `POST /v1/messages` | Used by clients such as Claude Code |
| Images | `POST /v1/images/generations`, `POST /v1/images/edits` | Requires a media model and image permission |
| Video | `POST /v1/videos/generations`, then poll `GET /v1/videos/{request_id}` | Asynchronous; pending is not failure |
| Voice | `POST /v1/tts`, `POST /v1/stt`, `/v1/custom-voices` | Audio or structured output |
| Realtime voice | `/v1/realtime` WebSocket | Separate from HTTP text configuration |

The site can pass through `x-ratelimit-limit-*`, `x-ratelimit-remaining-*`, `x-ratelimit-reset-*`, and `Retry-After`. For `429`, wait as instructed and use capped exponential backoff.

## Administrator: Configure a Grok upstream

1. Select Grok OAuth or API Key for the administrator account. Prefer an OAuth grant or a dedicated API key; never collect user passwords, raw SSO cookies, or reusable sign-in credentials.
2. The default upstream for an API-key account is `https://api.x.ai/v1`. An OAuth custom Base URL changes chat, media, and probe traffic only; OAuth authorization and token refresh continue to use official endpoints.
3. Set a custom Base URL only when a private gateway or regional proxy requires one, then verify TLS, paths, authentication headers, and each endpoint. Authentication headers are system-managed and must not be replaced through Header Override.
4. Map user-visible model IDs to actual upstream models in groups and model mapping. Test text, images, video, and voice separately; a text mapping does not automatically change media models or permissions.
5. Inspect `x-ratelimit-*`, `Retry-After`, and site rate-limit logs. After changing OAuth or API-key credentials, use a separate test key to verify `/models`, Responses, Chat compatibility, and every enabled media capability before exposing it to users.

Custom Base URLs, model mappings, and account capabilities are administrator settings. Tutorial examples never override the actual backend configuration.
