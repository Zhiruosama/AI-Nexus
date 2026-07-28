# 独立邮件服务集成：gRPC V1 协议约定

实际消息定义见 [mail-v1.proto](mail-v1.proto)。

## 1. 版本与所有权

- Protobuf package 固定为 `ainexus.mail.v1`；
- V1 字段编号发布后不得复用；
- 删除字段时使用 `reserved` 保留编号和名称；
- Mail Service 实现 `MailDispatchService`；
- AI-Nexus 实现 `MailDeliveryCallbackService`；
- 两个仓库必须从同一份版本化 Proto 生成代码，不能分别手写同名结构。

Proto 最终可以放入独立 contracts 仓库，也可以先由一个仓库作为唯一规范源。无论
采用哪种方式，都必须通过版本或提交号固定依赖，禁止复制后各自修改。

## 2. SubmitEmail 语义

`SubmitEmail` 表示“可靠受理邮件任务”，不表示邮件已经发送。

Mail Service 只有完成以下操作后才能返回 `ACCEPTED`：

1. 校验请求；
2. 使用 `request_id` 检查幂等；
3. 在本地事务中保存邮件任务和待发布 Outbox；
4. 提交事务。

相同 `request_id` 和相同业务载荷再次调用时返回 `DUPLICATE` 以及原
`message_id`。相同 `request_id` 携带不同载荷时应返回
`ALREADY_EXISTS`，不得覆盖旧任务。

## 3. GetEmailStatus 语义

该接口用于：

- gRPC Submit 超时后的结果确认；
- 回调长时间未到达时的主动对账；
- 运维诊断。

它不是高频轮询接口，应配置调用频率限制和超时。

## 4. ReportDelivery 语义

- `event_id` 是单个回调事件的幂等键；
- `sequence` 是同一 `message_id` 下单调递增的状态序号；
- 重复 `event_id` 必须返回成功确认；
- 小于等于已处理 `sequence` 的旧事件不得覆盖当前状态；
- 终态事件不得被非终态事件覆盖；
- `PROVIDER_ACCEPTED` 或 `DELIVERED` 首次到达时幂等激活验证码；
- 后续状态不得延长验证码有效期。

`DELIVERED` 蕴含邮件此前已经被供应商接受。允许它直接激活是为了避免
`PROVIDER_ACCEPTED` 回调丢失或乱序时验证码永远停留在 `PENDING_DISPATCH`。
`BOUNCED` 不属于肯定投递状态，不激活验证码。

## 5. 标准 gRPC 错误

| Code | 使用场景 | 客户端处理 |
| --- | --- | --- |
| `INVALID_ARGUMENT` | 邮箱、模板、时间或 Payload 不合法 | 不重试 |
| `UNAUTHENTICATED` | 服务身份认证失败 | 不重试并告警 |
| `PERMISSION_DENIED` | 调用方无权使用模板或接口 | 不重试并告警 |
| `ALREADY_EXISTS` | 相同 `request_id` 对应不同载荷 | 不重试并告警 |
| `RESOURCE_EXHAUSTED` | 服务级限流或容量保护 | 按 Retry-After 退避 |
| `NOT_FOUND` | 查询不存在的 `request_id` | 停止对账或人工排查 |
| `UNAVAILABLE` | 服务暂时不可用 | 使用相同幂等键重试 |
| `DEADLINE_EXCEEDED` | 调用超时，结果未知 | 查询状态或同键重试 |
| `INTERNAL` | 未分类服务错误 | 有界重试并告警 |

业务供应商失败通过投递状态和 `FailureInfo` 表达，不把异步发送失败伪装成
`SubmitEmail` 的同步 gRPC 错误。

## 6. 模板协议

V1 只允许：

```text
template_key + locale + typed payload
```

验证码模板标识建议为：

```text
verification_code.v1
```

AI-Nexus 不传邮件标题、HTML、SMTP 参数或动态回调 URL。Mail Service 负责模板版本、
主题、HTML、纯文本版本和样式。

## 7. 幂等保留时间

Mail Service 对 `request_id` 的幂等记录保留时间必须覆盖：

```text
最大排队时间 + 最大重试时间 + 回调重试时间 + 运维对账窗口
```

V1 建议至少保留 24 小时。即使邮件正文敏感数据已经清理，幂等键和最终状态仍应
保留到窗口结束。

## 8. 传输与敏感信息

- 本地开发可以使用 insecure gRPC，但必须由配置显式开启；
- 测试和生产使用 TLS，生产推荐 mTLS；
- 不把验证码、邮箱或供应商密钥放入 gRPC 错误详情；
- 链路追踪通过 gRPC Metadata 传递，不在业务 Payload 重复设计 Trace 字段；
- 日志只记录脱敏邮箱、`request_id`、`message_id`、状态和稳定错误码。
