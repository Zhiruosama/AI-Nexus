# AI-Nexus 文档索引

AI-Nexus 的目标是演进为“面向多模型 AI 服务的统一接入、异步任务调度与账户计费平台”。

## 当前文档

- [项目路线图](roadmap.md)：阶段目标、范围边界、验收标准与风险。
- [开发指南](guide/README.md)：本地启动、基础设施与接口测试入口。
- [架构设计](devel/design/design.md)：现有整体设计。
- [AI 生图核心设计](devel/design/ai-generation-core-design.md)
- [对话功能设计](devel/design/chat-feature.md)
- [中间件设计](devel/middleware/middleware.md)
- [可撤销会话认证](devel/security/session-auth/README.md)：JWT 与 Redis 会话设计。
- [独立邮件服务集成](devel/integrations/mail-service/README.md)：验证码职责边界、gRPC V1 协议与双系统改造计划。
- [独立 Chat Agent 需求](devel/integrations/chat-agent/README.md)：对话能力拆分的目标、职责边界、核心需求与交接说明。

文档应以代码和可复现验证为准。尚未落地或尚未验证的设计不得描述为已实现能力。
