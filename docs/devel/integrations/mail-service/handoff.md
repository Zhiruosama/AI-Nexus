# AI-Nexus Mail Service：另一开发会话交接上下文

以下内容用于直接交给负责独立 Mail Service 的开发会话。

---

## 项目背景

我要为 AI-Nexus 开发一个独立的 Mail Service。

AI-Nexus 是 Go 后端。当前旧邮件协议为
`VarifyService.GetVarifyCode(email)`，由外部服务生成验证码并将明文验证码返回。
AI-Nexus 随后写入 `user_verification_codes`，而注册、验证码登录和密码重置又从
Redis `code_{email}` 读取验证码。这个设计职责混乱并可能依赖共享 Redis，将被
整体替换。

协议规范以本目录中的以下文件为准：

- `requirements.md`
- `architecture.md`
- `protocol.md`
- `mail-v1.proto`
- `ai-nexus-changes.md`

## 已确定的系统边界

AI-Nexus 拥有验证码业务：

- 校验邮箱、用途和账号状态；
- 使用 `crypto/rand` 生成验证码；
- 保存验证码 HMAC 摘要；
- 管理 `PENDING_DISPATCH`、`ACTIVE`、`CONSUMED`、`LOCKED`、
  `EXPIRED`、`DELIVERY_FAILED`；
- 验证并原子消费验证码；
- 接收邮件状态回调。

Mail Service 拥有邮件投递：

- gRPC 幂等受理；
- 数据库持久化；
- Transactional Outbox；
- RabbitMQ 异步处理；
- 模板和国际化；
- SMTP/第三方 Provider Adapter；
- 超时、重试、指数退避、熔断和 DLQ；
- Callback Outbox 和状态查询；
- 指标、日志和链路追踪。

Mail Service 不得访问 AI-Nexus 的 Redis 或 MySQL。

## 已确定的调用流程

```text
AI-Nexus 生成 request_id 和验证码
  ↓
AI-Nexus 保存 PENDING_DISPATCH 的验证码摘要
  ↓
SubmitEmail(request_id, template_key, typed payload)
  ↓
Mail Service 在本地事务中保存任务和 Outbox
  ↓
返回 ACCEPTED 或 DUPLICATE
  ↓
Mail Service 异步投递
  ↓
供应商接受邮件
  ↓
ReportDelivery(PROVIDER_ACCEPTED)
  ↓
AI-Nexus 激活验证码并开始计算有效期
```

回调负责激活已有的 `PENDING_DISPATCH` 记录，不负责第一次创建验证码。

## 已确定的协议语义

- Protobuf package：`ainexus.mail.v1`；
- `request_id`：邮件提交幂等键，由 AI-Nexus 生成；
- `message_id`：邮件任务 ID，由 Mail Service 生成；
- `event_id`：回调事件幂等键；
- `sequence`：同一邮件状态的单调序列号；
- `SubmitEmail` 返回任务是否 `ACCEPTED/DUPLICATE`；
- `GetEmailStatus` 用于超时确认和回调对账；
- AI-Nexus 实现 `MailDeliveryCallbackService.ReportDelivery`；
- 不传任意 HTML，不传动态 callback URL；
- 使用 `template_key + locale + typed payload`；
- purpose 包含 `REGISTER`、`RESET_PASSWORD`、`LOGIN`；
- `dispatch_deadline` 之后禁止发送验证码邮件；
- `PROVIDER_ACCEPTED` 只表示供应商接受，不保证用户最终收到；
- `PROVIDER_ACCEPTED` 或 `DELIVERED` 都可以幂等激活验证码，不能依赖回调顺序；
- `BOUNCED` 不激活验证码；
- 调用语义是“至少一次 + 幂等”，不是虚假的严格一次。

## Mail Service 第一阶段目标

请先完成设计，不要直接跳过协议开始堆业务代码：

1. `requirements.md`；
2. `architecture.md`；
3. 使用提供的 `mail-v1.proto` 并审查兼容性；
4. 邮件任务、Outbox、回调 Outbox 和幂等表的数据模型；
5. 投递状态机和错误分类；
6. SMTP Provider 接口；
7. 重试、熔断和 DLQ 策略；
8. 配置、密钥和 mTLS 方案；
9. 可观测性方案；
10. 分阶段开发计划。

设计确认后再实施：

```text
gRPC 接入
→ 幂等持久化
→ Transactional Outbox
→ RabbitMQ
→ Worker
→ 模板渲染
→ SMTP Provider
→ 状态持久化
→ Callback Outbox
→ gRPC 回调
```

第一阶段应提供一个可控的 Fake Provider，支持模拟：

- 成功；
- 临时失败后成功；
- 永久失败；
- 超时；
- 限流；
- 重复处理。

真实 SMTP 接入应建立在状态机和可靠性测试已经通过之后。

## 安全底线

- 生产使用 TLS，推荐 mTLS；
- 验证码不得写入日志、错误详情和 Trace 标签；
- 异步任务中的验证码变量必须加密存储；
- 终态后及时清理验证码明文；
- 相同 `request_id` 和不同 Payload 必须拒绝；
- 模板由 Mail Service 管理；
- 错误消息必须脱敏；
- 服务关闭时需要优雅停止消费并正确归还未确认消息。

## 交付要求

完成后需要给 AI-Nexus 侧提供：

- 最终版 `.proto`；
- 生成客户端所需命令和版本；
- 本地启动方式；
- 健康检查和 readiness 定义；
- TLS/mTLS 配置说明；
- 错误码表；
- 状态机说明；
- 联调用 Fake Provider；
- 一套端到端测试步骤。

---

如果新系统设计与 `mail-v1.proto` 发生冲突，应先记录原因和兼容性影响，再修改协议，
不能单方面改变字段语义。
