<!-- beginner -->
## 步骤 1：选择向量模型

使用 Key 请求 `/v1/models`，选择明确支持 Embeddings 的模型。聊天模型名称不能直接用于向量接口。

## 步骤 2：发送请求

```bash
curl "<api_base_url>/embeddings" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<embedding-model-id>","input":["first document","second document"]}'
```

成功时 `data` 中每一项包含 `embedding` 数组和对应索引。将向量维度与数据库索引配置保持一致。

## 排障

批量大小、输入长度和维度由具体上游决定；本站不承诺统一的“最多 96 条”。如果启用了请求审计，输入可能按站点策略记录，因此不要提交密码或私人数据。

<!-- reference -->
## 请求与响应

端点：`POST <api_base_url>/embeddings`。`input` 可为单个字符串或字符串数组，具体支持以模型为准。重试前先区分参数错误、模型不兼容和上游限流。
