# 独立 Chat Agent 集成：需求目标

## 1. 当前情况

AI-Nexus 当前在同一个进程中实现：

- `/chat/*` HTTP 路由和 SSE；
- Provider Credential 的加密存储；
- Conversation、Message、Preset 的数据库访问；
- OpenAI、Anthropic、Gemini 和自定义 OpenAI Compatible Provider；
- 历史消息组装、模型流式调用和标题生成。

现有实现已经具备清晰的业务模块，但仍与主系统存在以下耦合：

- Chat Service 和 DAO 直接依赖 `gin.Context` 与全局数据库；
- 对话表、用户表和其他核心表位于同一数据库；
- AI-Nexus 持有 Provider Credential 的解密能力；
- 长时间 SSE 和模型请求与核心登录、用户接口运行在同一个进程；
- 标题生成使用不可恢复、不可观测的进程内 goroutine；
- 没有独立的 Agent Run、请求幂等和同一对话并发控制；
- 流式中途失败可能同时产生 error、`[DONE]` 和部分助手消息，终态不明确。

## 2. 建设目标

建设独立 Chat Agent Service，并保持客户端继续通过 AI-Nexus 的 `/chat/*` 接口
访问。

AI-Nexus 负责：

- 验证外部 JWT 和 Redis Session；
- 执行统一入口限流、CORS、RequestID 和安全中间件；
- 将 HTTP JSON 转换为内部 Unary gRPC；
- 将 gRPC Server Stream 转换为 SSE；
- 签发短时内部身份令牌；
- 在 Chat Agent 不可用时隔离故障，不影响核心服务其他接口。

Chat Agent Service 负责：

- Provider Credential 的加密、掩码、轮换和归属校验；
- Conversation、Message、Preset 和 Agent Run；
- 上下文构建和并发控制；
- Provider 路由、模型调用和流式事件；
- Token Usage、首 Token 延迟和运行终态；
- 标题生成等内部后台任务；
- 后续 Tool、Memory 和 Agent Loop 扩展。

## 3. 功能需求

### 3.1 外部接口兼容

第一阶段保留现有 HTTP 路径和主要 JSON 字段：

```text
/chat/api-keys
/chat/conversations
/chat/conversations/{conv_id}
/chat/conversations/{conv_id}/messages
/chat/presets
```

AI-Nexus Controller 由本地业务实现改为 Chat Agent gRPC 网关适配器。前端无需知道
Chat Agent 地址，也不直接持有内部凭证。

### 3.2 数据所有权

Chat Agent 独占写入：

- Provider Credential；
- Conversation；
- Message；
- Preset；
- Agent Run；
- 上下文摘要、Tool Call 等后续数据。

Chat Agent 不查询 AI-Nexus 的 `users`、Session 或钱包表。`user_id` 是由可信内部
身份传递的外部主体标识。

### 3.3 Provider Credential

- Credential 必须加密存储，响应只返回掩码；
- 密文包含密钥版本，支持后续轮换；
- 明文不得进入日志、Trace、错误详情或指标标签；
- Credential、Conversation 的关联必须属于同一用户；
- 自定义 Provider Endpoint 必须防御 SSRF、重定向到内网和 DNS Rebinding；
- 删除仍被 Conversation 使用的 Credential 时应拒绝，或采用明确的停用语义。

### 3.4 Agent Run

每次发送消息创建唯一 Agent Run：

- AI-Nexus 或客户端提供 `request_id` 作为业务幂等键；
- 同一用户下 `request_id` 唯一；
- Run 记录 Provider、模型、Token Usage、错误和时间指标；
- 同一 Conversation 同时最多一个活跃 Run；
- 用户断开连接时取消信号必须传播到 Provider；
- 已输出 Token 后不得透明重试整个模型请求；
- 部分输出必须标记为 partial，不能伪装成正常完成。

### 3.5 流式协议

`StreamMessage` 使用 gRPC Server Streaming，必须区分：

- Run 已接受；
- 文本增量；
- Usage 更新；
- 正常完成；
- 失败；
- 取消；
- 心跳。

同一 Stream 内事件包含单调递增的 `sequence`。正常完成、失败和取消只能出现一个
终态。

### 3.6 上下文管理

- V1 支持 System Prompt 和最近消息窗口；
- 必须限制单条消息长度、历史消息数量和最大请求体；
- 不得仅依赖客户端传入的 `max_tokens`；
- 后续支持按模型上下文 Token Budget 裁剪和摘要；
- 同一 Conversation 的并发请求不得读取相同旧历史后交错写入。

### 3.7 标题生成

- 首轮对话完成后可以异步生成标题；
- 标题任务必须有状态、并发限制和超时；
- Worker 执行时重新加载 Credential，不在 goroutine 中长期捕获明文 Key；
- 标题失败不影响主对话完成；
- 标题更新必须带 Conversation owner 或内部任务归属校验。

## 4. 可靠性需求

- Chat Agent 不可用时 `/chat/*` 返回 `503`，AI-Nexus 其他接口继续工作；
- gRPC 建连不得成为 AI-Nexus 整体启动的强依赖；
- Provider 超时、限流、认证失败使用稳定错误分类；
- 只在已知请求未开始或 Provider 支持幂等时进行安全重试；
- 服务关闭时停止接收新 Run，并在宽限期内取消或完成现有 Stream；
- 进程崩溃后，遗留 `STREAMING` Run 可被 Recovery Job 标记为失败并释放对话占用；
- 日志、指标和 Trace 贯通 `request_id`、`run_id`、`conversation_id`。

## 5. 安全需求

- 外部用户 JWT 只由 AI-Nexus 验证；
- 内部调用使用 mTLS 和短时、受众受限的内部 JWT；
- 内部 JWT 推荐使用 Ed25519/ES256 等非对称签名；
- Chat Agent 只信任经过服务身份认证的 AI-Nexus；
- Chat Agent 每次资源操作都校验 `resource.user_id == principal.sub`；
- 不直接信任客户端可伪造的 `X-User-ID`；
- Provider 错误必须脱敏后再返回；
- 自定义 Endpoint 默认只允许 HTTPS，并禁止访问内网和云 Metadata 地址。

## 6. 非目标

V1 不实现：

- 多 Agent 协作；
- RAG 和向量数据库；
- 长期语义记忆；
- MCP；
- Image Generation Tool；
- 流式断点续传；
- 自动跨 Provider 切换；
- 平台统一计费；
- 客户端直接访问 Chat Agent。

## 7. 验收标准

- 现有 Chat HTTP CRUD 和 SSE 流程无功能回归；
- AI-Nexus 不再直接读写 Chat 表或解密 Provider Credential；
- Chat Agent 不访问 AI-Nexus Core 数据库；
- 同一 `request_id` 不会重复调用模型或重复写入用户消息；
- 同一 Conversation 的并发第二个 Run 被明确拒绝；
- 客户端断开能够取消下游 Provider 请求；
- 正常、失败、取消三个终态互斥；
- 部分流失败不会被记录为正常完成；
- 非资源所有者无法读取、修改或删除 Conversation、Credential、Preset 和 Run；
- 自定义 Endpoint 无法访问 loopback、私网、link-local 和 Metadata 地址；
- Chat Agent 故障不会使 AI-Nexus 的用户、认证等核心接口下线；
- 数据迁移完成后，记录数量、归属关系和 Credential 解密抽检通过。
