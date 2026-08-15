<!-- beginner -->
## Step 1: Use a separate key per client

Create distinct keys for SillyTavern, command-line tools, and server applications. A leaked client can then be revoked without interrupting the others.

![API key creation and group settings](asset:common/api-key-create.png "Generic API key settings")

## Step 2: Select a group and model

The group controls upstream routing and model availability. After creation, call `/models` with that key; the live result is more reliable than model names shown in examples.

## Step 3: Add limits

| Limit | Recommendation |
| --- | --- |
| Key quota | Start small for a new client |
| Expiration | Set a date for temporary tests |
| 5-hour/day/7-day limits | Contain runaway scripts |
| IP allowlist | Use only with a stable server egress IP |
| IP blocklist | Deny known untrusted sources |

The list masks keys by default; use its copy action for the full value. Retest after changing a restriction.

## Common errors

- `API_KEY_DISABLED`: the key is disabled.
- `API_KEY_QUOTA_EXHAUSTED` or `insufficient_quota`: the key quota is exhausted.
- `ACCESS_DENIED`: the current client was rejected by IP rules.
- Model not found: the group does not allow it, or a display name was used instead of the model ID.

<!-- reference -->
## Security boundaries

Key quota and account balance are separate limits. IP evaluation can use a client address supplied by a trusted proxy, so deployments must configure trusted forwarding sources correctly. During rotation, verify the replacement before revoking the old key.
