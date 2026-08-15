<!-- beginner -->
## Step 1: Confirm Chat Completions support

Use `POST <api_base_url>/chat/completions`. Enter the Base URL only through `/v1`; do not append `/chat/completions` in client settings.

## Step 2: Send a minimal request

```json
{
  "model": "<model-id>",
  "messages": [
    { "role": "user", "content": "Reply only OK" }
  ],
  "stream": false
}
```

Read `choices[0].message.content` on success. If the selected group supports Responses only and the client can only use Chat Completions, use a Responses-capable client or another compatible group.

## Check these three items first

1. The final URL is `/v1/chat/completions`.
2. The model appears in `/v1/models` for this key.
3. The upstream capability permits Chat compatibility conversion.

<!-- reference -->
## Common fields

`messages` is an ordered list of role messages. Support for `temperature`, `max_tokens`, tools, and image input depends on the model and upstream. Do not parse Responses or Messages with the Chat response shape.
