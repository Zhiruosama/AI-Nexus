# 独立 Chat Agent 集成：gRPC 和 SSE 协议约定

实际消息草案见 [chat-agent-v1.proto](chat-agent-v1.proto)。

## 1. 版本和服务职责

- Protobuf package 固定为 `ainexus.chat.v1`；
- Chat Agent 实现 `CredentialService`、`ConversationService`、`PresetService` 和
  `AgentService`；
- AI-Nexus 是这些服务的内部客户端；
- V1 字段编号发布后不得复用；
- 删除字段时必须 `reserved` 原编号和名称；
- 两个仓库必须从同一个版本化 Proto 规范源生成代码。

草案暂不固定 `go_package`，因为最终 contracts module 尚未确定。开始生成 Go 代码前
必须选择唯一规范源并固定依赖版本，不能在两个仓库复制后各自修改。

## 2. 内部认证 Metadata

推荐 Metadata：

```text
authorization: Bearer <short-lived-internal-jwt>
x-request-id: <request-id>
traceparent: <w3c-trace-context>
```

内部 JWT 最低 Claims：

```json
{
  "iss": "ai-nexus-core",
  "aud": "ai-nexus-chat-agent",
  "sub": "user-uuid",
  "sid": "session-id",
  "jti": "internal-call-id",
  "iat": 1786200000,
  "exp": 1786200060,
  "scope": ["chat:read", "chat:write", "chat:stream"]
}
```

Chat Agent 从认证拦截器获得 Principal，业务 Request 不重复携带可被误信任的
`user_id`。推荐 AI-Nexus 使用非对称私钥签名，Chat Agent 只保存公钥。

## 3. Unary RPC 语义

所有资源操作都基于 Principal 的 `sub` 限定 owner。Chat Agent 不能因为请求来自
内网就省略 owner 条件。

资源不存在与不属于当前用户统一返回 `NOT_FOUND`。参数错误返回
`INVALID_ARGUMENT`，版本或并发冲突返回 `ABORTED`。

Credential 的 Secret：

- 只允许出现在 Create/Update Request；
- 不在任何 Response 中返回；
- List/Get 只返回 `secret_mask`；
- 不进入 gRPC 错误详情。

Create Credential、Conversation 和 Preset 的 `operation_id` 是创建命令幂等键。
相同用户、相同 RPC 和相同 `operation_id` 重复调用时返回第一次创建的资源；相同
幂等键携带不同有效载荷时返回 `ALREADY_EXISTS`，不得创建第二份资源。

## 4. StreamMessage 的受理边界

请求包括：

```text
request_id
conversation_id
content
generation_options
```

`request_id` 是业务幂等键，不等同于每次网络调用的 `x-request-id`。

在发送第一个 `RunAccepted` 前，Chat Agent 必须已经：

1. 验证 Conversation 和 Credential 归属；
2. 检查 `request_id` 幂等；
3. 原子占用 Conversation；
4. 创建 UserMessage 和 AgentRun；
5. 提交数据库事务。

因此客户端收到 `RunAccepted` 后，可以确信 Run 已经有持久化身份。

## 5. ChatEvent 规则

- `sequence` 在一个 Run 内从 1 单调递增；
- 第一个事件必须是 `RunAccepted`；
- `RunCompleted`、`RunFailed`、`RunCancelled` 三者最多出现一个；
- 终态事件必须是最后一个业务事件；
- `Heartbeat` 不改变业务状态；
- `UsageUpdated` 可以出现多次，最终 Usage 以终态 Run 记录为准；
- `ContentDelta` 只能出现在 `RunAccepted` 之后和终态之前；
- `RunFailed/RunCancelled` 指明是否保存了 partial 输出。

## 6. gRPC Status 与流内错误

在 `RunAccepted` 前失败，使用标准 gRPC Status：

| Code | 场景 | AI-Nexus HTTP 映射 |
| --- | --- | --- |
| `INVALID_ARGUMENT` | Content、参数、模型不合法 | `400` |
| `UNAUTHENTICATED` | 内部身份无效 | `401/服务告警` |
| `PERMISSION_DENIED` | Scope 不足 | `403` |
| `NOT_FOUND` | 资源不存在或不属于用户 | `404` |
| `ALREADY_EXISTS` | request_id 已完成或存在 | `409` |
| `ABORTED` | Conversation 有活跃 Run | `409` |
| `RESOURCE_EXHAUSTED` | 用户/Provider 并发或限流 | `429` |
| `UNAVAILABLE` | Chat Agent 或 Provider 暂时不可用 | `503` |
| `DEADLINE_EXCEEDED` | 前置操作超时 | `504` |
| `INTERNAL` | 未分类内部错误 | `500` |

`RunAccepted` 后 gRPC Status 只表示传输层状态，业务失败必须尽量先发送
`RunFailed/RunCancelled`。网络突然断开时，调用方通过 `GetRun` 对账：已经收到
`RunAccepted` 时按 `run_id` 查询，尚未收到时按自己持有的 `request_id` 查询。

## 7. SSE 映射

AI-Nexus 建议向外输出命名事件：

```text
event: accepted
data: {"run_id":"...","user_message_id":1}

event: delta
data: {"sequence":2,"delta":"你"}

event: usage
data: {"prompt_tokens":20,"completion_tokens":1,"total_tokens":21}

event: completed
data: {"assistant_message_id":2,"finish_reason":"STOP"}
```

失败：

```text
event: error
data: {
  "run_id":"...",
  "code":"PROVIDER_UNAVAILABLE",
  "message":"model provider is temporarily unavailable",
  "retryable":true,
  "partial_output_persisted":false
}
```

取消：

```text
event: cancelled
data: {"run_id":"...","reason":"CLIENT_DISCONNECTED"}
```

兼容期可以继续输出当前 `data: {delta...}` 和 `data: [DONE]`，但内部协议必须保留
明确终态。迁移完成后再由前端版本决定是否切换命名事件。

## 8. Idempotency-Key

外部消息接口建议接受：

```text
Idempotency-Key: <UUID>
```

AI-Nexus 将其映射到 `StreamMessageRequest.request_id`。如果旧客户端没有提供，网关
可以生成并在响应头返回，但客户端自动重试时无法得到跨请求幂等保证。

## 9. 取消语义

取消来源包括：

- HTTP/SSE 客户端断开；
- gRPC Context Cancel；
- 显式 `CancelRun`；
- 服务端 Deadline；
- Chat Agent 优雅关闭。

取消应传播到 Provider。`CancelRun` 是幂等操作：已终态 Run 返回当前终态，不把
`COMPLETED` 改成 `CANCELLED`。

## 10. 分页和时间

- Conversation 使用 `page_index/page_size` 保持现有接口兼容；
- Message 建议使用游标分页，V1 Proto 先提供 `before_message_id + page_size`；
- 时间统一使用 `google.protobuf.Timestamp` 和 UTC；
- 排序规则必须稳定，必要时使用 `(created_at, id)` 作为复合游标。

## 11. Provider 错误分类

稳定业务错误码至少包含：

```text
INVALID_CREDENTIAL
PROVIDER_UNAVAILABLE
PROVIDER_RATE_LIMITED
PROVIDER_TIMEOUT
MODEL_NOT_FOUND
CONTEXT_LENGTH_EXCEEDED
CONTENT_REJECTED
CONVERSATION_BUSY
RUN_CANCELLED
INTERNAL_ERROR
```

错误 `message` 是脱敏的人类可读描述，程序逻辑只依赖 `code` 和 `retryable`。
