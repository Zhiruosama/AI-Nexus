# 独立 Chat Agent 集成：AI-Nexus 改造与迁移清单

本文描述当前 AI-Nexus 仓库需要实施的改造。独立 Chat Agent Service 的内部实现由
新仓库负责。

## 1. 当前模块范围

当前 Chat 代码主要位于：

```text
internal/routes/chat
internal/controller/chat
internal/service/chat
internal/dao/chat
internal/domain/do/chat
internal/domain/dto/chat
internal/domain/query/chat
internal/domain/vo/chat
internal/pkg/chat
```

相关表：

```text
user_api_keys
conversations
conversation_messages
chat_presets
```

切换后，AI-Nexus 只保留 Route、HTTP DTO 兼容和网关 Controller。Service、DAO、
Provider 以及 Chat 表访问最终迁出。

## 2. 必须先修正的当前问题

### 2.1 删除对话的 owner 校验顺序

当前 `DeleteConversation` 在事务中先按 `conv_id` 删除 Message，随后才按
`conv_id + user_uuid` 删除 Conversation。知道他人 `conv_id` 的用户可能先删除他人
消息，即使 Conversation 删除最终因 owner 不匹配而失败。

迁移实现必须在同一事务中：

1. 按 `conv_id + user_uuid` 查询并锁定 Conversation；
2. 不存在时返回 `NOT_FOUND`；
3. 删除该 Conversation 的 Message、Run 和后台任务；
4. 删除 Conversation；
5. 提交事务。

不能把当前 DAO 原样复制到新服务。

### 2.2 Custom Provider SSRF

当前只判断自定义 `base_url` 是否为空，没有防御用户访问内网、Redis、MySQL 或云
Metadata 地址。新 Chat Agent 必须实施协议文档中的 Endpoint 校验和受控 Dialer。

### 2.3 流式终态不明确

当前 Chunk 出错后可能写入 error，再写 `[DONE]`，并保存部分 Assistant Message，
但调用整体不一定返回错误。迁移时必须使用互斥的 `Completed/Failed/Cancelled`。

### 2.4 标题 goroutine

当前标题生成 goroutine 捕获明文 Credential，无法恢复或优雅关闭。新服务使用内部
持久化 Job，并在执行时重新读取 Credential。

### 2.5 Conversation 并发

当前同一 Conversation 可以同时运行多个 Stream，导致历史快照和消息顺序交错。
新服务通过 `active_run_id` 和 CAS 控制一个活跃 Run。

### 2.6 Credential 解密错误

当前 Credential 列表掩码生成忽略解密错误。新服务需要记录脱敏告警并返回稳定内部
错误，支持带版本密文和密钥轮换。

## 3. AI-Nexus 保留的职责

AI-Nexus 保留：

- `/chat/*` HTTP 路由；
- 外部 JWT + Redis Session 认证；
- HTTP DTO 校验和兼容映射；
- RequestID、Trace、限流和 CORS；
- 内部身份令牌签发；
- Unary gRPC 调用；
- gRPC Stream 到 SSE 的转换；
- gRPC 错误到 HTTP/SSE 的稳定映射。

AI-Nexus 不再：

- 查询 Chat 数据库；
- 加密或解密 Provider Credential；
- 组装历史消息；
- 直接调用模型 Provider；
- 保存 Conversation 或 Message；
- 启动标题生成任务。

## 4. 建议新增模块

```text
internal/
  chatgateway/
    client.go             # gRPC client lifecycle
    credentials.go        # Credential Unary adapter
    conversations.go      # Conversation Unary adapter
    presets.go            # Preset Unary adapter
    stream.go             # Server Stream wrapper
    errors.go             # gRPC → domain/http mapping

  internalauth/
    issuer.go             # internal JWT issuer
    claims.go
```

现有 `internal/controller/chat` 改为依赖接口：

```go
type ChatGateway interface {
    CreateConversation(ctx context.Context, ...) (..., error)
    StreamMessage(ctx context.Context, ...) (EventStream, error)
}
```

Controller 不直接依赖生成的 gRPC Client，便于 Fake Client 单元测试。

## 5. 配置项

建议增加：

```text
CHAT_AGENT_ENABLED
CHAT_AGENT_GRPC_ADDRESS
CHAT_AGENT_GRPC_CONNECT_TIMEOUT
CHAT_AGENT_GRPC_UNARY_TIMEOUT
CHAT_AGENT_GRPC_STREAM_TIMEOUT
CHAT_AGENT_GRPC_TLS_ENABLED
CHAT_AGENT_GRPC_CA_FILE
CHAT_AGENT_GRPC_CERT_FILE
CHAT_AGENT_GRPC_KEY_FILE

CHAT_INTERNAL_JWT_ISSUER
CHAT_INTERNAL_JWT_AUDIENCE
CHAT_INTERNAL_JWT_PRIVATE_KEY_FILE
CHAT_INTERNAL_JWT_TTL
```

`CHAT_AGENT_GRPC_STREAM_TIMEOUT` 应大于普通 Unary Timeout，并受到服务端最大 Run 时间
约束。真实私钥不得写入仓库配置。

## 6. 内部身份签发

AuthMiddleware 已产生：

```text
Principal{UserID, SessionID}
```

Chat Controller 调用 gRPC 前，由 AI-Nexus 签发短时内部 JWT：

```text
iss = ai-nexus-core
aud = ai-nexus-chat-agent
sub = Principal.UserID
sid = Principal.SessionID
jti = 当前内部调用 ID
exp = now + 30~60 秒
scope = 对应 chat scope
```

每次调用通过 Metadata 传递，不将用户提供的 `Authorization` 原样转发。内部 JWT
私钥由 AI-Nexus 持有，Chat Agent 只持有验证公钥。

## 7. HTTP 到 gRPC 映射

### 7.1 Credential

外部仍可保留 `/chat/api-keys` 命名，内部映射为 `ProviderCredential`。当前：

```text
openai     → PROVIDER_TYPE_OPENAI
anthropic  → PROVIDER_TYPE_ANTHROPIC
gemini     → PROVIDER_TYPE_GEMINI
custom     → PROVIDER_TYPE_OPENAI_COMPATIBLE
```

HTTP 响应继续返回掩码，不返回 Secret。

### 7.2 Conversation Detail

当前详情一次返回 Conversation 和全部 Message。内部协议拆为：

```text
GetConversation
+
ListMessages
```

兼容期 AI-Nexus 可以组合结果，但必须限制最大消息数量。后续外部 API 应改为 Message
游标分页，避免超大对话一次加载。

### 7.3 Stream Message

AI-Nexus：

1. 从 `Idempotency-Key` 获取 `request_id`，缺失时生成；
2. 打开 `AgentService.StreamMessage` 并读取首事件；
3. 首次 `Recv` 返回 gRPC Status 时，在尚未提交响应前映射为普通 HTTP 错误；
4. 首事件为 `RunAccepted` 后再设置并提交 SSE Header；
5. 将 ChatEvent 映射为 SSE；
6. HTTP Context 取消时立即取消 gRPC Context；
7. 收到唯一终态后正常结束；
8. 传输异常时输出脱敏 error，并允许客户端通过 `run_id` 或 `request_id` 对账。

反向代理需要关闭 SSE Buffer，并配置大于最大 Run 时间的 Idle Timeout。

## 8. 故障隔离和健康检查

- AI-Nexus 启动时不要求 Chat Agent 必须在线；
- Core liveness 不包含 Chat Agent 状态；
- Core readiness 可以暴露 Chat Agent 为 degraded dependency，但不能让用户认证等接口
  一起停止接流量；
- Chat Agent 不可用时，仅 `/chat/*` 返回 `503`；
- gRPC Client 使用有界 Backoff，不能高速重连刷日志；
- 熔断只作用于 Chat Gateway 调用，不影响其他模块。

## 9. 数据库迁移

目标是将以下表迁往 Chat Agent 独立数据库：

```text
user_api_keys
conversations
conversation_messages
chat_presets
```

并新增：

```text
agent_runs
background_jobs
```

推荐短维护窗口迁移，不做长期双写：

1. 部署 Chat Agent 和新 Schema，但不接正式流量；
2. 使用 Fake/测试用户完成 gRPC 验证；
3. 暂停 `/chat/*` 写操作；
4. 导出并导入四张旧表；
5. 增加新字段、索引和 Agent Run 表；
6. 核对每张表行数、主键范围、归属关系和抽样数据；
7. 使用旧 Encryption Key 抽样验证 Credential 可解密；
8. 开启 `CHAT_AGENT_ENABLED`；
9. 完成 CRUD、SSE、取消和错误回归；
10. 旧表进入只读观察期；
11. 稳定后移除旧代码和旧表。

需要正式 migration，不能只修改 `configs/db.sql`，因为初始化脚本不会升级已有数据库。

## 10. 切换和回滚

Feature Flag：

```text
CHAT_AGENT_ENABLED=true/false
```

切换前旧路径保持可构建，但一旦新服务开始产生新 Conversation/Message，简单切回旧
数据库会丢失新数据。因此：

- 灰度前完整备份；
- 切换窗口内禁止两边同时写；
- 回滚需要反向迁移切换后新增数据；
- 观察期不等于可以长期双写；
- 稳定后尽快删除旧实现，避免两套逻辑漂移。

## 11. 测试范围

### Controller/Gateway 单元测试

- HTTP DTO 到 Proto 的字段映射；
- Provider 枚举映射；
- gRPC Status 到 HTTP 状态映射；
- ChatEvent 到 SSE 映射；
- HTTP Cancel 传播到 gRPC；
- Secret 不进入响应或错误。

### 双服务集成测试

- Credential CRUD；
- Conversation 和 Preset CRUD；
- Detail 组合和 Message 分页；
- 正常 Token Stream；
- 首 Token 前失败；
- 部分输出后失败；
- 客户端主动取消；
- 同 request_id 重试；
- 同一 Conversation 并发；
- Chat Agent 停机时 Core 其他接口可用。

### 数据迁移验证

- 表行数和外键式引用一致；
- Conversation 均能找到 owner 和 Credential；
- Message 均能找到 Conversation；
- Credential 抽样解密和掩码一致；
- 旧接口与新接口关键响应对比；
- 迁移日志不输出 Secret。

## 12. 推荐实施顺序

1. 冻结 `ainexus.chat.v1` Proto 和内部认证协议；
2. Chat Agent 使用独立数据库和 Fake Provider 完成闭环；
3. AI-Nexus 增加 Chat Gateway Client 和 Fake gRPC 集成测试；
4. 迁移 OpenAI/Anthropic/Gemini/Compatible Provider；
5. 完成 mTLS、内部 JWT、SSRF 防护和 Provider 错误分类；
6. 完成数据迁移工具和校验脚本；
7. 维护窗口切流；
8. 观察稳定后移除 AI-Nexus 本地 Chat Service/DAO/Provider；
9. 再开始 Tool Registry、Memory 和 Image Tool。
