<!-- beginner -->
## Step 1: List models with the current key

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

Results are filtered by key group, platform, and model allowlist. Two keys on one account can see different models. A name shown to an administrator or in an old guide is not proof that the current key can use it.

## Step 2: Separate request ID, display name, and upstream model

- **Requested model ID:** the `model` value sent by the client; it must exactly match `/models`.
- **Display name:** a label used by the client or site and may differ from the request ID.
- **Upstream model:** the ID after model mapping, visible only where the usage view and permissions expose it.

Aliases, mappings, and routing can change the upstream ID. A display name alone does not identify the final route.

## Step 3: Verify protocol capabilities

Presence in `/models` means the key can select a model; it does not guarantee every endpoint or parameter. Test each required protocol with a minimal request:

| Capability | Endpoint |
| --- | --- |
| Responses | `/v1/responses` |
| Chat Completions | `/v1/chat/completions` |
| Messages | `/v1/messages` |
| Embeddings | `/v1/embeddings` |

Images, video, voice, tools, and reasoning levels require separate verification and cannot be inferred from a text model name.

## Model quality questions

### Does one model name always identify one upstream?

Not necessarily. A group can contain multiple accounts or routes, and administrators can configure model mappings. A model ID is a request contract, not proof of a fixed account, subscription tier, or resource pool.

### Is a fixed resource pool or invariant capability guaranteed?

This guide does not make claims that a single request cannot verify. DaiGua can investigate the requested model, group, mapping, selected account, usage record, and upstream response. Visibility depends on administrator permissions and log redaction.

### Why can the same model behave differently?

Prompts, context compaction, tool output, reasoning level, client version, upstream capacity, and routed accounts can all affect results. Reproduce with the same minimal input and record timestamps plus `X-Request-ID`; subjective output differences alone do not prove model substitution.

<!-- reference -->
## Model discovery boundary

`/models` is a dynamic capability view for the current key, not a permanent product catalog. Models may appear, disappear, or become temporarily unavailable because of groups, allowlists, upstream state, or administrator configuration. Clients should refresh the list and report a missing ID clearly.

A quality investigation needs the requested model, upstream model when permitted, protocol, client version, reasoning parameters, input summary, timestamp, status, latency, and `X-Request-ID`. Never submit a full conversation or key.
