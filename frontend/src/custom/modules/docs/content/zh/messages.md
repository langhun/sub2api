<!-- beginner -->
## 步骤 1：设置 Anthropic 地址和 Key

Anthropic SDK 的 Base URL 通常填写站点根地址，由 SDK 请求 `/v1/messages`。设置 `ANTHROPIC_API_KEY=sk-demo-replace-me`，或按客户端说明使用 Bearer Token。

## 步骤 2：发送最小请求

```bash
curl "<site-root>/v1/messages" \
  -H "x-api-key: sk-demo-replace-me" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"<model-id>","max_tokens":64,"messages":[{"role":"user","content":"Reply only OK"}]}'
```

`content` 可以是字符串，也可以是内容块数组。多模态、工具、思考块和 `output_config` 是否可用取决于模型与转换能力。

## 成功判断与排障

成功响应包含 assistant 内容块。失败时检查站点根地址是否被错误写成两次 `/v1`、`anthropic-version` 是否存在，以及模型是否支持 Messages。

<!-- reference -->
## 与 Chat 的区别

Messages 的 `system` 位于顶层，响应 `content` 是块数组；流式事件使用 `message_start`、`content_block_delta`、`message_stop`，不能按 Chat 的 `[DONE]` 解析。
