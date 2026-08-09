# 独立邮件服务集成：AI-Nexus 改造清单

本文记录主系统侧的实施决策。V0.1 改造已经完成；运行和验收方式见
[实施与联调手册](implementation.md)。Mail Service 的内部实现由独立仓库负责。

## 1. 当前代码替换范围

需要逐步替换：

- `proto/message.proto` 中的 `VarifyService.GetVarifyCode`；
- `internal/grpc/grpc.go` 中返回验证码的 `GetVerificationCode`；
- `user.Service.SendEmailCode` 中“外部生成验证码”的逻辑；
- `code_{email}` Redis Key；
- 注册、验证码登录、密码重置中的明文字符串比较；
- `user_verification_codes.code` 明文存储方式。

旧协议和新协议不得长期并行共用同一验证码流程，避免出现两套数据源。

## 2. 实际模块结构

```text
internal/
  verification/
    service.go          # 生成、激活、验证、消费
    store.go            # Store 接口
    model.go            # purpose、状态和传输边界
    store.go            # MySQL 状态、事件事务和一次性消费
    service.go          # 生成、发送、验证、消费与对账

  mailgateway/
    client.go           # SubmitEmail/GetEmail 与同键恢复

  mailcallback/
    server.go           # gRPC Callback Server
```

MySQL 是验证码状态和回调 Journal 的权威数据源，Redis 只负责发送冷却。这样
`event_id` 去重、`sequence` 防乱序、激活和终止可以在同一个数据库事务内完成。

## 3. 配置项

建议增加：

```text
MAIL_GRPC_ADDRESS
MAIL_GRPC_TIMEOUT
MAIL_GRPC_TLS_ENABLED
MAIL_GRPC_CA_FILE
MAIL_GRPC_CERT_FILE
MAIL_GRPC_KEY_FILE

MAIL_CALLBACK_HOST
MAIL_CALLBACK_PORT
MAIL_CALLBACK_TLS_ENABLED

VERIFICATION_CODE_LENGTH
VERIFICATION_VALID_TTL
VERIFICATION_PENDING_TTL
VERIFICATION_DISPATCH_TIMEOUT
VERIFICATION_MAX_ATTEMPTS
VERIFICATION_HMAC_SECRET
VERIFICATION_EMAIL_FINGERPRINT_SECRET
```

真实密钥只能进入本地 `.env` 或部署密钥系统，模板配置中只提供占位符。

## 4. HTTP API 调整

现有：

```text
POST /user/send-code
```

可以保留路径，但建议统一使用 JSON，并返回：

```http
202 Accepted
```

```json
{
  "code": 202,
  "message": "verification email accepted",
  "request_id": "UUID"
}
```

`purpose` 应支持稳定枚举语义：

```text
REGISTER
RESET_PASSWORD
LOGIN
```

如果为了兼容现有客户端暂时继续接收 `1/2/3`，只在 HTTP DTO 层转换，领域层和
gRPC 层不能继续传递魔法数字。

可以增加只读状态接口用于联调：

```text
GET /user/email-verifications/{request_id}/status
```

该接口不得返回验证码、摘要或完整邮箱。

## 5. 发送事务边界

建议顺序：

1. 规范化邮箱；
2. 校验 purpose 和账号条件；
3. 原子占用发送冷却窗口；
4. 生成 `request_id` 和验证码；
5. 保存 `PENDING_DISPATCH` 摘要记录；
6. 使用相同 `request_id` 调用 `SubmitEmail`；
7. `ACCEPTED/DUPLICATE` 时返回 `202`；
8. 明确业务拒绝时标记请求失败；
9. 超时等结果未知场景保留记录，并同键重试或查询状态。

不能因为第一次 RPC 超时就生成新的 `request_id`，否则可能发送两封邮件。

## 6. 回调处理

AI-Nexus 需要启动独立 gRPC Server，实现：

```text
DeliveryEventReceiverService.ReportDeliveryEvent
```

一次回调应原子完成：

1. 验证 Mail Service 身份；
2. 检查 `event_id` 是否已处理；
3. 通过 `request_id` 找到验证码记录；
4. 检查 `message_id` 和状态序列；
5. 根据状态激活或终止验证码；
6. 保存最新 `sequence`；
7. 记录 `event_id`；
8. 返回确认。

Redis Lua Script 或事务需要保证并发回调不会重复激活、覆盖新状态或延长有效期。
`PROVIDER_ACCEPTED` 和 `DELIVERED` 都可以首次激活，不能依赖两个事件严格按顺序
到达；`BOUNCED` 不激活验证码。

## 7. 验证和消费

注册、登录、密码重置不再直接执行：

```text
GET code_{email} == input_code
```

而统一调用 Verification Service：

```text
VerifyAndConsume(email, purpose, code)
```

该操作必须原子完成：

- 记录存在且状态为 `ACTIVE`；
- 当前时间没有超过 `expires_at`；
- 使用常量时间比较 HMAC 摘要；
- 错误时增加尝试次数；
- 达到上限时进入 `LOCKED`；
- 正确时进入 `CONSUMED` 或直接删除可验证记录。

业务数据库操作失败后的验证码消费语义需要单独决定。推荐先完成业务前置校验，
再通过短时业务幂等键避免“验证码已消费但数据库提交失败”造成重复副作用。

## 8. 数据库迁移

已选择 MySQL 权威状态方案：新增 `email_verification_challenges` 和
`email_delivery_events`；应用不再读写明文表 `user_verification_codes`，新建数据库也不再创建它。

数据库变更必须通过新增迁移完成，不能只修改 `configs/db.sql`，因为该初始化脚本不会
自动更新已经存在的本地数据库。

## 9. 测试范围

### 单元测试

- 安全随机验证码和 HMAC 验证；
- purpose 隔离；
- 激活、过期、锁定和一次性消费；
- 重复/乱序回调；
- Submit 超时使用同一 `request_id` 重试；
- gRPC 状态码到领域错误的映射。

### 集成测试

- Mail Service 接受任务后成功激活验证码；
- 同一 `request_id` 只生成一封邮件；
- 回调暂时失败后可以恢复；
- `PENDING_DISPATCH` 对账后可以恢复状态；
- 注册、验证码登录和密码重置全链路；
- Mail Service 不可用时 HTTP 返回可理解的暂时失败结果。

### 故障测试

- RabbitMQ 暂停和恢复；
- 邮件供应商超时、限流和认证失败；
- Mail Service 在受理后立即重启；
- AI-Nexus 在邮件发送成功前后重启；
- 回调重复、延迟和乱序。

## 10. 推荐实施顺序

1. 删除旧 `ainexus.mail.v1` Proto；
2. Mail Service 完成幂等受理、Outbox、队列和模拟供应商；
3. AI-Nexus 实现 Verification Store 和安全验证码；（已完成）
4. AI-Nexus 接入 `SubmitEmail/GetEmail`；（已完成）
5. AI-Nexus 启动 Callback Server 并实现激活；（已完成）
6. 改造注册、登录和密码重置；（已完成）
7. 完成真实 SMTP Provider、重试、熔断和 DLQ；
8. 增加对账、指标、告警和故障测试；
9. 移除旧 `VarifyService` 和旧验证码数据路径。（已完成）
