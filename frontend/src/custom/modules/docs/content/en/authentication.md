<!-- beginner -->
## Step 1: Copy the correct Base URL

Copy `api_base_url` from the key usage dialog. OpenAI clients normally need a URL containing `/v1`; Anthropic SDKs usually receive the site root and append `/v1/messages`; Gemini uses `/v1beta` paths.

## Step 2: Choose the authentication header

| Protocol | Header |
| --- | --- |
| OpenAI / Responses | `Authorization: Bearer sk-demo-replace-me` |
| Anthropic | `x-api-key: sk-demo-replace-me`; some clients support Bearer |
| Gemini | `x-goog-api-key: sk-demo-replace-me` or a compatible query parameter |

Do not send different keys in multiple headers. A browser frontend should not retain a long-lived key; proxy requests through your own server.

## Step 3: Verify

List models first, then send a minimal text request. For `401`, check the key and header. For `403 ACCESS_DENIED`, check account state, balance, and IP restrictions.

<!-- reference -->
## Base URL rules

Clients append protocol paths differently. During troubleshooting capture the final request URL, not only the value entered in settings. Response header `X-Request-ID` helps trace a request; error JSON differs by protocol and does not always contain a `request_id` field.
