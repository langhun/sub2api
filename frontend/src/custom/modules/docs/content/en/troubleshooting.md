<!-- beginner -->
## Troubleshoot in this order

1. **Key:** copy it again and check enabled state, expiry, quota, and IP rules.
2. **Group:** confirm the key is assigned to the target group.
3. **Model:** confirm the ID remains in `/models` for that key.
4. **Base URL:** inspect the final request URL for duplicate `/v1` or endpoint paths.
5. **Protocol:** use a non-streaming, tool-free request for Responses, Messages, or Chat Completions.
6. **Advanced parameters:** remove images, tools, reasoning levels, long context, and client plugins.
7. **Upstream:** only then investigate capacity, rate limits, account state, or a temporary incident.

## Common status codes

| Status | Check first |
| --- | --- |
| `400` | Request shape, tool schema, reasoning parameters, image format |
| `401` | Key, authorization header, environment variable in the client process |
| `403` | Group, balance, account state, IP policy, or risk policy |
| `404` | Base URL, duplicate endpoint, model ID, unavailable protocol |
| `429` | Key limits, subscription window, upstream rate limit, `Retry-After` |
| `5xx` | Request ID, upstream capacity, timeout, and failover result |

## Unexpected model behavior or latency

Do not start by assuming model substitution. Repeat the same minimal input two or three times with the same key, model, and protocol, then compare:

- the model ID and reasoning level actually sent by the client;
- context compaction, tool calls, or automatic subagents;
- requested model, upstream model, first-token latency, duration, and status in Usage;
- `429`, capacity, or failover records in the same time window.

Capacity can affect one upstream account or model without affecting the whole service. Do not attribute an incident without a timestamp, model, and request ID.

## Incompatible parameters

When plain text works but an advanced request fails, restore one feature at a time: streaming, system instructions, images, tools, reasoning levels, then long context. Third-party clients can send fields unsupported by DaiGua or the upstream. For `400`, retain a redacted request shape instead of retrying the same request rapidly.

## Submit diagnostics

Provide the client and full version, operating system, timestamp and timezone, key ID instead of key content, group, requested model, final URL, protocol, stream setting, status, latency, `X-Request-ID`, and a redacted error body. Screenshots must hide keys, cookies, accounts, source code, and conversations.

<!-- reference -->
## Minimal reproduction and retries

Start with a non-streaming single-message curl request, then restore the client and advanced features in order. Do not automatically retry `400`, `401`, or `403`. For `429` and temporary `5xx`, honor `Retry-After` or use jittered exponential backoff with a retry cap.

Model-quality and capacity investigations must correlate the exact user agent, requested model, and a narrow time window. A display name, client popup, or one subjective answer cannot establish the root cause.
