# Mail Service V0.1：AI-Nexus 实施与联调手册

## 已实现范围

- 使用 Email-Service 固定提交的官方生成包，调用 `DeliveryService.SubmitEmail/GetEmail`；
- Nexus 生成六位随机验证码，仅保存 HMAC 摘要和邮箱 HMAC 指纹；
- MySQL 保存显式验证码状态机及回调事件 Journal；
- Nexus 在 `:8081` 实现 `DeliveryEventReceiverService.ReportDeliveryEvent`；
- `event_id` 重复和低 `sequence` 事件在同一事务内分别返回 `DUPLICATE`、`IGNORED_STALE`；
- `PROVIDER_ACCEPTED/DELIVERED` 只激活一次，不延长验证码有效期；
- 注册、验证码登录、重置密码统一使用带行锁的 `VerifyAndConsume`；
- Redis 只保存按邮箱指纹和 purpose 隔离的 60 秒发送冷却；
- `Idempotency-Key` HTTP Header 可固定一次逻辑发送；省略时由 Nexus 生成 UUID；
- 长时间 `PENDING_DISPATCH` 记录通过 `GetEmail` 定时对账。

## HTTP 语义

`POST /user/send-code` 继续接收表单：

```text
email=<address>
purpose=1|2|3
```

用途为 `1=REGISTER`、`2=RESET_PASSWORD`、`3=LOGIN`。成功返回 `202 Accepted` 和
`request_id`。同一逻辑请求重试时必须复用原 `Idempotency-Key`；不能自行换 key，否则代表申请新邮件。

`202` 只说明 Mail Service 已可靠受理。验证码需要在 Nexus 收到 `PROVIDER_ACCEPTED`
或首次 `DELIVERED` 后才进入 `ACTIVE`。

## 本地启动

AI-Nexus：

```bash
cd ~/workspace/AI-Nexus
make infra-up
make migrate-mail-integration
make run
```

Email-Service 默认 RabbitMQ 端口与 Nexus 冲突。本机同时运行时，在
`Email-Service/.env` 使用 `5673/15673`，并让 `RABBITMQ_URL` 指向 `localhost:5673`，然后执行：

```bash
cd ~/workspace/Email-Service
set -a; source .env; set +a
make infra-up
make migrate-up
make db-dev-seed
make run
```

本地端口：Nexus HTTP `8000`、Mail Service gRPC `8080`、Nexus 回调 gRPC `8081`。
Fake Provider 可以验证状态链路，但不会真正发送邮件；要从邮箱读取验证码，需按 Email-Service
说明显式启用 SMTP。

## 数据库与测试

已有 MySQL 数据卷必须执行：

```bash
make migrate-mail-integration
```

验证普通测试和真实 MySQL 事务：

```bash
go test ./...
make test-mail-integration
```

测试覆盖协议字段映射、回调 ACK、一次激活、终态映射、重复/乱序事件、失败次数记录和
验证码一次性消费。

## 当前限制

- 本地允许 insecure gRPC；生产必须配置 TLS，后续接入控制面身份时升级 mTLS；
- V0.1 没有对外验证码状态查询 HTTP API；状态恢复由内部对账任务负责；
- Email-Service 当前对非空 `metadata` 的 JSONB 读回字节规范化存在一致性问题，Nexus 按协议省略
  该可选字段；不影响验证码投递与关联；
- 迁移会删除旧 `user_verification_codes` 表；其中是已废弃的短期明文验证码，不具备保留价值。
