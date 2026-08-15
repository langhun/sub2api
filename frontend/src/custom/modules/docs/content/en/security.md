<!-- beginner -->
## Step 1: Limit key exposure

Use a separate key per client with reasonable quota, expiration, and periodic limits. Add an IP allowlist only for a stable egress address.

## Step 2: Store it safely

Keep keys in user-level environment variables, operating-system credential storage, or a server secret manager. Never place one in browser frontend code, characters, lorebooks, screenshots, tickets, chat messages, or Git.

## Step 3: Rotate and revoke

If exposure is suspected, disable the old key, create a replacement, update clients, and run a minimal test. Inspect usage for unexpected time, model, and source IP.

<!-- reference -->
## Redaction rules

Examples always use `sk-demo-...`. Prefer removing a key completely from diagnostics rather than keeping a prefix and suffix. Do not send keys in URL query parameters because proxies and browser history can retain full URLs. Periodically remove old environment variables and client configuration backups.
