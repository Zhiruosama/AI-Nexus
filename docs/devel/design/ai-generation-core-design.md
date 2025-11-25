# AI生图核心模块设计文档

## 一、系统架构设计

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                          Frontend 前端                           │
│  - 提交生图请求(文生图/图生图)                                      │
│  - WebSocket 实时接收任务进度                                      │
│  - 展示生成结果与历史记录                                           │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTP + WebSocket
┌────────────────────────▼────────────────────────────────────────┐
│                    Gin HTTP Server                               │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 现有中间件层(完全复用)                                       │   │
│  │  - JWT 认证 (AuthMiddleware)                              │   │
│  │  - 请求限流 (RateLimitingMiddleware)                      │   │
│  │  - 防重放 (DeduplicationMiddleware)                       │   │
│  │  - RequestID 追踪                                         │   │
│  │  - Logger                                                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Controller 层                                              │   │
│  │  - GenerationController (生图请求处理)                    │   │
│  │  - TaskController (任务状态查询)                          │   │
│  │  - ModelController (模型列表)                             │   │
│  │  - WebSocketController (WS连接管理)                       │   │
│  │  - 参数校验与预处理                                        │    │
│  └──────────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    Service 服务层                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ GenerationService                                        │    │
│  │  - 创建任务记录                                            │    │
│  │  - 发送消息到 RabbitMQ                                     │    │
│  │  - 任务状态查询与更新                                       │    │
│  │  - 任务取消逻辑                                            │    │
│  └─────────────────────────────────────────────────────────┘    │
└──────┬──────────────────────────────────────────────────────────┘
       │
       │ 发布消息
┌──────▼──────────────────────────────────────────────────────────┐
│                    RabbitMQ 消息队列                              │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Exchange: generation.topic (Topic类型)                    │   │
│  │                                                           │   │
│  │ Routing Keys:                                             │   │
│  │  - generation.text2img    (文生图任务)                     │   │
│  │  - generation.img2img     (图生图任务)                     │   │
│  └─────────────────────┬────────────────────────────────────┘   │
│                        │                                         │
│  ┌─────────────────────▼────────────────────────────────────┐   │
│  │ Queues (持久化队列)                                         │   │
│  │  - queue.text2img       (文生图队列)                       │   │
│  │  - queue.img2img        (图生图队列)                       │   │
│  │  - queue.dead_letter    (死信队列, 处理失败任务)            │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ Worker 消费消息
┌──────────────────────────▼──────────────────────────────────────┐
│                   Worker Pool 工作进程池                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Text2Img Workers (并发数: 3-5)                            │   │
│  │  - 从 queue.text2img 消费消息                             │   │
│  │  - 调用 ModelScope API (文生图模型)                       │   │
│  │  - 保存图片到本地存储                                      │   │
│  │  - 更新任务状态到 MySQL                                    │   │
│  │  - 通过 WebSocket 推送进度                                 │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Img2Img Workers (并发数: 2-3)                             │   │
│  │  - 从 queue.img2img 消费消息                              │   │
│  │  - 调用 ModelScope API (图生图模型)                       │   │
│  │  - 处理逻辑同上                                            │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────┬─────────────────────┬─────────────────────────────────┘
           │                     │
┌──────────▼─────────┐  ┌────────▼──────────┐
│  ModelScope SDK    │  │ 本地文件存储       │
│                    │  │                   │
│ - 文生图模型        │  │ - static/images/  │
│ - 图生图模型        │  │ - UUID命名        │
│ - 自动重试机制      │  │ - WebP压缩        │
└──────────┬─────────┘  └────────┬──────────┘
           │                     │
           └──────────┬──────────┘
                      │
┌─────────────────────▼────────────────────────────────────────────┐
│                    数据持久层                                      │
│  ┌────────────────────┐          ┌──────────────────────┐        │
│  │  MySQL 数据库       │          │  Redis 缓存           │        │
│  │                    │          │                      │        │
│  │  - users (现有)     │          │  - 任务状态缓存       │        │
│  │  - generation_tasks│          │  - WebSocket Session │        │
│  │  - generation_models│         │  - 限流计数器         │        │
│  └────────────────────┘          └──────────────────────┘        │
└──────────────────────────────────────────────────────────────────┘
```

### 1.2 核心设计原则

#### 1.2.1 解耦设计
- **HTTP服务与Worker分离**: 接收请求和处理任务完全解耦
- **消息队列中间件**: RabbitMQ作为通信桥梁,支持异步处理

#### 1.2.2 异步处理
- 用户提交请求后立即返回任务ID
- Worker异步处理生图任务
- WebSocket实时推送进度与结果

#### 1.2.3 高可用设计
- RabbitMQ持久化队列,防止消息丢失
- 死信队列处理失败任务
- 自动重试机制(最多3次)

#### 1.2.4 现有组件复用
- **完全复用现有中间件**: JWT认证、限流、防重放、日志追踪
- **复用工具类**: Logger、图片处理、UUID生成
- **沿用分层架构**: Controller → Service → DAO

---

## 二、技术选型

### 2.1 核心技术栈

| 组件类型 | 技术选型 | 版本要求 | 说明 |
|---------|---------|---------|------|
| **后端框架** | Gin | v1.11+ | 现有框架,无需更换 |
| **ORM** | GORM | v1.31+ | 现有ORM,复用即可 |
| **消息队列** | RabbitMQ | 3.13+ | 可靠的消息中间件 |
| **数据库** | MySQL | 8.0+ | 现有数据库 |
| **缓存** | Redis | 7.0+ | 现有缓存 |
| **AI服务** | ModelScope | 最新 | 阿里达摩院开源平台 |
| **实时通信** | WebSocket | - | gorilla/websocket |
| **图片存储** | 本地文件系统 | - | static/images/ |

### 2.2 新增Go依赖

```go
// go.mod 新增依赖
require (
    // RabbitMQ客户端
    github.com/rabbitmq/amqp091-go v1.10.0

    // WebSocket支持
    github.com/gorilla/websocket v1.5.1

    // ModelScope SDK (如果有官方Go SDK)
    // 或使用HTTP客户端直接调用API
    // github.com/go-resty/resty/v2 v2.11.0

    // 图片处理(现有项目已有)
    // github.com/disintegration/imaging v1.6.2
)
```

### 2.3 ModelScope选型理由

1. **免费额度**: 提供一定的免费调用额度,适合初期开发
2. **国内访问快**: 阿里云服务,国内网络稳定
3. **模型丰富**: 支持多种Stable Diffusion模型
4. **API简单**: RESTful API,易于集成
5. **中文文档**: 官方中文文档完善

---

## 三、数据库设计

### 3.1 核心表设计

#### 3.1.1 生图任务表 (image_generation_tasks)

```sql
CREATE TABLE IF NOT EXISTS `image_generation_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_id` CHAR(36) NOT NULL COMMENT '任务UUID,对外暴露',
  `user_uuid` CHAR(36) NOT NULL COMMENT '用户UUID, 关联 users.uuid',

  -- 任务基本信息
  `task_type` TINYINT UNSIGNED NOT NULL COMMENT '任务类型: 1-文生图, 2-图生图',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '任务状态: 0-待处理, 1-队列中, 2-处理中, 3-已完成, 4-失败, 5-已取消',

  -- 输入参数
  `prompt` TEXT NOT NULL COMMENT '正向提示词，最大512字符',
  `negative_prompt` TEXT COMMENT '负向提示词，最大512字符',
  `model_id` VARCHAR(64) NOT NULL COMMENT '模型ID,关联 generation_models.model_id',
  `width` INT UNSIGNED DEFAULT 512 COMMENT '图片宽度(像素)',
  `height` INT UNSIGNED DEFAULT 512 COMMENT '图片高度(像素)',
  `num_inference_steps` INT UNSIGNED DEFAULT 20 COMMENT '推理步数(20-50)',
  `guidance_scale` DECIMAL(4,2) DEFAULT 7.5 COMMENT 'CFG Scale(1.0-20.0)',
  `seed` BIGINT COMMENT '随机种子 (用户指定或自动生成)',

  -- 图生图专用参数
  `input_image_url` VARCHAR(512) COMMENT '输入图片URL (仅图生图)',
  `strength` DECIMAL(3,2) DEFAULT 0.75 COMMENT '强度 0.00-1.00 (仅图生图)',

  -- 输出结果
  `output_image_url` VARCHAR(512) COMMENT '生成的图片URL',
  `actual_seed` BIGINT COMMENT '实际使用的种子值',

  -- 错误处理
  `error_message` TEXT COMMENT '错误详情',
  `retry_count` TINYINT UNSIGNED DEFAULT 0 COMMENT '已重试次数',
  `max_retry` TINYINT UNSIGNED DEFAULT 3 COMMENT '最大重试次数',

  -- 性能指标
  `generation_time_ms` INT UNSIGNED COMMENT '生成耗时(毫秒)',
  `queue_time_ms` INT UNSIGNED COMMENT '队列等待时长(毫秒)',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `queued_at` DATETIME COMMENT '进入队列时间',
  `started_at` DATETIME COMMENT '开始处理时间',
  `completed_at` DATETIME COMMENT '完成时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_task_id` (`task_id`),
  KEY `idx_user_uuid` (`user_uuid`),
  KEY `idx_status` (`status`),
  KEY `idx_user_status` (`user_uuid`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI生图任务表';
```

#### 3.1.2 模型配置表 (image_generation_models)

```sql
CREATE TABLE IF NOT EXISTS `image_generation_models` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `model_id` VARCHAR(64) NOT NULL COMMENT '模型标识 (如: sd-v1.5, sd-v2.1)',

  -- 基本信息
  `model_name` VARCHAR(128) NOT NULL COMMENT '模型显示名称',
  `model_type` VARCHAR(32) NOT NULL COMMENT '类型: text2img/img2img',
  `provider` VARCHAR(32) DEFAULT 'modelscope' COMMENT '提供商: modelscope',

  -- 显示与排序
  `description` TEXT COMMENT '模型描述',
  `tags` JSON COMMENT '标签: ["快速", "高质量"]',
  `sort_order` INT DEFAULT 0 COMMENT '排序权重',

  -- 统计信息
  `total_usage` BIGINT UNSIGNED DEFAULT 0 COMMENT '累计使用次数',
  `success_rate` DECIMAL(5,2) COMMENT '成功率百分比',

  -- 状态
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否启用',
  `is_recommended` BOOLEAN DEFAULT FALSE COMMENT '是否推荐',

  -- 第三方平台相关
  `third_party_model_id` VARCHAR(128) NOT NULL COMMENT '第三方平台模型ID',
  `base_url` VARCHAR(512) COMMENT 'API调用地址',

  -- 能力参数
  `default_width` INT UNSIGNED DEFAULT 512 COMMENT '默认宽度',
  `default_height` INT UNSIGNED DEFAULT 512 COMMENT '默认高度',
  `max_width` INT UNSIGNED DEFAULT 1024 COMMENT '最大宽度',
  `max_height` INT UNSIGNED DEFAULT 1024 COMMENT '最大高度',
  `min_steps` INT UNSIGNED DEFAULT 10 COMMENT '最小推理步数',
  `max_steps` INT UNSIGNED DEFAULT 100 COMMENT '最大推理步数',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_model_id` (`model_id`),
  KEY `idx_model_type` (`model_type`),
  KEY `idx_provider` (`provider`),
  KEY `idx_tags` (`tags`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='生图模型配置表';
```

### 3.2 初始数据

#### 初始化模型配置

```sql
-- 插入初始模型配置
INSERT INTO `generation_models` (
  `model_id`,
  `model_name`,
  `model_type`,
  `modelscope_model_id`,
  `description`,
  `is_active`,
  `is_recommended`
) VALUES
(
  'sd-v1.5',
  'Stable Diffusion 1.5',
  'text2img',
  'AI-ModelScope/stable-diffusion-v1-5',
  '经典的SD1.5模型,速度快,质量稳定',
  TRUE,
  TRUE
),
(
  'sd-v2.1',
  'Stable Diffusion 2.1',
  'text2img',
  'AI-ModelScope/stable-diffusion-2-1',
  'SD2.1模型,生成质量更高',
  TRUE,
  FALSE
);
```

### 3.3 索引优化

```sql
-- 高频查询场景优化
-- 1. 用户查询自己的任务列表
EXPLAIN SELECT * FROM generation_tasks
WHERE user_uuid = ? AND status = 3
ORDER BY created_at DESC LIMIT 20;
-- 使用索引: idx_user_status

-- 2. Worker查询待处理任务
EXPLAIN SELECT * FROM generation_tasks
WHERE status = 1
ORDER BY created_at ASC LIMIT 1;
-- 使用索引: idx_status
```

---

## 四、RabbitMQ队列设计

### 4.1 Exchange设计

```yaml
类型: Topic Exchange
名称: generation.topic
持久化: true
自动删除: false
内部: false
```

### 4.2 Routing Key规范

```
格式: generation.{task_type}

示例:
- generation.text2img    (文生图任务)
- generation.img2img     (图生图任务)
```

### 4.3 队列设计

#### 4.3.1 主业务队列

```yaml
queue.text2img:
  durable: true                     # 持久化
  auto_delete: false
  exclusive: false
  arguments:
    x-message-ttl: 1800000          # 消息TTL: 30分钟
    x-max-length: 1000              # 最大队列长度
    x-dead-letter-exchange: generation.dlx
    x-dead-letter-routing-key: dead_letter

queue.img2img:
  # 配置同上
```

#### 4.3.2 死信队列

```yaml
exchange: generation.dlx (Direct Exchange)

queue.dead_letter:
  durable: true
  arguments:
    x-message-ttl: 604800000        # 7天后自动删除
```

**死信场景:**
1. 消息被拒绝 (basic.nack, requeue=false)
2. 消息TTL过期
3. 队列达到最大长度

### 4.4 消息格式

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_uuid": "user-uuid-here",
  "task_type": 1,
  "retry_count": 0,
  "created_at": "2025-11-09T10:30:00Z",

  "payload": {
    "prompt": "A majestic cat sitting on a throne",
    "negative_prompt": "blurry, low quality",
    "model_id": "sd-v1.5",
    "width": 512,
    "height": 512,
    "num_inference_steps": 20,
    "guidance_scale": 7.5,
    "seed": 1234567890
  }
}
```

### 4.5 消息确认机制

#### 生产者确认 (Publisher Confirms)
```go
// 启用确认模式
channel.ConfirmSelect()

// 发布消息
err := channel.PublishWithContext(
    ctx,
    "generation.topic",
    "generation.text2img",
    false, // mandatory
    false, // immediate
    amqp091.Publishing{
        DeliveryMode: amqp091.Persistent, // 持久化
        ContentType:  "application/json",
        MessageId:    task.TaskID,
        Body:         jsonBytes,
    },
)

// 等待确认
if err != nil {
    // 发送失败,回滚数据库
}
```

#### 消费者确认 (Consumer Acknowledgements)
```go
// 手动确认模式
msgs, err := channel.Consume(
    "queue.text2img",
    "",    // consumer
    false, // auto-ack = false (手动确认)
    false, // exclusive
    false, // no-local
    false, // no-wait
    nil,   // args
)

// 处理消息
if processSuccess {
    msg.Ack(false) // 确认成功
} else if canRetry {
    msg.Nack(false, true) // 拒绝并重新入队
} else {
    msg.Nack(false, false) // 拒绝并进入死信队列
}
```

---

## 五、业务流程设计

### 5.1 文生图完整流程

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 用户提交请求                                               │
└─────────────────────────────────────────────────────────────┘

用户 → POST /api/v1/generation/text2img
       Headers: Authorization: Bearer {jwt_token}
       Body: {
         "prompt": "A beautiful sunset over mountains",
         "negative_prompt": "blurry, low quality",
         "model_id": "sd-v1.5",
         "width": 512,
         "height": 512,
         "num_inference_steps": 20,
         "guidance_scale": 7.5
       }

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 2. Controller层处理                                          │
└─────────────────────────────────────────────────────────────┘

GenerationController.Text2Img(ctx *gin.Context)
  │
  ├─ 1. 从JWT中间件获取 user_uuid
  │
  ├─ 2. 参数校验 (Gin Validator)
  │    - prompt不为空
  │    - model_id存在
  │    - width/height合法
  │
  ├─ 3. 调用 Service 层
  │    task, err := service.CreateText2ImgTask(dto)
  │
  └─ 4. 返回响应
       ctx.JSON(200, gin.H{
         "code": 200,
         "data": {
           "task_id": "xxx",
           "status": "queued"
         }
       })

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 3. Service层处理                                             │
└─────────────────────────────────────────────────────────────┘

GenerationService.CreateText2ImgTask(dto)
  │
  ├─ 1. 生成 task_id (UUID)
  │    taskID := uuid.New().String()
  │
  ├─ 2. 校验模型是否存在
  │    model := dao.GetModelByID(dto.ModelID)
  │    IF model == nil THEN RETURN Error("模型不存在")
  │
  ├─ 3. 生成随机种子(如果未指定)
  │    IF dto.Seed == 0 THEN
  │      dto.Seed = rand.Int63()
  │
  ├─ 4. 创建任务记录
  │    task := &GenerationTaskDO{
  │      TaskID:    taskID,
  │      UserUUID:  userUUID,
  │      TaskType:  1, // 文生图
  │      Status:    0, // 待处理
  │      Prompt:    dto.Prompt,
  │      ModelID:   dto.ModelID,
  │      ...
  │    }
  │    dao.CreateTask(task)
  │
  ├─ 5. 构造RabbitMQ消息
  │    message := Message{
  │      TaskID:   taskID,
  │      UserUUID: userUUID,
  │      TaskType: 1,
  │      Payload:  dto,
  │    }
  │
  ├─ 6. 发送到RabbitMQ
  │    err := mq.Publish(
  │      exchange:    "generation.topic",
  │      routingKey:  "generation.text2img",
  │      message:     message,
  │    )
  │    IF err != nil THEN
  │      dao.DeleteTask(taskID) // 回滚
  │      RETURN Error("消息发送失败")
  │
  ├─ 7. 更新任务状态为"队列中"
  │    dao.UpdateTaskStatus(taskID, 1, NOW())
  │
  └─ 8. 返回任务对象
       RETURN task, nil

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 4. Worker消费消息                                            │
└─────────────────────────────────────────────────────────────┘

Text2ImgWorker.HandleMessage(msg)
  │
  ├─ 1. 反序列化消息
  │    task := json.Unmarshal(msg.Body)
  │
  ├─ 2. 查询任务详情
  │    taskDO := dao.GetTaskByID(task.TaskID)
  │
  ├─ 3. 检查任务状态
  │    IF taskDO.Status != 1 THEN // 不是"队列中"
  │      msg.Ack()
  │      RETURN // 可能已被取消
  │
  ├─ 4. 更新状态为"处理中"
  │    dao.UpdateTaskStatus(task.TaskID, 2, NOW())
  │
  ├─ 5. WebSocket推送进度
  │    ws.Send(userUUID, {
  │      "type": "task_progress",
  │      "task_id": taskID,
  │      "status": "processing"
  │    })
  │
  ├─ 6. 调用ModelScope API
  │    │
  │    ├─ 构造请求
  │    │    request := {
  │    │      "model": taskDO.ModelScopeModelID,
  │    │      "input": {
  │    │        "prompt": taskDO.Prompt,
  │    │        "negative_prompt": taskDO.NegativePrompt,
  │    │        "num_inference_steps": taskDO.NumInferenceSteps,
  │    │        "guidance_scale": taskDO.GuidanceScale,
  │    │        "seed": taskDO.Seed,
  │    │      }
  │    │    }
  │    │
  │    ├─ 发送HTTP请求
  │    │    response := http.Post(apiURL, request)
  │    │
  │    ├─ 处理响应
  │    │    IF response.StatusCode != 200 THEN
  │    │      → 错误处理流程
  │    │
  │    └─ 获取生成的图片(Base64)
  │         imageBase64 := response.Data.OutputImage
  │
  ├─ 7. 保存图片到本地
  │    │
  │    ├─ 解码Base64
  │    │    imageData := base64.Decode(imageBase64)
  │    │
  │    ├─ 生成文件名
  │    │    filename := fmt.Sprintf("%s.png", taskID)
  │    │    filePath := "static/images/" + filename
  │    │
  │    ├─ 转换为WebP(复用现有工具)
  │    │    pkg.ConvertToWebP(imageData, filePath)
  │    │
  │    └─ 生成访问URL
  │         outputURL := "/static/images/" + filename
  │
  ├─ 8. 更新任务为"已完成"
  │    dao.UpdateTask(taskID, {
  │      Status:           3,
  │      OutputImageURL:   outputURL,
  │      ActualSeed:       taskDO.Seed,
  │      GenerationTimeMs: elapsedTime,
  │      CompletedAt:      NOW(),
  │    })
  │
  ├─ 9. WebSocket推送完成通知
  │    ws.Send(userUUID, {
  │      "type": "task_completed",
  │      "task_id": taskID,
  │      "output_url": outputURL
  │    })
  │
  └─ 10. 确认消息
        msg.Ack()

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 5. 用户查询结果                                              │
└─────────────────────────────────────────────────────────────┘

方式1: 轮询
  GET /api/v1/generation/tasks/{task_id}

  GenerationService.GetTaskByID(taskID)
    │
    └─ SELECT * FROM generation_tasks WHERE task_id = ?

方式2: WebSocket推送 (推荐)
  用户建立WebSocket连接
  Worker完成后主动推送结果
```

### 5.2 图生图流程

```
图生图流程与文生图类似,区别在于:

1. 请求参数
   - 需要上传原始图片 (multipart/form-data)
   - 增加 strength 参数(0.0-1.0)

2. Service层处理
   - 保存上传的图片到临时目录
   - 生成 input_image_url

3. Worker处理
   - 读取原始图片
   - 转换为Base64传给ModelScope
   - ModelScope API使用img2img模型
```

### 5.3 错误处理流程

```
┌─────────────────────────────────────────────────────────────┐
│ Worker处理失败场景                                           │
└─────────────────────────────────────────────────────────────┘

IF ModelScope API调用失败 THEN
  │
  ├─ 判断错误类型
  │
  ├─ 可重试错误 (网络超时、500错误)
  │   │
  │   ├─ IF retry_count < max_retry (3次) THEN
  │   │   │
  │   │   ├─ 更新重试次数
  │   │   │    UPDATE generation_tasks
  │   │   │    SET retry_count = retry_count + 1
  │   │   │
  │   │   ├─ Nack消息(requeue=true)
  │   │   │    msg.Nack(false, true)
  │   │   │
  │   │   └─ RETURN (消息重新入队)
  │   │
  │   └─ ELSE (超过最大重试次数)
  │       │
  │       └─ 标记为"失败"
  │
  └─ 不可重试错误 (400参数错误、内容违规)
      │
      ├─ 更新任务状态
      │    UPDATE generation_tasks
      │    SET status=4,
      │        error_message=?,
      │        completed_at=NOW()
      │
      ├─ WebSocket推送失败通知
      │    ws.Send(userUUID, {
      │      "type": "task_failed",
      │      "task_id": taskID,
      │      "error": errorMessage
      │    })
      │
      └─ 消息进入死信队列
           msg.Nack(false, false)
```

### 5.4 任务取消流程

```
用户 → DELETE /api/v1/generation/tasks/{task_id}

GenerationService.CancelTask(taskID, userUUID)
  │
  ├─ 1. 查询任务
  │    task := dao.GetTaskByID(taskID)
  │
  ├─ 2. 权限校验
  │    IF task.UserUUID != userUUID THEN
  │      RETURN Error("无权操作")
  │
  ├─ 3. 状态校验
  │    IF task.Status == 2 OR task.Status == 3 THEN
  │      RETURN Error("任务已开始处理,无法取消")
  │
  ├─ 4. 更新状态为"已取消"
  │    dao.UpdateTaskStatus(taskID, 5)
  │
  └─ 5. 返回成功
       RETURN Success
```

---

## 六、接口设计

### 6.1 生图相关接口

#### 6.1.1 文生图

```http
POST /api/v1/generation/text2img
Content-Type: application/json
Authorization: Bearer {jwt_token}

Request Body:
{
  "prompt": "A beautiful sunset over mountains, digital art",
  "negative_prompt": "blurry, low quality, distorted",
  "model_id": "sd-v1.5",
  "width": 512,
  "height": 512,
  "num_inference_steps": 20,
  "guidance_scale": 7.5,
  "seed": 1234567890  // 可选,不传则随机生成
}

Response 200:
{
  "code": 200,
  "message": "任务创建成功",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "queued",
    "estimated_time_seconds": 30,
    "created_at": "2025-11-09T10:30:00Z"
  }
}

Response 400:
{
  "code": 400,
  "message": "参数错误: prompt不能为空"
}
```

#### 6.1.2 图生图

```http
POST /api/v1/generation/img2img
Content-Type: multipart/form-data
Authorization: Bearer {jwt_token}

Request Form:
  - init_image: File (required)
  - prompt: String (required)
  - negative_prompt: String (optional)
  - model_id: String (required)
  - strength: Float (0.0-1.0, default: 0.75)
  - num_inference_steps: Integer (default: 20)
  - guidance_scale: Float (default: 7.5)
  - seed: Integer (optional)

Response: 同 text2img
```

#### 6.1.3 查询任务状态

```http
GET /api/v1/generation/tasks/{task_id}
Authorization: Bearer {jwt_token}

Response 200:
{
  "code": 200,
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "completed",
    "task_type": "text2img",
    "prompt": "A beautiful sunset...",
    "model_id": "sd-v1.5",
    "output_image_url": "/static/images/550e8400.webp",
    "width": 512,
    "height": 512,
    "actual_seed": 1234567890,
    "generation_time_ms": 3500,
    "created_at": "2025-11-09T10:30:00Z",
    "completed_at": "2025-11-09T10:30:35Z"
  }
}

状态值:
- queued: 队列中
- processing: 处理中
- completed: 已完成
- failed: 失败
- cancelled: 已取消
```

#### 6.1.4 获取任务列表

```http
GET /api/v1/generation/tasks?page=1&page_size=20&status=completed
Authorization: Bearer {jwt_token}

Query Params:
  - page: 页码 (default: 1)
  - page_size: 每页数量 (default: 20)
  - status: 状态筛选 (optional)

Response 200:
{
  "code": 200,
  "data": {
    "total": 156,
    "page": 1,
    "page_size": 20,
    "tasks": [
      {
        "task_id": "xxx",
        "status": "completed",
        "prompt": "...",
        "output_image_url": "/static/images/xxx.webp",
        "created_at": "2025-11-09T10:30:00Z"
      },
      ...
    ]
  }
}
```

#### 6.1.5 取消任务

```http
DELETE /api/v1/generation/tasks/{task_id}
Authorization: Bearer {jwt_token}

Response 200:
{
  "code": 200,
  "message": "任务已取消"
}

Response 400:
{
  "code": 400,
  "message": "任务已开始处理,无法取消"
}
```

### 6.2 模型相关接口

#### 6.2.1 获取可用模型列表

```http
GET /api/v1/generation/models?type=text2img
Authorization: Bearer {jwt_token} (可选)

Query Params:
  - type: 模型类型 (text2img/img2img)

Response 200:
{
  "code": 200,
  "data": {
    "models": [
      {
        "model_id": "sd-v1.5",
        "model_name": "Stable Diffusion 1.5",
        "model_type": "text2img",
        "description": "经典的SD1.5模型,速度快,质量稳定",
        "default_width": 512,
        "default_height": 512,
        "max_width": 1024,
        "max_height": 1024,
        "tags": ["推荐", "快速"],
        "is_recommended": true
      },
      ...
    ]
  }
}
```

### 6.3 WebSocket接口

#### 6.3.1 建立连接

```
WebSocket: ws://api.example.com/ws?token={jwt_token}

连接成功后服务端推送:
{
  "type": "connected",
  "user_uuid": "xxx",
  "timestamp": "2025-11-09T10:30:00Z"
}
```

#### 6.3.2 实时消息格式

```json
// 任务进度推送
{
  "type": "task_progress",
  "task_id": "xxx",
  "status": "processing",
  "message": "正在生成图片..."
}

// 任务完成推送
{
  "type": "task_completed",
  "task_id": "xxx",
  "status": "completed",
  "output_image_url": "/static/images/xxx.webp",
  "generation_time_ms": 3500
}

// 任务失败推送
{
  "type": "task_failed",
  "task_id": "xxx",
  "status": "failed",
  "error_message": "模型调用失败"
}
```

---

## 七、目录结构

```
AI-Nexus/
├── cmd/
│   ├── ainexus.go              # HTTP服务入口 (现有)
│   └── worker/
│       └── main.go             # Worker独立进程入口 (新增)
│
├── internal/
│   ├── controller/
│   │   ├── user/               # 现有用户控制器
│   │   └── generation/         # 新增
│   │       ├── generation-controller.go
│   │       ├── task-controller.go
│   │       └── websocket-controller.go
│   │
│   ├── service/
│   │   ├── user/               # 现有用户服务
│   │   └── generation/         # 新增
│   │       ├── generation-service.go
│   │       ├── task-service.go
│   │       └── modelscope-client.go
│   │
│   ├── dao/
│   │   ├── user/               # 现有用户DAO
│   │   └── generation/         # 新增
│   │       ├── task-dao.go
│   │       └── model-dao.go
│   │
│   ├── domain/
│   │   ├── do/
│   │   │   └── generation/     # 新增
│   │   │       ├── task-do.go
│   │   │       └── model-do.go
│   │   ├── dto/
│   │   │   └── generation/     # 新增
│   │   │       ├── text2img-dto.go
│   │   │       └── img2img-dto.go
│   │   ├── vo/
│   │   │   └── generation/     # 新增
│   │   │       ├── task-vo.go
│   │   │       └── model-vo.go
│   │   └── query/
│   │       └── generation/     # 新增
│   │           └── task-query.go
│   │
│   ├── routes/
│   │   ├── user/               # 现有用户路由
│   │   └── generation/         # 新增
│   │       └── generation-routes.go
│   │
│   ├── middleware/             # 现有中间件(完全复用)
│   │   ├── auth.go
│   │   ├── ratelimit.go
│   │   ├── deduplication.go
│   │   └── ...
│   │
│   ├── pkg/
│   │   ├── db/                 # 现有DB (复用)
│   │   ├── rdb/                # 现有Redis (复用)
│   │   ├── logger/             # 现有Logger (复用)
│   │   ├── queue/              # 新增
│   │   │   ├── rabbitmq.go     # RabbitMQ连接管理
│   │   │   ├── producer.go     # 消息生产者
│   │   │   └── consumer.go     # 消息消费者
│   │   ├── websocket/          # 新增
│   │   │   ├── hub.go          # WebSocket Hub
│   │   │   └── client.go       # WebSocket Client
│   │   └── storage/            # 新增
│   │       └── local.go        # 本地文件存储
│   │
│   └── worker/                 # 新增
│       ├── text2img-worker.go
│       └── img2img-worker.go
│
├── configs/
│   ├── config.yaml             # 配置文件 (新增RabbitMQ/ModelScope配置)
│   └── db.sql                  # 数据库脚本 (新增表)
│
├── docs/
│   └── devel/
│       └── design/
│           └── ai-generation-core-design.md  # 本文档
│
└── static/
    └── images/                 # 生成图片存储目录 (新增)
```

---

## 八、ModelScope集成方案

### 8.1 ModelScope API概述

ModelScope是阿里达摩院开源的AI模型平台,提供以下能力:

1. **文生图**: 基于Stable Diffusion系列模型
2. **图生图**: 图像风格迁移、图像编辑
3. **免费额度**: 提供一定的免费调用量
4. **API简单**: RESTful API,易于集成

### 8.2 API认证

```go
// ModelScope API需要API Key
const (
    ModelScopeAPIURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis"
    APIKey           = "your-api-key-here" // 从配置文件读取
)

// 请求头
headers := map[string]string{
    "Authorization": fmt.Sprintf("Bearer %s", APIKey),
    "Content-Type":  "application/json",
}
```

### 8.3 文生图API调用

```go
// ModelScope文生图请求
type Text2ImgRequest struct {
    Model string                 `json:"model"`
    Input Text2ImgInput          `json:"input"`
    Parameters Text2ImgParameters `json:"parameters"`
}

type Text2ImgInput struct {
    Prompt         string `json:"prompt"`
    NegativePrompt string `json:"negative_prompt,omitempty"`
}

type Text2ImgParameters struct {
    Size              string  `json:"size"`              // "512*512", "1024*1024"
    NumInferenceSteps int     `json:"n"`                 // 推理步数
    Seed              int64   `json:"seed,omitempty"`    // 随机种子
    GuidanceScale     float64 `json:"guidance_scale"`    // CFG Scale
}

// 示例调用
func CallModelScopeText2Img(task *GenerationTaskDO) (string, error) {
    request := Text2ImgRequest{
        Model: "stable-diffusion-v1.5",
        Input: Text2ImgInput{
            Prompt:         task.Prompt,
            NegativePrompt: task.NegativePrompt,
        },
        Parameters: Text2ImgParameters{
            Size:              fmt.Sprintf("%d*%d", task.Width, task.Height),
            NumInferenceSteps: task.NumInferenceSteps,
            Seed:              task.Seed,
            GuidanceScale:     task.GuidanceScale,
        },
    }

    // 发送HTTP POST请求
    response, err := http.Post(ModelScopeAPIURL, headers, request)
    if err != nil {
        return "", err
    }

    // 解析响应
    var result struct {
        Output struct {
            Results []struct {
                URL string `json:"url"` // 图片URL
            } `json:"results"`
        } `json:"output"`
    }

    json.Unmarshal(response.Body, &result)
    return result.Output.Results[0].URL, nil
}
```

### 8.4 图生图API调用

```go
// 图生图请求 (需要上传图片)
type Img2ImgRequest struct {
    Model      string            `json:"model"`
    Input      Img2ImgInput      `json:"input"`
    Parameters Img2ImgParameters `json:"parameters"`
}

type Img2ImgInput struct {
    ImageURL       string `json:"image_url"`       // 原始图片URL
    Prompt         string `json:"prompt"`
    NegativePrompt string `json:"negative_prompt,omitempty"`
}

type Img2ImgParameters struct {
    Strength          float64 `json:"strength"`           // 0.0-1.0
    NumInferenceSteps int     `json:"n"`
    Seed              int64   `json:"seed,omitempty"`
}
```

### 8.5 错误处理

```go
// ModelScope错误码
const (
    ErrCodeInvalidRequest    = "InvalidParameter"
    ErrCodeQuotaExceeded     = "QuotaExceeded"
    ErrCodeContentFiltered   = "ContentFiltered"
    ErrCodeInternalError     = "InternalError"
)

// 错误处理逻辑
func HandleModelScopeError(errCode string) (canRetry bool) {
    switch errCode {
    case ErrCodeInvalidRequest:
        return false // 参数错误,不可重试
    case ErrCodeContentFiltered:
        return false // 内容违规,不可重试
    case ErrCodeQuotaExceeded:
        return false // 额度不足,不可重试
    case ErrCodeInternalError:
        return true // 服务器错误,可以重试
    default:
        return true // 未知错误,尝试重试
    }
}
```

### 8.6 ModelScope模型推荐

| 模型名称 | ModelScope ID | 适用场景 | 生成速度 | 质量 |
|---------|---------------|---------|---------|------|
| SD 1.5 | stable-diffusion-v1.5 | 通用场景 | 快 | 中 |
| SD 2.1 | stable-diffusion-2.1 | 高质量生成 | 中 | 高 |
| SDXL | stable-diffusion-xl | 超高清生成 | 慢 | 极高 |

---

## 九、WebSocket实时通知

### 9.1 WebSocket Hub设计

```go
// internal/pkg/websocket/hub.go
package websocket

import (
    "sync"
    "github.com/gorilla/websocket"
)

// Hub管理所有WebSocket连接
type Hub struct {
    // user_uuid -> *Client
    clients    map[string]*Client
    clientsMux sync.RWMutex

    // 广播通道
    broadcast chan Message

    // 注册/注销通道
    register   chan *Client
    unregister chan *Client
}

type Message struct {
    UserUUID string      `json:"-"`
    Type     string      `json:"type"`
    Data     interface{} `json:"data"`
}

var GlobalHub *Hub

func init() {
    GlobalHub = &Hub{
        clients:    make(map[string]*Client),
        broadcast:  make(chan Message, 256),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
    go GlobalHub.Run()
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clientsMux.Lock()
            h.clients[client.UserUUID] = client
            h.clientsMux.Unlock()

        case client := <-h.unregister:
            h.clientsMux.Lock()
            if _, ok := h.clients[client.UserUUID]; ok {
                delete(h.clients, client.UserUUID)
                close(client.send)
            }
            h.clientsMux.Unlock()

        case message := <-h.broadcast:
            h.clientsMux.RLock()
            if client, ok := h.clients[message.UserUUID]; ok {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, message.UserUUID)
                }
            }
            h.clientsMux.RUnlock()
        }
    }
}

// SendToUser 向指定用户推送消息
func (h *Hub) SendToUser(userUUID string, msgType string, data interface{}) {
    h.broadcast <- Message{
        UserUUID: userUUID,
        Type:     msgType,
        Data:     data,
    }
}
```

### 9.2 WebSocket Client

```go
// internal/pkg/websocket/client.go
package websocket

import (
    "time"
    "github.com/gorilla/websocket"
)

type Client struct {
    UserUUID string
    conn     *websocket.Conn
    send     chan Message
}

func NewClient(userUUID string, conn *websocket.Conn) *Client {
    return &Client{
        UserUUID: userUUID,
        conn:     conn,
        send:     make(chan Message, 256),
    }
}

// 读取消息(心跳检测)
func (c *Client) ReadPump() {
    defer func() {
        GlobalHub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, _, err := c.conn.ReadMessage()
        if err != nil {
            break
        }
    }
}

// 写入消息
func (c *Client) WritePump() {
    ticker := time.NewTicker(30 * time.Second)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            err := c.conn.WriteJSON(message)
            if err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

### 9.3 WebSocket Controller

```go
// internal/controller/generation/websocket-controller.go
package generation

import (
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "internal/pkg/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 生产环境需要校验Origin
    },
}

func HandleWebSocket(ctx *gin.Context) {
    // 从query或header获取JWT token
    token := ctx.Query("token")

    // 解析token获取user_uuid
    userUUID, err := parseJWT(token)
    if err != nil {
        ctx.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // 升级HTTP连接为WebSocket
    conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
    if err != nil {
        return
    }

    // 创建客户端
    client := websocket.NewClient(userUUID, conn)

    // 注册到Hub
    websocket.GlobalHub.Register <- client

    // 发送连接成功消息
    websocket.GlobalHub.SendToUser(userUUID, "connected", map[string]interface{}{
        "user_uuid": userUUID,
        "timestamp": time.Now(),
    })

    // 启动读写协程
    go client.WritePump()
    go client.ReadPump()
}
```

### 9.4 Worker中推送消息

```go
// Worker处理完成后推送
func (w *Text2ImgWorker) pushTaskCompleted(task *GenerationTaskDO) {
    websocket.GlobalHub.SendToUser(task.UserUUID, "task_completed", map[string]interface{}{
        "task_id":           task.TaskID,
        "status":            "completed",
        "output_image_url":  task.OutputImageURL,
        "generation_time_ms": task.GenerationTimeMs,
    })
}

// Worker处理失败后推送
func (w *Text2ImgWorker) pushTaskFailed(task *GenerationTaskDO, errMsg string) {
    websocket.GlobalHub.SendToUser(task.UserUUID, "task_failed", map[string]interface{}{
        "task_id":       task.TaskID,
        "status":        "failed",
        "error_message": errMsg,
    })
}
```

---

## 十、扩展性设计

### 10.1 水平扩展

#### 10.1.1 HTTP服务扩展
- Nginx负载均衡
- 多实例部署
- Session存储在Redis
- WebSocket支持Sticky Session

#### 10.1.2 Worker扩展
- 独立部署Worker进程
- 可按任务类型分配Worker
- 支持跨机器部署
- 动态调整并发数

#### 10.1.3 RabbitMQ集群
- 3节点镜像队列集群
- 自动故障转移
- 消息高可用

### 10.2 功能扩展

#### 10.2.1 多AI提供商支持
- 抽象AIProvider接口
- 支持切换到Stability AI、Midjourney等
- 失败自动切换备用Provider

#### 10.2.2 新功能模块
- ControlNet精准控制
- 图片放大 (Upscale)
- 局部重绘 (Inpaint)
- Lora模型训练

### 10.3 监控与告警

#### 10.3.1 业务指标
- 任务成功率
- 平均生成时长
- 队列积压数量
- API调用费用

#### 10.3.2 系统指标
- CPU/内存使用率
- RabbitMQ队列长度
- Redis缓存命中率
- 数据库慢查询

---

## 附录

### A. 配置文件示例

```yaml
# configs/config.yaml (新增部分)

# RabbitMQ配置
rabbitmq:
  host: localhost
  port: 5672
  user: guest
  password: guest
  vhost: /

# ModelScope配置
modelscope:
  api_url: https://dashscope.aliyuncs.com/api/v1/services/aigc/text2image/image-synthesis
  api_key: your-api-key-here
  timeout: 60s

# Worker配置
worker:
  text2img_concurrency: 3
  img2img_concurrency: 2
  retry_max: 3
  retry_delay: 10s
```

### B. 数据库迁移脚本

见 `configs/db.sql` 中新增的两张表

### C. RabbitMQ初始化

```bash
# 创建Exchange
rabbitmqadmin declare exchange name=generation.topic type=topic durable=true

# 创建队列
rabbitmqadmin declare queue name=queue.text2img durable=true \
  arguments='{"x-message-ttl":1800000,"x-max-length":1000,"x-dead-letter-exchange":"generation.dlx"}'

rabbitmqadmin declare queue name=queue.img2img durable=true \
  arguments='{"x-message-ttl":1800000,"x-max-length":1000,"x-dead-letter-exchange":"generation.dlx"}'

# 绑定队列到Exchange
rabbitmqadmin declare binding source=generation.topic \
  destination=queue.text2img routing_key=generation.text2img

rabbitmqadmin declare binding source=generation.topic \
  destination=queue.img2img routing_key=generation.img2img

# 创建死信队列
rabbitmqadmin declare exchange name=generation.dlx type=direct durable=true
rabbitmqadmin declare queue name=queue.dead_letter durable=true
rabbitmqadmin declare binding source=generation.dlx \
  destination=queue.dead_letter routing_key=dead_letter
```

---