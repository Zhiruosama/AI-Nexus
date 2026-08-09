# 本地开发指南

## 启动

```bash
make bootstrap
make infra-up
make migrate-mail-integration
make run
```

默认 API 地址为 `http://localhost:8000`。首次启动会创建本机私有 `.env` 和 `configs/config.yaml`；两者均不得提交。

## 基础设施

| 服务 | 地址 |
| --- | --- |
| MySQL | `127.0.0.1:13306` |
| Redis | `127.0.0.1:6379` |
| RabbitMQ AMQP | `127.0.0.1:5672` |
| RabbitMQ 管理台 | `http://localhost:15672` |

查看状态：`make infra-status`。停止：`make infra-down`。`make infra-reset` 会删除本项目的容器数据卷，仅用于可丢弃的本地数据。

## 验证与接口测试

```bash
make check
```

Postman Collection 和环境模板位于仓库根目录 `postman/`。导入后选择 `AI-Nexus Local` 环境；登录请求会将返回的 JWT 保存到 `jwt_token`。

验证码邮件依赖独立 Email-Service：本地默认从 `127.0.0.1:8080` 调用，并在 `:8081`
接收投递事件。完整启动和端口冲突处理见
[Mail Service 实施与联调手册](../devel/integrations/mail-service/implementation.md)。
ModelScope 生图仍需要单独配置 API Key。
