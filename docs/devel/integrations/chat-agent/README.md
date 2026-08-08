# 独立 Chat Agent 集成

本目录描述将当前 AI-Nexus 内部 Chat 模块解耦为独立 Chat Agent Service 的 V1
方案，覆盖职责边界、数据所有权、流式 gRPC 协议、Agent Run 状态机、迁移和回滚。

## 当前状态

当前代码仍由 AI-Nexus 单体进程直接管理对话数据、Provider Credential 和 SSE
模型流。本文档是已讨论的目标设计，不代表独立服务已经实施。

## 阅读顺序

1. [需求目标](requirements.md)
2. [架构、状态机与数据设计](architecture.md)
3. [gRPC 和 SSE 协议约定](protocol.md)
4. [chat-agent-v1.proto 草案](chat-agent-v1.proto)
5. [AI-Nexus 改造与迁移清单](ai-nexus-changes.md)
6. [另一开发会话交接上下文](handoff.md)

## V1 定位

V1 先完成现有多 Provider 流式对话能力的可靠迁移，并建立 Agent Runtime 的基础：

```text
Conversation + Message + Provider Credential
                ↓
         有状态 Agent Run
                ↓
       gRPC Server Streaming
```

Tool Calling、长期记忆、RAG、MCP 和多 Agent 协作属于后续阶段，不能为了使用
“Agent”名称而塞入第一次服务拆分。

## 一句话边界

```text
AI-Nexus 负责外部身份认证和统一 HTTP/SSE 入口；
Chat Agent 负责对话数据、模型凭证、Agent Run 和模型调用。
```
