# 独立邮件服务集成

本目录描述 AI-Nexus 与独立 Mail Service 之间的职责边界、验证码生命周期和
gRPC V1 协议。目标是替换当前由外部服务生成验证码的简单调用，建立可异步投递、
可重试、可熔断、可观测的邮件系统。

## 当前状态

协议和系统边界处于设计确认阶段，尚未在 AI-Nexus 中实施。文档中的能力不得视为
当前代码已经支持。

## 阅读顺序

1. [需求目标](requirements.md)
2. [架构与核心流程](architecture.md)
3. [gRPC 协议约定](protocol.md)
4. [mail-v1.proto 草案](mail-v1.proto)
5. [AI-Nexus 改造清单](ai-nexus-changes.md)
6. [另一开发会话交接上下文](handoff.md)

## 一句话边界

```text
AI-Nexus 拥有验证码及其业务状态；Mail Service 拥有邮件模板和投递过程。
```

Mail Service 不得直接读写 AI-Nexus 的 Redis 或业务数据库，AI-Nexus 也不直接
管理 SMTP 重试、供应商熔断和邮件模板。
