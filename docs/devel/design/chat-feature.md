# AI 对话功能设计文档

## 1. 核心思路

在现有图像生成的基础上，新增 AI 对话模块。用户自带 API Key，平台提供统一的多模型对话接口，支持 OpenAI / Anthropic / Gemini 及任意 OpenAI 兼容端点（自定义 base_url）。

与图像生成不同，对话是**同步流式**的——用户发一条消息，立即开始收到逐 token 的响应。因此不走 RabbitMQ 异步队列，而是直接 SSE（Server-Sent Events）流式返回。

## 2. 功能清单

### P0（核心）

| 功能              | 说明                                                  |
| ----------------- | ----------------------------------------------------- |
| 多模型对话        | 统一接口调用 OpenAI / Anthropic / Gemini              |
| SSE 流式响应      | 逐 token 推送，前端实时渲染                           |
| 自定义接入点      | 用户可填 base_url + api_key，接入任意 OpenAI 兼容服务 |
| 用户 API Key 管理 | 加密存储，支持增删改查，按 provider 分类              |
| 对话管理          | 创建/删除/列表对话，每个对话独立上下文                |
| 消息持久化        | 完整保存对话历史（用户消息 + 模型回复）               |

### P1（增强）

| 功能             | 说明                                             |
| ---------------- | ------------------------------------------------ |
| 系统预设         | 可设置 system prompt，支持保存为模板复用         |
| 参数调节         | temperature / top_p / max_tokens 等，前端可调    |
| Token 用量统计   | 记录每次对话的 token 消耗（prompt + completion） |
| 对话标题自动生成 | 首轮对话后调用模型自动生成标题                   |
| 上下文窗口管理   | 当历史消息超过模型上下文限制时，自动截断早期消息 |

### P2（进阶，暂不实现）

| 功能       | 说明                                         |
| ---------- | -------------------------------------------- |
| 多模态输入 | 支持发送图片（Vision），复用现有图片上传逻辑 |
| 对话分享   | 生成分享链接，他人可只读查看                 |
| 流式中断   | 用户可中途停止生成                           |
| 费用估算   | 根据各模型定价估算花费                       |

## 3. 架构设计

### 3.1 Provider 抽象层

所有模型提供商实现统一接口，新增 provider 只需实现该接口：

```go
// Provider 定义对话模型提供商的统一接口
type Provider interface {
    // ChatStream 发起流式对话，通过 channel 逐块返回
    ChatStream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
}

type ChatRequest struct {
    Model       string
    Messages    []Message
    Temperature float64
    TopP        float64
    MaxTokens   int
    // provider 特定的额外参数
    Extra       map[string]any
}

type Message struct {
    Role    string // system / user / assistant
    Content string
}

type StreamChunk struct {
    Delta        string // 增量文本
    FinishReason string // stop / length / error
    Usage        *Usage // 仅最后一个 chunk 携带
    Err          error
}

type Usage struct {
    PromptTokens     int
    CompletionTokens int
}
```

### 3.2 Provider 实现

```
internal/pkg/chat/
├── provider.go          # Provider 接口定义 + 工厂函数
├── openai.go            # OpenAI / OpenAI 兼容（自定义 base_url 也走这个）
├── anthropic.go         # Anthropic Claude（Messages API）
└── gemini.go            # Google Gemini（generateContent API）
```

**关键点**：OpenAI 兼容模式覆盖了 DeepSeek、Moonshot、GLM、本地 Ollama 等大量服务。用户只需填 base_url + api_key + model name 即可接入，无需为每个服务单独写 provider。

Anthropic 和 Gemini 的 API 格式不同，需要独立实现：

- Anthropic：`POST /v1/messages`，`stream: true`，SSE 格式为 `event: content_block_delta`
- Gemini：`POST /v1beta/models/{model}:streamGenerateContent?alt=sse`

### 3.3 SSE 流式传输

对话不走 WebSocket（已有的 WS 用于图像生成任务推送），而是用 SSE：

```
POST /chat/completions  (SSE)
Content-Type: text/event-stream

data: {"delta":"你","finish_reason":""}
data: {"delta":"好","finish_reason":""}
data: {"delta":"！","finish_reason":"stop","usage":{"prompt_tokens":10,"completion_tokens":3}}
data: [DONE]
```

**为什么选 SSE 而非复用 WebSocket**：

- 对话是请求-响应模式，SSE 语义更匹配
- 每个对话请求独立，不需要维持长连接状态
- 前端用 `fetch` + `ReadableStream` 即可处理，不依赖 WS 连接
- WS 继续用于图像生成等异步任务推送，职责分离

### 3.4 数据模型

```sql
-- 用户 API Key（加密存储）
CREATE TABLE user_api_keys (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  user_uuid   CHAR(36) NOT NULL,
  provider    VARCHAR(32) NOT NULL COMMENT 'openai / anthropic / gemini / custom',
  name        VARCHAR(64) NOT NULL COMMENT '用户自定义名称，如"我的 GPT Key"',
  api_key_enc VARBINARY(512) NOT NULL COMMENT 'AES-GCM 加密后的 API Key',
  base_url    VARCHAR(512) DEFAULT '' COMMENT '自定义端点（custom provider 必填）',
  is_active   BOOLEAN DEFAULT TRUE,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_user_provider (user_uuid, provider)
);

-- 对话
CREATE TABLE conversations (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  conv_id     CHAR(36) NOT NULL UNIQUE COMMENT '对话 UUID',
  user_uuid   CHAR(36) NOT NULL,
  title       VARCHAR(128) DEFAULT '' COMMENT '对话标题',
  provider    VARCHAR(32) NOT NULL,
  model       VARCHAR(64) NOT NULL COMMENT '使用的模型名',
  system_prompt TEXT COMMENT '系统预设',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_user_updated (user_uuid, updated_at DESC)
);

-- 对话消息
CREATE TABLE conversation_messages (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  conv_id     CHAR(36) NOT NULL,
  role        ENUM('system','user','assistant') NOT NULL,
  content     TEXT NOT NULL,
  token_count INT UNSIGNED DEFAULT 0 COMMENT '该条消息的 token 数',
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  INDEX idx_conv_created (conv_id, created_at)
);
```

### 3.5 模块结构（遵循现有分层）

```
internal/
├── routes/chat/              # 路由注册
├── controller/chat/          # 参数绑定、SSE 写入
├── service/chat/             # 业务编排：取 key、拼消息、调 provider、存记录
├── dao/chat/                 # DB 操作
├── domain/
│   ├── do/chat/              # conversations / messages / user_api_keys 表映射
│   ├── dto/chat/             # 请求体：SendMessageDTO / CreateConversationDTO 等
│   ├── vo/chat/              # 响应体：ConversationVO / MessageVO 等
│   └── query/chat/           # 查询参数
└── pkg/chat/                 # Provider 抽象 + 各实现
```

### 3.6 请求流程

```
用户发消息
  │
  ▼
Controller: 绑定 DTO，设置 SSE 响应头
  │
  ▼
Service:
  ├─ 1. 查 user_api_keys 表，解密 API Key
  ├─ 2. 查 conversation_messages 表，组装历史消息
  ├─ 3. 构造 ChatRequest，调用 Provider.ChatStream()
  ├─ 4. 逐 chunk 写 SSE 响应（同时拼接完整回复）
  └─ 5. 流结束后：保存用户消息 + 助手回复到 DB，更新 token 用量
  │
  ▼
前端: fetch + ReadableStream 逐块渲染
```

## 4. API 设计

### 4.1 API Key 管理

```
POST   /chat/api-keys          创建 API Key
GET    /chat/api-keys           获取用户所有 Key（api_key 脱敏显示）
PUT    /chat/api-keys/:id       更新（改名/换 key/改 base_url）
DELETE /chat/api-keys/:id       删除
```

### 4.2 对话管理

```
POST   /chat/conversations                   创建对话（指定 provider + model）
GET    /chat/conversations                    获取用户对话列表（分页，按更新时间倒序）
GET    /chat/conversations/:conv_id           获取对话详情 + 历史消息
DELETE /chat/conversations/:conv_id           删除对话
PUT    /chat/conversations/:conv_id/title     修改标题
```

### 4.3 对话交互

```
POST   /chat/conversations/:conv_id/messages  发送消息（SSE 流式返回）
```

请求体：

```json
{
  "content": "你好",
  "api_key_id": 1,
  "temperature": 0.7,
  "max_tokens": 2048
}
```

### 4.4 系统预设（P1）

```
POST   /chat/presets            创建预设
GET    /chat/presets            列表
PUT    /chat/presets/:id        更新
DELETE /chat/presets/:id        删除
```

## 5. 安全设计

| 关注点       | 方案                                                                  |
| ------------ | --------------------------------------------------------------------- |
| API Key 存储 | AES-256-GCM 加密，密钥由 config.yaml `chat.encryption_key` 配置 |
| API Key 展示 | 列表接口只返回 `sk-****xxxx` 形式的掩码                             |
| 请求鉴权     | 所有 `/chat/*` 路由挂 AuthMiddleware + RateLimitingMiddleware       |
| Key 归属校验 | Service 层校验 api_key_id 属于当前用户，防止越权使用他人 Key          |
| 输入过滤     | 复用现有 content-moderation 敏感词过滤                                |

## 6. 已确认的决策

| 决策项 | 结论 |
|---|---|
| 加密密钥管理 | 写入 `configs/config.yaml` 新增 `chat.encryption_key` 字段 |
| 消息持久化 | 设定上限，每个对话最多 200 条消息，超出后最早的消息不再发送给模型（DB 保留） |
| 功能范围 | P0 + P1 全部实现 |
| 前端 | 仅做展示 Demo，放在 `http/` 目录下，单 HTML 文件（PWA 风格，fetch + SSE） |
| 平台内置 Key | 不做，必须用户自带 API Key |

## 7. 前端 Demo

放在 `http/html/chat.html`，单文件 PWA 风格，功能：

- API Key 管理（增删、选择 provider）
- 对话列表侧边栏（创建/删除/切换）
- 聊天界面（消息气泡 + SSE 流式渲染）
- 参数面板（model / temperature / max_tokens / system prompt）
- 纯 HTML + CSS + 原生 JS，不依赖框架，fetch 调用后端 API
