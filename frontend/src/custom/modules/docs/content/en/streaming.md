<!-- beginner -->
## Step 1: Make a non-streaming request first

Verify the key, model, and endpoint with `stream: false`, then enable `stream: true`. This separates protocol failures from stream parsing failures.

## Step 2: Parse the correct event family

| Protocol | Typical delta | Completion |
| --- | --- | --- |
| Chat Completions | `choices[].delta` | `data: [DONE]` |
| Anthropic Messages | `content_block_delta` | `message_stop` |
| Responses | `response.output_text.delta` | `response.completed` or a failure event |

Do not treat partial text after a network interruption as a complete answer. Record the final event type, time, and `X-Request-ID` before retrying the whole request.

## Common failures

Proxy buffering can delay all chunks until the end. Treating all three protocols as the same event format causes blank output or JSON parse errors.

<!-- reference -->
## SSE notes

Process `text/event-stream` line by line, tolerate heartbeats and unknown events, and use appropriate read timeouts. Resume behavior and charges for an interrupted stream depend on the protocol, upstream, and current billing implementation; neither continuation nor refund is guaranteed.
