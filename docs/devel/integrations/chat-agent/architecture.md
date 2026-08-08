# 独立 Chat Agent 集成：架构、状态机与数据设计

## 1. 目标架构

```text
Client
  │ HTTP JSON / SSE + User JWT
  ▼
AI-Nexus Core
  ├── AuthMiddleware：JWT + Redis Session
  ├── RateLimit / RequestID / CORS
  ├── HTTP DTO ↔ Proto
  ├── gRPC Stream → SSE
  └── Internal Identity Issuer
          │
          │ mTLS + Internal JWT + gRPC
          ▼
Chat Agent Service
  ├── Credential Application Service
  ├── Conversation Application Service
  ├── Agent Run Service
  ├── Context Builder
  ├── Provider Registry
  ├── Agent Runtime
  ├── Background Jobs
  └── Chat-owned MySQL
          │
          ├── OpenAI
          ├── Anthropic
          ├── Gemini
          └── OpenAI Compatible Endpoint
```

AI-Nexus 是外部 API Gateway，不代理或读取 Chat 数据库。Chat Agent 是 Chat 领域的
唯一数据写入者。

## 2. 普通 CRUD 流程

```text
Client 请求 /chat/conversations
  ↓
AI-Nexus 验证用户登录态
  ↓
AI-Nexus 签发 aud=chat-agent 的短时内部 JWT
  ↓
Unary gRPC + RequestID
  ↓
Chat Agent 验证 mTLS、JWT iss/aud/exp/sub
  ↓
按照 principal.sub 查询或修改资源
  ↓
返回 Proto Response
  ↓
AI-Nexus 映射为兼容 HTTP JSON
```

资源不存在和资源不属于当前用户，对外均可映射为 `NOT_FOUND`，避免泄露资源是否
真实存在。

## 3. 流式消息流程

```text
1. Client POST /chat/conversations/{id}/messages
2. AI-Nexus 验证用户 JWT/Session，读取 Idempotency-Key
3. AI-Nexus 调用 StreamMessage(request_id, conversation_id, content, options)
4. Chat Agent 验证内部身份和参数
5. 在数据库事务中：
   - 校验 Conversation 和 Credential 归属
   - 按 request_id 检查幂等
   - 原子占用 Conversation.active_run_id
   - 创建 UserMessage
   - 创建 AgentRun(PENDING)
6. AgentRun 转为 STREAMING，发送 RunAccepted
7. 加载历史并构造 Provider Request
8. Provider 返回 Token，Chat Agent 发送 ContentDelta
9. 结束时在事务中：
   - 保存完整或部分 AssistantMessage
   - 更新 Usage 和 Run 终态
   - 清除 Conversation.active_run_id
10. 发送唯一终态事件
11. AI-Nexus 将 ChatEvent 转换为 SSE
```

如果 gRPC 在 `RunAccepted` 前失败，使用标准 gRPC Status。如果已经发送
`RunAccepted`，后续业务失败必须通过 `RunFailed` 或 `RunCancelled` 事件表达。

## 4. Agent Run 状态机

```text
PENDING
  ├── Provider 开始请求 ─→ STREAMING
  ├── 前置校验失败 ─────→ FAILED
  └── 用户取消 ─────────→ CANCELLED

STREAMING
  ├── 正常结束 ─────────→ COMPLETED
  ├── Provider/网络失败 ─→ FAILED
  └── Context Cancel ────→ CANCELLED
```

终态：

```text
COMPLETED / FAILED / CANCELLED
```

规则：

- 终态只能写入一次；
- 状态迁移使用带旧状态条件的 CAS；
- 清除 `active_run_id` 时必须带当前 `run_id` 条件；
- 已经产生内容的失败或取消可以保存 partial AssistantMessage；
- partial Message 不得计作完整回答；
- 进程崩溃导致长期 `STREAMING` 时，由 Recovery Job 转为 `FAILED`。

## 5. Conversation 并发控制

建议在 `conversations` 增加：

```text
active_run_id
```

占用：

```sql
UPDATE conversations
SET active_run_id = :run_id
WHERE conv_id = :conv_id
  AND user_uuid = :user_id
  AND active_run_id IS NULL;
```

影响行数为 0 时，表示 Conversation 不存在、无权访问或已有活跃 Run。服务在已验证
owner 后向调用方返回 `CONVERSATION_BUSY`。

释放：

```sql
UPDATE conversations
SET active_run_id = NULL
WHERE conv_id = :conv_id
  AND active_run_id = :run_id;
```

不能无条件清空，否则旧 Run 的延迟清理可能误释放一个新 Run。

## 6. 请求幂等

`agent_runs` 建立：

```text
UNIQUE(user_uuid, request_id)
```

同一 `request_id`：

- 已 `COMPLETED`：不重新调用 Provider，调用方可查询已有 Run 和消息；
- `PENDING/STREAMING`：返回 `ALREADY_RUNNING`；
- `FAILED/CANCELLED`：不自动重跑，客户端使用新的 `request_id` 创建新 Run。

V1 不支持从指定 `sequence` 恢复 Stream。网络断开后通过 `GetRun` 和 `ListMessages`
确认终态：收到 `RunAccepted` 后可按 `run_id` 查询，在首事件到达前断开则按调用方
原本持有的 `request_id` 查询。

## 7. 数据所有权和建议表结构

### 7.1 Provider Credentials

在现有字段基础上增加：

```text
key_version
endpoint_fingerprint
last_used_at
disabled_reason
```

密文采用：

```text
version || nonce || AES-256-GCM ciphertext || tag
```

V1 可以迁移现有密文并暂时复用旧密钥，稳定后再使用带版本的新密钥重新加密。

### 7.2 Conversations

建议字段：

```text
conv_id
user_uuid
credential_id
title
model
system_prompt
active_run_id
created_at
updated_at
```

### 7.3 Messages

建议增加：

```text
run_id
sequence
is_partial
finish_reason
```

`role` 为 V1 保留：

```text
system / user / assistant
```

未来 Tool Calling 时再扩展 `tool` 及结构化 Content Part，不能把 Tool JSON 强行塞进
普通文本字段后长期兼容。

### 7.4 Agent Runs

建议字段：

```text
id
run_id                UNIQUE
request_id
user_uuid
conv_id
user_message_id
assistant_message_id
status
provider
model
finish_reason
prompt_tokens
completion_tokens
error_code
error_message_sanitized
partial_output
started_at
first_token_at
completed_at
created_at
updated_at

UNIQUE(user_uuid, request_id)
INDEX(conv_id, created_at)
INDEX(status, updated_at)
```

## 8. 流式持久化

不为每个 Token 执行数据库事务。推荐：

```text
开始事务：Run + UserMessage
  ↓
流式过程：内存 Buffer + gRPC Delta
  ↓
结束事务：AssistantMessage + Usage + Run 终态
```

如果需要更强恢复能力，可以后续增加按时间或字符数写入检查点，但这不属于 V1。

## 9. Provider 重试和熔断

- 连接建立前的明确临时失败可以有限重试；
- 请求是否已被供应商受理结果不明确时，不盲目重试；
- 收到第一个 Delta 后绝不透明重放整次请求；
- 认证失败、参数错误、内容拒绝不可重试；
- 429 按 `Retry-After` 和用户等待预算决定是否返回失败；
- 熔断按 Provider 和 Endpoint 隔离，不能一个自定义 Endpoint 拖垮全部 Provider；
- 并发限制同时按用户、Credential 和 Provider 执行。

## 10. 自定义 Endpoint SSRF 防护

创建和使用 Credential 时均需检查：

- 默认只允许 `https`；
- 禁止 URL UserInfo；
- 禁止 loopback、private、link-local、multicast 和未指定地址；
- 禁止云 Metadata 地址；
- DNS 解析后检查所有 IP；
- 自定义 Dialer 连接时再次校验实际目标 IP；
- 重定向目标重新执行完整校验；
- 限制端口和响应体；
- 本地开发允许 localhost 必须由服务端配置显式开启。

只在保存 Credential 时检查字符串不足以防止 DNS Rebinding。

## 11. 标题生成

主 Run 完成后创建内部标题任务：

```text
Run COMPLETED
  ↓
Conversation 仍为默认标题
  ↓
创建 Durable Title Job
  ↓
Worker 重新加载 Credential
  ↓
调用 Provider，限制超时和输出长度
  ↓
带 owner/conv_id 条件更新标题
```

标题生成失败只影响标题，不改变主 Run 终态。

## 12. 后续 Agent 扩展

V1 内部接口预留：

```go
type ModelProvider interface {
    Stream(ctx context.Context, req ModelRequest) (ModelStream, error)
}

type Tool interface {
    Name() string
    Definition() ToolDefinition
    Execute(ctx context.Context, input []byte) (ToolResult, error)
}

type Memory interface {
    BuildContext(ctx context.Context, conversationID string) ([]Message, error)
    Record(ctx context.Context, event MemoryEvent) error
}
```

后续可以将 Image Generation Service 注册成工具，但 Tool 不负责认证、计费幂等或
任务可靠性，这些仍由对应领域服务保证。
