<!-- beginner -->
## 步骤 1：准备账号和余额

登录本站，确认账户可用且有余额或有效订阅。准备一个支持 OpenAI 兼容接口的客户端；第一次测试建议使用终端中的 `curl`。

## 步骤 2：创建 API Key

打开 **API 密钥**，点击创建。名称用于辨认用途；分组决定能调用哪些模型；额度和有效期可以限制损失范围。

![创建 API Key 时需要填写的四个关键位置](asset:common/api-key-create.png "API Key 创建界面通用示意图")

| 字段 | 这里填什么 |
| --- | --- |
| 名称 | 例如 `my-first-client` |
| 分组 | 选择包含目标模型的分组 |
| 额度 | 新手测试可设置较小额度 |
| IP 限制 | 不确定时先留空，成功后再收紧 |

复制按钮取得完整 Key。本文示例统一写成 `sk-demo-replace-me`，不要把真实 Key 发给他人。

## 步骤 3：确认模型和 Base URL

在 Key 的“使用”弹窗复制本站显示的 `api_base_url`，不要凭印象拼接域名。再请求模型列表：

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

从返回结果复制一个模型 ID。看不到目标模型时，先检查 Key 分组和模型白名单。

## 步骤 4：发送测试消息

```bash
curl "<api_base_url>/chat/completions" \
  -H "Authorization: Bearer sk-demo-replace-me" \
  -H "Content-Type: application/json" \
  -d '{"model":"<model-id>","messages":[{"role":"user","content":"Reply only OK"}]}'
```

响应里出现 assistant 内容并返回 `OK` 即为成功。随后到 **用量** 页面手动刷新，核对模型、状态和费用记录。

## 失败时先检查三项

1. Base URL 是否完全来自本站弹窗，并且没有重复 `/v1`。
2. Key 是否启用、未过期、仍有额度，且分组允许该模型。
3. 客户端使用的协议是否与端点匹配。

仍失败时提交时间、模型 ID、端点、HTTP 状态码、响应头 `X-Request-ID` 和已脱敏错误体；不要提交完整 Key。

<!-- reference -->
## 最小接口参考

- 模型发现：`GET <api_base_url>/models`
- Chat Completions：`POST <api_base_url>/chat/completions`
- Responses：`POST <api_base_url>/responses`
- Anthropic Messages：`POST <site-root>/v1/messages`

`api_base_url` 是否已经包含 `/v1` 以本站弹窗为准。不要把 `/chat/completions` 写进 Base URL 字段。
