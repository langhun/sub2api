<!-- beginner -->
## 步骤 1：先完成非流式请求

先用 `stream: false` 验证 Key、模型和端点，再切换到 `stream: true`。这样可以把协议问题与流式解析问题分开。

## 步骤 2：按协议解析事件

| 协议 | 典型增量事件 | 结束标记 |
| --- | --- | --- |
| Chat Completions | `choices[].delta` | `data: [DONE]` |
| Anthropic Messages | `content_block_delta` | `message_stop` |
| Responses | `response.output_text.delta` | `response.completed` 或失败事件 |

网络中断后不要把不完整文本当成完整回答。记录最后一个事件类型、时间和 `X-Request-ID`，再决定是否重新发起整次请求。

## 常见错误

代理缓冲会让流式内容集中到最后才显示；客户端若把三套事件当成同一种格式，会出现空白回答或 JSON 解析错误。

<!-- reference -->
## SSE 要点

客户端应逐行处理 `text/event-stream`，容忍心跳和未知事件，并为读取设置合理超时。断线续传和已产生用量的处理由协议、上游和本站计费逻辑共同决定，不应假设一定可续传或一定退款。
