<!-- beginner -->
## 步骤 1：使用当前 Key 查询模型

```bash
curl "<api_base_url>/models" -H "Authorization: Bearer sk-demo-replace-me"
```

模型列表按 Key 分组、平台和白名单过滤。同一账户的两个 Key 可能看到不同模型。

## 步骤 2：复制精确 ID

客户端的 Model 字段填写 `data[].id`，不要填写网页展示名、供应商名称或教程中的历史默认值。

## 步骤 3：确认协议

模型可见不等于支持每个端点。分别确认 Responses、Chat Completions、Messages、Embeddings、图片、视频和 Voice 能力。首次测试使用该协议的最小请求。

<!-- reference -->
## 兼容性原则

模型映射可能把请求 ID 转换成上游模型 ID；用量页可帮助对照。若 `/models` 可见但请求 404/400，检查端点能力和请求参数；若模型不可见，先处理分组和白名单，不要不断更换拼写。
