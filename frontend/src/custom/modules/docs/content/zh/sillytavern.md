<!-- beginner -->
## 步骤 1：准备 SillyTavern 和本站账号

确认 SillyTavern 版本，本文按 `1.18.0` 的通用界面编写。先登录本站并确认账户有可用余额或订阅；第一次接入不要先导入角色卡、世界书或复杂预设。

## 步骤 2：创建专用 API Key

打开本站 **API 密钥**，创建名称为 `sillytavern` 的独立 Key。选择包含目标模型的分组，测试额度先设小一些；首次请求成功前先不设置 IP 限制。

## 步骤 3：确认分组和模型

使用该 Key 请求模型列表：

```bash
curl "<api_base_url>/models" \
  -H "Authorization: Bearer sk-demo-replace-me"
```

复制返回的精确模型 ID。列表中没有目标模型时，先调整 Key 分组或模型白名单，不要继续在 SillyTavern 中反复重试。

## 步骤 4：复制 Base URL

在 Key 的“使用”弹窗复制本站显示的 `api_base_url`。它通常以 `/v1` 结尾，但应以弹窗实际值为准；不要自己拼域名，也不要追加 `/chat/completions`。

## 步骤 5：打开 API Connections

打开 SillyTavern 顶部或侧边栏的 **API Connections**。不同主题会改变图标位置，但字段名称不变。

![SillyTavern Custom OpenAI-compatible 字段和测试按钮](asset:sillytavern/custom-openai.png "SillyTavern 1.18 通用连接示意图")

## 步骤 6：选择连接类型

按顺序选择：

1. `API Type = Chat Completion`
2. `Chat Completion Source = Custom (OpenAI-compatible)`

不要选择锁定官方 OpenAI 地址的预设。本站接入需要可填写自定义 Base URL 的来源。

## 步骤 7：逐项填写字段

| SillyTavern 字段 | 这里填什么 | 常见错误 |
| --- | --- | --- |
| Custom Endpoint (Base URL) | 本站 `api_base_url` | 追加了 `/chat/completions` 或重复 `/v1` |
| Custom API Key | SillyTavern 专用 Key | 复制了遮罩值、Key 已停用或过期 |
| Enter a Model ID | `/models` 返回的精确模型 ID | 使用教程中的示例名或多了空格 |
| Available Models | 加载成功后选择同一模型 | 列表为空时仍随便选择 |

SillyTavern 会自动请求 `{base}/models` 和 `{base}/chat/completions`。

## 步骤 8：保持兼容选项默认值

第一次测试将 `Prompt Post-Processing` 设为 `None`，流式开关保持客户端默认。需要工具调用时不要选择 `no tools` 变体；严格模式也不要作为“更兼容”的默认选项。

## 步骤 9：连接并发送 Test Message

先点击 `Connect`，确认没有鉴权错误，再点击 `Test Message`。测试会产生少量用量。看到正常 assistant 回复，且本站 **用量** 页面能查到同一模型的记录，才算连接成功。

## 步骤 10：逐步恢复聊天功能

成功后再依次启用流式回复、角色卡、世界书、工具和长上下文。每次只增加一项，出现问题时就能定位到具体设置；不要在首次测试时一次打开全部扩展。

## 失败时先检查三项

1. Base URL 是否原样复制，且没有追加端点或重复 `/v1`。
2. Key 是否有效、已分组，`/models` 是否包含当前模型。
3. 当前分组是否支持 Chat Completions；Responses-only 分组需要更换客户端或分组。

仍失败时提交 SillyTavern 完整版本、时间和时区、Base URL、模型、流式设置、Post-Processing、HTTP 状态码、`X-Request-ID` 和脱敏截图。不要提交 Key、Cookie、角色卡或私聊原文。

<!-- reference -->
## Prompt Post-Processing

- `None`：不显式处理提示词，除非所选 API 自身有强制要求。
- `Merge consecutive roles`：合并相邻的同角色消息。
- `Semi-strict`：合并相邻角色，并且只允许一条可选的 system 消息。
- `Strict`：在 Semi-strict 的基础上要求第一条对话消息来自 user。若第一条 assistant 消息前没有 user 消息，SillyTavern 会插入 `config.yaml` 中的 `promptPlaceholder`，默认内容为 `[Start a new chat]`。
- `Single user message (no tools)`：把所有角色的全部消息合并为一条 user 消息，只应作为无法接受多角色消息的上游的最后兼容手段。

`Merge consecutive roles`、`Semi-strict` 和 `Strict` 都有 `with tools` 与 `no tools` 两种变体：`with tools` 保留工具调用，`no tools` 移除工具调用。`Single user message` 仅提供 `no tools` 形式，也会移除工具调用。任何 `no tools` 选项都不适合函数调用、工具型扩展或依赖工具历史的对话。

只有上游明确报连续同角色、多 system 消息或角色交替错误时，才按 `Merge`、`Semi-strict`、`Strict` 逐级尝试。部分旧版本出现 `Bypass API status check`；它不是 Custom 接入的稳定必填项，只在真实请求成功而状态检测误报时考虑。

## 协议边界

SillyTavern 的 Custom OpenAI-compatible 连接使用 Chat Completions。若某个 Grok 分组只接受 Responses，修改 Prompt Post-Processing 无法解决协议不匹配，应改用支持 Responses 的客户端或选择提供 Chat 兼容能力的分组。
