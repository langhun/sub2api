<!-- beginner -->
## Step 1: List models with the current key

```bash
curl "<api_base_url>/models" -H "Authorization: Bearer sk-demo-replace-me"
```

The result is filtered by key group, platform, and model allowlist. Two keys on one account can see different models.

## Step 2: Copy the exact ID

Use `data[].id` in a client's Model field, not a display label, provider name, or historical tutorial default.

## Step 3: Confirm the protocol

Visibility does not imply support for every endpoint. Check Responses, Chat Completions, Messages, embeddings, images, video, and voice separately. Use a minimal request for the selected protocol.

<!-- reference -->
## Compatibility principle

Model mapping can translate a requested ID into an upstream ID; Usage helps compare them. If a visible model returns 404/400, check endpoint capability and parameters. If it is absent, fix the group and allowlist instead of guessing alternate spellings.
