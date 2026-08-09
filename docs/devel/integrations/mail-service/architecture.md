# 独立邮件服务集成：架构与核心流程

## 1. 系统边界

```text
客户端
  │
  │ HTTP：申请验证码
  ▼
AI-Nexus
  ├── 业务规则、限流
  ├── 生成验证码
  ├── MySQL 验证码状态与回调 Journal
  ├── Redis 发送冷却窗口
  ├── 验证和一次性消费
  └── gRPC Callback Server
          │
          │ gRPC DeliveryService.SubmitEmail
          ▼
Mail Service
  ├── 幂等受理、任务数据库
  ├── Transactional Outbox
  ├── RabbitMQ
  ├── 模板渲染
  ├── SMTP/供应商适配
  ├── 重试、熔断、DLQ
  └── Callback Outbox
          │
          └── gRPC ReportDeliveryEvent ──→ AI-Nexus
```

两个系统只通过协议交换命令和状态，不共享 Redis、数据库表或内部消息队列。

## 2. 验证码发送流程

```text
1. 客户端调用 AI-Nexus 申请验证码
2. AI-Nexus 校验邮箱、purpose、账号状态和发送频率
3. AI-Nexus 生成 request_id 和验证码
4. AI-Nexus 保存验证码 HMAC 摘要，状态为 PENDING_DISPATCH
5. AI-Nexus 调用 DeliveryService.SubmitEmail
6. Mail Service 以 request_id 幂等持久化任务和 Outbox
7. Mail Service 返回 ACCEPTED 或 DUPLICATE
8. AI-Nexus 向客户端返回 202 Accepted 和 request_id
9. Mail Service 异步入队、渲染并调用供应商
10. 供应商接受邮件后，Mail Service 持久化状态和回调 Outbox
11. Mail Service 调用 ReportDeliveryEvent(PROVIDER_ACCEPTED)
12. AI-Nexus 幂等激活验证码并设置有效期
```

AI-Nexus 在步骤 4 先建立 `PENDING_DISPATCH`，避免邮件已发送但回调暂时丢失时
完全找不到该验证码。回调负责激活记录，而不是第一次创建记录。

## 3. AI-Nexus 验证码状态机

```text
PENDING_DISPATCH
  ├── PROVIDER_ACCEPTED/DELIVERED → ACTIVE
  ├── PERMANENTLY_FAILED ─→ DELIVERY_FAILED
  ├── DEAD_LETTERED ──────→ DELIVERY_FAILED
  └── 等待超时 ──────────→ EXPIRED

ACTIVE
  ├── 验证成功 ──────────→ CONSUMED
  ├── 错误次数耗尽 ──────→ LOCKED
  └── 有效期结束 ────────→ EXPIRED
```

`ACTIVE` 只允许设置一次 `active_at` 和 `expires_at`。`DELIVERED` 蕴含供应商
此前已经接受邮件，因此即使它因回调乱序先到，也可以激活验证码。后续肯定投递
状态只能更新投递观测状态，不能重新计算验证码有效期。

## 4. Mail Service 投递状态

```text
QUEUED
  ↓
SENDING
  ├── 临时失败 ──→ RETRYING ──→ SENDING
  ├── 供应商接受 → PROVIDER_ACCEPTED
  ├── 永久失败 ──→ PERMANENTLY_FAILED
  └── 重试耗尽 ──→ DEAD_LETTERED

PROVIDER_ACCEPTED
  ├── 供应商送达事件 → DELIVERED
  └── 供应商退信事件 → BOUNCED
```

SMTP 或供应商返回成功通常只表示“供应商接受了邮件”，不等价于用户已经在收件箱
看到邮件。因此 V1 使用 `PROVIDER_ACCEPTED`，不笼统命名为 `SENT_SUCCESS`。

## 5. 本地数据模型

为避免在 Redis Key 中暴露邮箱，先计算：

```text
email_fingerprint = HMAC-SHA256(key_fingerprint_secret, normalized_email)
```

MySQL `email_verification_challenges` 保存：

```text
request_id、message_id、email_fingerprint、purpose、code_digest、state、
failed_attempts、latest_delivery_status、latest_sequence 与生命周期时间
```

MySQL `email_delivery_events` 以 `event_id` 为主键保存回调 Journal，并按
`message_id + sequence` 判断乱序。事件去重、序列校验和验证码状态转换在同一个事务内完成。

Redis 只保存发送冷却窗口：

```text
verification:cooldown:{purpose}:{email_fingerprint}
```

验证码摘要建议为：

```text
HMAC-SHA256(
  verification_secret,
  request_id || 0x00 || code
)
```

仅使用普通 SHA-256 不安全，因为六位验证码的搜索空间很小。

## 6. 时间语义

- `requested_at`：AI-Nexus 创建请求的时间；
- `dispatch_deadline`：邮件允许开始投递的最晚时间；
- `occurred_at`：状态在 Mail Service 中实际发生的时间；
- `active_at`：AI-Nexus 首次接受肯定投递状态的业务激活时间；
- `expires_at`：`active_at + valid_for_seconds`。

Mail Service 在 `dispatch_deadline` 之后不得继续发送验证码邮件。推荐初始值：

```text
dispatch_deadline = requested_at + 2 分钟
valid_for_seconds = 5 分钟
PENDING_DISPATCH 保留时间 = 15 分钟
```

具体时间最终应进入配置，不硬编码在协议实现中。

## 7. 异常和恢复

### SubmitEmail 超时

超时不能证明任务没有被 Mail Service 接收。AI-Nexus 必须使用同一个
`request_id` 重试，Mail Service 返回原任务，不得重复发送。

### 回调丢失

Mail Service 持久化 Callback Outbox 并重试。AI-Nexus 同时可以对长时间停留在
`PENDING_DISPATCH` 的记录调用 `GetEmail` 对账。

### AI-Nexus 暂时不可用

邮件发送不受影响，Callback Worker 按退避策略重试。超过策略上限后进入回调 DLQ，
不得静默丢弃。

### 邮件供应商不可用

Mail Service 的供应商适配层触发熔断，已有任务保留在队列或延迟队列。熔断恢复后
继续处理，但超过 `dispatch_deadline` 的验证码邮件必须终止。

## 8. 一致性取舍

MySQL、Redis、RabbitMQ 和远程 gRPC 之间不存在一个统一事务。本设计不追求虚假的
跨系统强事务，而使用以下组合达到可恢复的一致性：

- 本地事务 + Transactional Outbox；
- 至少一次发送；
- `request_id` 和 `event_id` 幂等；
- 状态序列号防乱序；
- 状态查询定期对账；
- 有界超时与过期清理。
