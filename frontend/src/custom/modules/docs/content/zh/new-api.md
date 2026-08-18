<!-- beginner -->
## 步骤 1：确认适用场景

本教程适用于把本站作为 New API 的上游渠道。客户端原生支持 Responses 时应直接使用 `/v1/responses`；只有下游客户端只能发送 Chat Completions、而目标上游需要 Responses 时，才启用 New API 的实验性转换策略。

## 步骤 2：创建 New API 专用 Key

在本站 **API 密钥** 新建独立 Key，选择目标分组并设置小额测试额度。该 Key 只放在 New API 上游渠道中，不能发给 New API 的下游用户。

## 步骤 3：验证分组和模型

用本站弹窗显示的 `api_base_url` 验证 Key：

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

保存返回的精确模型 ID。看不到目标模型时先检查 Key 分组、模型白名单和启用状态。

## 步骤 4：区分两个 Base URL

New API 会自行追加 `/v1/models`、`/v1/chat/completions` 或 `/v1/responses`，所以两个位置的地址不同：

| 使用位置 | 正确值 |
| --- | --- |
| SDK 或客户端直连本站 | 本站完整 `<api_base_url>`，通常以 `/v1` 结尾 |
| New API 上游渠道 Base URL | `<site-root>`，必须去掉末尾 `/v1` |

本站公开示例 `api_base_url` 为 `https://api.daigua.icu/v1`，因此 New API 渠道只填 `https://api.daigua.icu`。否则可能产生 `/v1/v1/responses`。实际部署请以 Key 的“使用方式”弹窗为准，只去掉末尾的 `/v1`。

## 步骤 5：创建 OpenAI 上游渠道

在 New API 渠道管理中新增渠道：

| 字段 | 这里填什么 |
| --- | --- |
| 类型 | `OpenAI` |
| Base URL | `<site-root>`，不带 `/v1` |
| API Key | 步骤 2 创建的本站 Key |
| 模型 | 步骤 3 实际返回的模型 ID |

保存后记下数字 **渠道 ID**，后面的策略必须精确引用它。

## 步骤 6：关闭渠道请求体透传

编辑该渠道的高级设置，关闭 `Request Body Passthrough`、`Pass Through Body` 或“请求体透传”。不同主题的名称可能不同；渠道透传开启时，Chat 到 Responses 的转换不会执行。

## 步骤 7：配置全局转换策略

进入 **系统设置 -> 模型设置 -> 全局模型配置**；旧主题可能位于 **模型设置 -> 全局设置**。关闭全局“启用请求透传”，找到 `ChatCompletions -> Responses` 兼容配置并填写：

```json
{
  "enabled": true,
  "all_channels": false,
  "channel_ids": [123],
  "model_patterns": [
    "^<model-id>$"
  ]
}
```

把 `123` 替换为步骤 5 的渠道 ID，并用实际入站模型名编写正则。建议保持 `all_channels:false`，只对已验收的本站渠道开启。

## 步骤 8：理解转换链路

策略命中后，请求链路是：

```text
客户端 /v1/chat/completions
  -> New API 转换请求
  -> 本站 /v1/responses
  -> New API 转回 Chat Completions 响应
```

下游客户端继续接收 Chat 格式；非流请求仍应收到一份 JSON，流式请求才接收 SSE。

## 步骤 9：分别验证上下游

先用本站 Key 直测 `<api_base_url>/responses`，再用 **New API 自己签发的下游令牌** 请求 New API 的 `/v1/chat/completions`。两次请求都只发送 `Reply only OK`，不要混用上游 Key 和下游令牌。

最后在本站 **用量** 中查找对应记录。命中转换时，本站看到的入站端点应为 `/v1/responses`；仅仅“渠道测试成功”或返回 `200` 不能证明转换已经触发。

## 步骤 10：失败时定位并提交诊断

先检查三项：New API 渠道 Base URL 是否误带 `/v1`；全局和渠道两级请求体透传是否都已关闭；渠道 ID 与模型正则是否命中真实请求。

仍失败时提交 New API 完整版本、渠道类型与 ID、脱敏 Base URL、原始入站模型名、是否流式、发生时间和时区、HTTP 状态码、`X-Request-ID` 及脱敏错误体。不要提交本站 Key 或 New API 下游令牌。

<!-- reference -->
## 转换触发条件

以下条件必须同时成立：入站路径是 `/v1/chat/completions`；全局请求透传关闭；渠道请求体透传关闭；策略 `enabled:true`；渠道命中 `channel_ids`；至少一条非空 `model_patterns` 正则命中原始模型名。

`enabled:true` 本身不够。正则为空、渠道 ID 不符或任一级透传开启时，请求仍会按 Chat Completions 转发。

## 常见结果

| 现象 | 优先检查 |
| --- | --- |
| 本站仍记录 `/v1/chat/completions` | 转换策略未命中，检查两级透传、渠道 ID 和模型正则 |
| `401` | New API 上游渠道是否使用本站 Key，下游是否使用 New API 自己的令牌 |
| `404` / `405` | 渠道 Base URL 是否误带 `/v1`，日志中是否出现 `/v1/v1/` |
| 页面没有兼容策略 | 当前 New API 版本可能不支持该实验功能，需要升级或直接使用 Responses 客户端 |
| 非流请求收到 SSE | 记录两端版本、响应头和请求 ID，按响应契约问题提交诊断，不要强制客户端错误解析 |
