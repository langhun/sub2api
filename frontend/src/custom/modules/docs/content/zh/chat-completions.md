<!-- beginner -->
## 步骤 1：确认客户端支持 Chat Completions

使用 `POST <api_base_url>/chat/completions`。Base URL 配置框只填到 `/v1`，不要追加 `/chat/completions`。

## 步骤 2：发送最小请求

```json
{
  "model": "<model-id>",
  "messages": [
    { "role": "user", "content": "Reply only OK" }
  ],
  "stream": false
}
```

成功时读取 `choices[0].message.content`。如果当前分组只支持 Responses，而客户端只能调用 Chat Completions，请更换支持 Responses 的客户端或分组。

## 失败时先检查三项

1. 最终 URL 是否为 `/v1/chat/completions`。
2. 模型是否出现在该 Key 的 `/v1/models`。
3. 上游能力是否允许 Chat 兼容转换。

<!-- reference -->
## 常用字段

`messages` 是按顺序排列的角色消息；`temperature`、`max_tokens`、工具调用和图片输入是否支持取决于模型与上游。不要用 Chat 的响应结构解析 Responses 或 Messages。
