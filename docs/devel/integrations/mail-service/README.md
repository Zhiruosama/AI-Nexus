# 独立邮件服务集成

本目录描述 AI-Nexus 与独立 Mail Service 之间的职责边界、验证码生命周期和
gRPC V1 协议。目标是替换当前由外部服务生成验证码的简单调用，建立可异步投递、
可重试、可熔断、可观测的邮件系统。

## 当前状态

V0.1 已在 AI-Nexus 实施并完成 Fake Provider 双服务联调。当前唯一协议来源是
Email-Service 仓库的 `api/proto/mailservice/delivery/v1/`，Nexus 通过固定的 Email-Service
提交版本依赖生成包。原 `ainexus.mail.v1` 草案和 `VarifyService/GetVarifyCode` 已废弃并删除。

## 阅读顺序

1. [需求目标](requirements.md)
2. [架构与核心流程](architecture.md)
3. [gRPC 协议约定](protocol.md)
4. [AI-Nexus 实施与联调手册](implementation.md)
5. [AI-Nexus 改造清单](ai-nexus-changes.md)
6. [初始设计会话交接上下文（历史）](handoff.md)

## 一句话边界

```text
AI-Nexus 拥有验证码及其业务状态；Mail Service 拥有邮件模板和投递过程。
```

Mail Service 不得直接读写 AI-Nexus 的 Redis 或业务数据库，AI-Nexus 也不直接
管理 SMTP 重试、供应商熔断和邮件模板。
