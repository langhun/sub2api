<!-- beginner -->
## Step 1: Select an embedding model

List `/v1/models` with the key and choose a model that explicitly supports embeddings. A chat model name is not automatically valid for vector requests.

## Step 2: Send the request

```bash
curl "<api_base_url>/embeddings" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<embedding-model-id>","input":["first document","second document"]}'
```

Each successful `data` item contains an `embedding` array and its index. Match the vector dimension to the database index.

## Troubleshooting

Batch size, input length, and dimensions are upstream-specific; this gateway does not promise a universal 96-item limit. Request auditing may retain input according to site policy, so never send passwords or private data.

<!-- reference -->
## Request and response

Endpoint: `POST <api_base_url>/embeddings`. `input` can be one string or an array where the model supports it. Distinguish invalid parameters, model incompatibility, and upstream rate limits before retrying.
