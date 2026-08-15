<!-- beginner -->
## Step 1: Configure the Anthropic address and key

Anthropic SDKs normally receive the site root and request `/v1/messages` themselves. Set `ANTHROPIC_API_KEY=sk-demo-replace-me`, or use a Bearer token where the client supports it.

## Step 2: Send a minimal request

```bash
curl "<site-root>/v1/messages" \
  -H "x-api-key: sk-demo-replace-me" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"<model-id>","max_tokens":64,"messages":[{"role":"user","content":"Reply only OK"}]}'
```

`content` can be a string or an array of content blocks. Multimodal input, tools, thinking blocks, and `output_config` depend on model and conversion capabilities.

## Success and troubleshooting

A successful response contains assistant content blocks. Check for a duplicated `/v1`, a missing `anthropic-version`, and Messages compatibility when a request fails.

<!-- reference -->
## Differences from Chat

Messages places `system` at the top level and returns a content-block array. Streaming uses `message_start`, `content_block_delta`, and `message_stop`; it must not be parsed as Chat `[DONE]` events.
