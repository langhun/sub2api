<!-- beginner -->
## Troubleshoot in a fixed order

1. **Key**: recopy it and check enabled state, expiration, and quota.
2. **Group**: confirm the key is assigned to a group.
3. **Model**: confirm the ID appears in `/models` for this key.
4. **Base URL**: inspect the final URL for duplicate `/v1` or endpoint suffixes.
5. **Protocol**: use a non-streaming, tool-free minimal request.
6. **Upstream**: only then investigate capacity, rate limits, or outages.

## Three common symptoms

- `401`: wrong key or authentication header.
- `403`: missing group, balance, account, or IP policy.
- Models list works but chat fails: protocol capability or advanced parameter incompatibility.

## Submit diagnostics

Provide client/version, time and timezone, final URL, model, stream mode, status, redacted error body, and `X-Request-ID`. Redact keys, cookies, email, and conversation content from screenshots.

<!-- reference -->
## Minimization method

Start with a one-message, non-streaming curl request. Restore the client, streaming, system prompt, character, tools, and advanced parameters one at a time. Never retry 401/403 automatically. Honor `Retry-After` or exponential backoff for 429 and temporary 5xx.
