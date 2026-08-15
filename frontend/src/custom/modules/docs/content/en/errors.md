<!-- beginner -->
## Step 1: Record the full symptom

Record client version, time and timezone, final request URL, model ID, streaming mode, HTTP status, error body, and response header `X-Request-ID`. Remove keys, cookies, and private conversation content first.

## Step 2: Act by status

| Status | Common cause | First action |
| --- | --- | --- |
| 400 | Invalid protocol JSON or parameters | Compare against the minimal request |
| 401 | Invalid, disabled, or expired key | Recopy it and check key state |
| 403 | Balance, group, account, or IP restriction | Check key group and account state |
| 404 | Wrong Base URL, endpoint, or model | Inspect final URL and `/models` |
| 429 | Key quota, rate, or upstream limit | Read the error code and retry later |
| 5xx | Temporary site or upstream failure | Preserve request ID and avoid rapid retry |

## Step 3: Minimize reproduction

Disable streaming, tools, and advanced parameters. Keep one user message. Restore settings one at a time after the minimal request succeeds.

<!-- reference -->
## Protocol error shapes

OpenAI commonly returns `{error:{type,message}}`; Anthropic uses `{type:"error",error:{type,message}}`; Gemini uses `{error:{code,message,status}}`. Clients must tolerate missing fields. Use jittered exponential backoff for `429` and temporary `5xx`, but never retry authentication or parameter failures blindly.
