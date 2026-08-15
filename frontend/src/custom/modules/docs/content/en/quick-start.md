<!-- beginner -->
## Step 1: Prepare your account

Sign in and confirm that the account has balance or an active subscription. Use `curl` for the first test so client-side defaults cannot hide configuration mistakes.

## Step 2: Create an API key

Open **API Keys** and create a key. The group controls available models; quotas and expiration limit exposure.

![The four key fields when creating an API key](asset:common/api-key-create.png "Generic API key creation screen")

| Field | What to enter |
| --- | --- |
| Name | For example `my-first-client` |
| Group | A group containing the target model |
| Quota | A small amount for initial testing |
| IP rules | Leave empty until the first request works |

Use the copy action to obtain the full key. `sk-demo-replace-me` is a placeholder, never a real credential.

## Step 3: Confirm the model and Base URL

Copy `api_base_url` from the site's key usage dialog, then list models:

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Copy an actual model ID from the response. If it is missing, check the key group and model allowlist.

## Step 4: Send a test message

```bash
curl "<api_base_url>/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

An assistant response containing `OK` means the request worked. Refresh **Usage** and confirm its model, status, and charge.

## Check these three items first

1. The Base URL exactly matches the site dialog and `/v1` is not duplicated.
2. The key is enabled, unexpired, within quota, and its group allows the model.
3. The client protocol matches the endpoint.

For further diagnosis provide time, model ID, endpoint, HTTP status, response header `X-Request-ID`, and a redacted error body. Never send the full key.

<!-- reference -->
## Minimal endpoint reference

- Models: `GET <api_base_url>/models`
- Chat Completions: `POST <api_base_url>/chat/completions`
- Responses: `POST <api_base_url>/responses`
- Anthropic Messages: `POST <site-root>/v1/messages`

Whether `api_base_url` already contains `/v1` is determined by the site dialog. Do not put `/chat/completions` in a Base URL field.
