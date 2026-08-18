<!-- beginner -->
## Step 1: Confirm the use case

This guide uses the site as an upstream channel in New API. A client that already supports Responses should call `/v1/responses` directly. Enable New API's experimental conversion only when a downstream client can send Chat Completions but the selected upstream needs Responses.

## Step 2: Create a dedicated New API key

Create a separate key under **API Keys**, select the target group, and set a small test quota. This key belongs only in the New API upstream channel and must never be given to downstream New API users.

## Step 3: Verify the group and model

Use the `api_base_url` shown by the site to test the key:

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Keep an exact returned model ID. If it is missing, fix the key group, model allowlist, or enabled state first.

## Step 4: Distinguish the two Base URLs

New API appends `/v1/models`, `/v1/chat/completions`, or `/v1/responses`, so the two fields intentionally differ:

| Where it is used | Correct value |
| --- | --- |
| SDK or client calling this site directly | Full `<api_base_url>`, normally ending in `/v1` |
| New API upstream channel Base URL | `<site-root>` with the trailing `/v1` removed |

For the site's public example `api_base_url` `https://api.daigua.icu/v1`, enter only `https://api.daigua.icu` in the New API channel. Otherwise it may produce `/v1/v1/responses`. In a real deployment, use the value from the key's **Use key** dialog and remove only its final `/v1`.

## Step 5: Create an OpenAI upstream channel

Add a channel in New API:

| Field | What to enter |
| --- | --- |
| Type | `OpenAI` |
| Base URL | `<site-root>` without `/v1` |
| API Key | The site key from Step 2 |
| Models | Exact model IDs returned in Step 3 |

Save it and record the numeric **channel ID**. The policy must reference that exact ID.

## Step 6: Disable channel request-body passthrough

Edit the channel's advanced settings and disable `Request Body Passthrough`, `Pass Through Body`, or its localized equivalent. Labels vary by theme. Conversion is bypassed while channel passthrough is enabled.

## Step 7: Configure the global conversion policy

Open **System Settings -> Model Settings -> Global Model Configuration**; older themes may use **Model Settings -> Global Settings**. Disable global request passthrough, locate `ChatCompletions -> Responses`, and enter:

```json
{
  "enabled": true,
  "all_channels": false,
  "channel_ids": [123],
  "model_patterns": [
    "^<model-id>$"
  ]
}
```

Replace `123` with the channel ID from Step 5 and write regular expressions for the actual inbound model names. Keep `all_channels:false` unless every upstream channel has been verified.

## Step 8: Understand the request path

When the policy matches, the route is:

```text
Client /v1/chat/completions
  -> New API converts the request
  -> This site /v1/responses
  -> New API converts the response back to Chat Completions
```

The downstream client still receives Chat format. A non-streaming request should receive one JSON response, while streaming uses SSE.

## Step 9: Verify both sides separately

First test `<api_base_url>/responses` directly with the site key. Then use a **downstream token issued by New API** to call New API's `/v1/chat/completions`. Send only `Reply only OK` and never interchange the upstream key and downstream token.

Finally find the matching entry under this site's **Usage**. When conversion triggers, this site should record `/v1/responses` as the inbound endpoint. A successful channel test or a `200` response alone does not prove that conversion occurred.

## Step 10: Troubleshoot and submit diagnostics

Check three items first: whether the New API channel Base URL incorrectly includes `/v1`; whether both global and channel request passthrough are disabled; and whether the channel ID plus model regular expression match the real request.

For further diagnosis provide the full New API version, channel type and ID, redacted Base URL, original inbound model, streaming mode, time and timezone, HTTP status, `X-Request-ID`, and a redacted error body. Never include the site key or downstream New API token.

<!-- reference -->
## Conversion conditions

All conditions must be true: inbound path `/v1/chat/completions`; global passthrough disabled; channel passthrough disabled; policy `enabled:true`; selected channel in `channel_ids`; and at least one non-empty `model_patterns` expression matching the original model name.

`enabled:true` alone is insufficient. An empty pattern, different channel ID, or either passthrough setting keeps the request on Chat Completions.

## Common results

| Symptom | Check first |
| --- | --- |
| This site still records `/v1/chat/completions` | Policy did not trigger; check passthrough, channel ID, and model pattern |
| `401` | Upstream channel uses the site key and downstream clients use New API's own tokens |
| `404` / `405` | Channel Base URL may include `/v1`; inspect logs for `/v1/v1/` |
| Compatibility policy is absent | The current New API release may not support the experiment; upgrade or use a Responses client |
| A non-stream request receives SSE | Record both versions, headers, and request ID and report a response-contract issue instead of forcing incorrect parsing |
