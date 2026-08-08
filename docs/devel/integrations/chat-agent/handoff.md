# AI-Nexus Chat Agent：另一开发会话交接上下文

以下内容用于直接交给负责独立 Chat Agent Service 的开发会话。

---

## 项目背景

我要从 Go 后端 AI-Nexus 中拆出独立 Chat Agent Service。

当前 AI-Nexus 已实现：

- Provider Credential CRUD 和 AES-GCM 加密；
- Conversation、Message、Preset CRUD；
- OpenAI、Anthropic、Gemini 和自定义 OpenAI Compatible Provider；
- SSE 流式对话；
- 最近消息上下文；
- Token Usage；
- 首轮自动标题。

当前实现仍位于 AI-Nexus 单体进程，Service/DAO 依赖 Gin 和全局数据库，没有
Agent Run、请求幂等、同一对话并发控制和明确流式终态。

规范以以下文件为准：

- `requirements.md`
- `architecture.md`
- `protocol.md`
- `chat-agent-v1.proto`
- `ai-nexus-changes.md`

## 已确定的职责边界

AI-Nexus Core 负责：

- 外部 JWT + Redis Session；
- `/chat/*` HTTP 路由兼容；
- 限流、CORS、RequestID 和 Trace；
- 短时内部 JWT；
- HTTP JSON 到 Unary gRPC；
- gRPC Server Stream 到 SSE；
- Chat Agent 故障隔离。

Chat Agent Service 负责：

- Provider Credential；
- Conversation、Message、Preset；
- Agent Run 和幂等；
- 上下文构建；
- Provider 路由和流式调用；
- Token Usage 和时间指标；
- 标题后台任务；
- 后续 Agent Runtime、Tool 和 Memory。

Chat Agent 不得访问 AI-Nexus 的用户、Session、钱包或其他核心表。AI-Nexus 不得
继续读写 Chat 表或解密 Provider Credential。

## 已确定的网络边界

```text
Client
  ↓ User JWT + HTTP/SSE
AI-Nexus Core
  ↓ mTLS + short-lived internal JWT + gRPC
Chat Agent Service
  ↓ Provider API
OpenAI / Anthropic / Gemini / Compatible
```

客户端不直接访问 Chat Agent。

普通 CRUD 使用 Unary RPC。消息生成使用：

```proto
rpc StreamMessage(StreamMessageRequest) returns (stream ChatEvent);
```

## 已确定的内部身份

内部调用推荐同时使用 mTLS 和短时非对称签名 JWT：

```text
iss = ai-nexus-core
aud = ai-nexus-chat-agent
sub = user UUID
sid = login session ID
jti = internal call ID
exp = 30~60 秒
scope = chat read/write/stream
```

Chat Agent 从认证拦截器获得 Principal。业务 Request 不携带和信任 `user_id`。

## 已确定的 Stream 语义

第一个事件必须是：

```text
RunAccepted
```

它只在以下内容已经事务提交后发送：

- owner 和 Credential 已验证；
- request_id 幂等已检查；
- Conversation 已原子占用；
- UserMessage 已保存；
- AgentRun 已创建。

随后事件：

```text
ContentDelta
UsageUpdated
Heartbeat
```

终态三选一：

```text
RunCompleted
RunFailed
RunCancelled
```

同一 Run 事件有单调 `sequence`。收到首个 Delta 后不得透明重试整次 Provider 请求。

## 已确定的 Run 状态机

```text
PENDING → STREAMING → COMPLETED
                    ├→ FAILED
                    └→ CANCELLED
```

`request_id` 在用户维度唯一。同一个 Conversation 同时只允许一个活跃 Run，使用
`conversations.active_run_id` 和数据库 CAS，不依赖普通进程锁。

网络断开后 V1 不恢复 Token Stream。已收到 `RunAccepted` 时按 `run_id` 调用
`GetRun`；首事件到达前断开时按原 `request_id` 调用 `GetRun`，再结合
`ListMessages` 对账。

## 数据所有权

Chat Agent 独立数据库拥有：

```text
provider_credentials
conversations
conversation_messages
chat_presets
agent_runs
background_jobs
```

本地开发可以和 Core 共用 MySQL 容器，但使用独立 Database/Schema 和数据库账号。

现有数据需要从：

```text
user_api_keys
conversations
conversation_messages
chat_presets
```

迁移到新服务。第一阶段通过短维护窗口迁移，不做长期双写。

## 必须修复、不能照搬的旧问题

1. 当前删除 Conversation 会先按 `conv_id` 删除 Message，再校验 Conversation owner，
   存在越权删除他人消息风险；必须先锁定并验证 owner。
2. 当前 custom `base_url` 没有 SSRF 防护；必须阻止 loopback、私网、link-local、
   Metadata、恶意重定向和 DNS Rebinding。
3. 当前流中失败可能继续写 `[DONE]` 并保存 partial Message；新协议终态必须互斥。
4. 当前标题生成 goroutine 捕获明文 Credential，且不可恢复；改为持久化后台任务。
5. 当前同一 Conversation 可以并发生成；使用 `active_run_id` 控制。
6. 当前 Credential 解密错误会被忽略；新服务必须处理密钥版本和解密失败。

## V1 必须交付

1. `ainexus.chat.v1` 最终 Proto；
2. gRPC Server 和认证/恢复/日志 Interceptor；
3. 独立 MySQL Schema 和 migration；
4. Credential CRUD、加密和掩码；
5. Conversation、Message、Preset CRUD；
6. Agent Run、request_id 幂等和 Conversation 并发控制；
7. Fake Provider；
8. Server Streaming 和唯一终态；
9. CancelRun 和 Context Cancel 传播；
10. OpenAI、Anthropic、Gemini、OpenAI Compatible；
11. 自定义 Endpoint SSRF 防护；
12. Provider 超时、错误分类、并发限制和熔断；
13. 标题后台任务；
14. health、readiness、metrics、Trace 和优雅关闭；
15. 数据迁移工具、验证和回滚说明。

## V1 暂不实施

- RAG；
- 向量数据库；
- 长期记忆；
- MCP；
- 多 Agent；
- Image Tool；
- 流式断点续传；
- 平台统一计费。

代码结构应为未来能力预留接口，但不能提前堆入未验证复杂度。

## 建议开发顺序

```text
协议和状态机
  ↓
数据库和 Repository
  ↓
内部认证
  ↓
Fake Provider + Agent Run
  ↓
gRPC Server Streaming
  ↓
Credential/Conversation/Preset CRUD
  ↓
真实 Providers
  ↓
SSRF、超时、熔断和限流
  ↓
标题后台任务
  ↓
AI-Nexus 联调
  ↓
数据迁移和切流
```

## Fake Provider 必测场景

- 正常逐 Token 输出；
- 首 Token 前失败；
- 输出部分 Token 后失败；
- Provider 超时；
- 客户端断开；
- 显式 CancelRun；
- 重复 request_id；
- 同一 Conversation 并发；
- 服务优雅关闭；
- 进程崩溃后的遗留 Run Recovery。

## 安全底线

- Credential Secret 不进入日志、Trace、指标和响应；
- Chat Agent 每个查询和修改都带 owner 条件；
- 内部 JWT 校验 `iss/aud/sub/exp/scope`；
- 生产环境使用 TLS/mTLS；
- Provider 原始错误脱敏；
- 自定义 Endpoint 使用受控解析、DNS 和 Dialer；
- Tool 扩展未来必须执行独立权限校验，不能直接信任模型输出。

## 交付给 AI-Nexus 的内容

- 最终 `.proto` 和生成命令；
- contracts 版本固定方式；
- 本地启动和 Docker Compose 接入说明；
- gRPC TLS/mTLS 与内部 JWT 公钥配置；
- Error Code 表；
- SSE 映射样例；
- Migration、数据校验和回滚命令；
- Fake Provider 使用方法；
- 端到端测试步骤；
- health/readiness 和可观测指标定义。

---

如果实现需要调整 `chat-agent-v1.proto`，必须先说明修改原因、兼容性影响和 AI-Nexus
网关改造成本，不能单方面改变已约定字段语义。
