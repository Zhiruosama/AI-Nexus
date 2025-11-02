# AI生图模块完整设计方案 (基于RabbitMQ)

> 作者: Claude
> 日期: 2025-11-02
> 版本: v1.0
> 技术栈: Go + RabbitMQ + MySQL + Redis + Stability AI

---

## 目录

- [一、系统架构设计](#一系统架构设计)
- [二、数据库设计](#二数据库设计)
- [三、RabbitMQ队列设计](#三rabbitmq队列设计)
- [四、业务流程设计](#四业务流程设计)
- [五、充值模块设计](#五充值模块设计)
- [六、接口设计](#六接口设计)
- [七、目录结构](#七目录结构)
- [八、技术选型说明](#八技术选型说明)
- [九、扩展性设计](#九扩展性设计)

---

## 一、系统架构设计

### 1.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                          Frontend 前端                            │
│  - 提交生图请求                                                    │
│  - WebSocket 实时接收进度                                          │
│  - 展示生成结果/历史记录                                            │
└────────────────────────┬────────────────────────────────────────┘
                         │ HTTP/WebSocket
┌────────────────────────▼────────────────────────────────────────┐
│                    Gin HTTP Server                               │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Middleware 中间件层                                        │   │
│  │  - JWT 认证                                               │   │
│  │  - 请求限流 (Redis)                                        │   │
│  │  - 请求去重                                                │   │
│  │  - 日志追踪 (RequestID)                                    │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Controller 控制器层                                        │   │
│  │  - GenerationController (生图接口)                        │   │
│  │  - CreditController (积分/充值接口)                        │   │
│  │  - ModelController (模型管理接口)                          │   │
│  └──────────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│                    Service 服务层                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Generation      │  │ Credit          │  │ Payment         │ │
│  │ Service         │  │ Service         │  │ Service         │ │
│  │                 │  │                 │  │                 │ │
│  │ - 创建任务       │  │ - 积分查询       │  │ - 支付宝/微信    │ │
│  │ - 参数校验       │  │ - 积分扣除       │  │ - 回调处理       │ │
│  │ - 发送到队列     │  │ - 积分充值       │  │ - 订单管理       │ │
│  │ - 任务查询       │  │ - 流水记录       │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└──────┬──────────────────┬──────────────────┬────────────────────┘
       │                  │                  │
       │                  │                  │
┌──────▼──────────────────▼──────────────────▼────────────────────┐
│                    RabbitMQ 消息队列                              │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Exchange: generation.topic (Topic 类型)                   │   │
│  │                                                           │   │
│  │  Routing Keys:                                            │   │
│  │   - generation.text2img    (文生图)                       │   │
│  │   - generation.img2img     (图生图)                       │   │
│  │   - generation.upscale     (图片放大)                     │   │
│  │   - generation.inpaint     (局部重绘)                     │   │
│  └──────────────────────────────────────────────────────────┘   │
│                              │                                   │
│                              ▼                                   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Queues (持久化队列)                                         │   │
│  │                                                           │   │
│  │  - queue.text2img       (优先级队列, 支持重试)             │   │
│  │  - queue.img2img                                          │   │
│  │  - queue.upscale                                          │   │
│  │  - queue.dead_letter    (死信队列, 处理失败消息)           │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ Consume (消费消息)
┌──────────────────────────▼──────────────────────────────────────┐
│                   Worker Pool 工作进程池                          │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Text2Img Workers (并发数: 5)                              │   │
│  │  - 消费 queue.text2img 消息                               │   │
│  │  - 调用 Stability AI API                                  │   │
│  │  - 保存图片到存储                                          │   │
│  │  - 更新任务状态到 MySQL                                    │   │
│  │  - WebSocket 推送进度                                      │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Img2Img Workers (并发数: 3)                               │   │
│  └──────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Upscale Workers (并发数: 2)                               │   │
│  └──────────────────────────────────────────────────────────┘   │
└──────────┬─────────────────────┬─────────────────────────────────┘
           │                     │
           │                     │
┌──────────▼─────────┐  ┌────────▼──────────┐
│  Stability AI SDK  │  │ Storage Service   │
│                    │  │                   │
│  - Text2Img API    │  │ - MinIO (对象存储) │
│  - Img2Img API     │  │ - 本地文件系统     │
│  - Upscale API     │  │ - CDN 加速        │
│  - 错误重试        │  │                   │
└──────────┬─────────┘  └────────┬──────────┘
           │                     │
           └──────────┬──────────┘
                      │
┌─────────────────────▼────────────────────────────────────────────┐
│                    数据持久层                                      │
│  ┌────────────────────┐          ┌──────────────────────┐        │
│  │  MySQL 数据库       │          │  Redis 缓存           │        │
│  │                    │          │                      │        │
│  │  - 用户表           │          │  - 任务状态缓存       │        │
│  │  - 积分表           │          │  - 用户积分缓存       │        │
│  │  - 生图任务表       │          │  - 限流计数器         │        │
│  │  - 积分流水表       │          │  - 分布式锁           │        │
│  │  - 充值订单表       │          │  - WebSocket Session │        │
│  │  - 模型配置表       │          │                      │        │
│  └────────────────────┘          └──────────────────────┘        │
└──────────────────────────────────────────────────────────────────┘
```

### 1.2 核心设计原则

#### 1.2.1 解耦设计
- HTTP 服务只负责接收请求和返回响应
- Worker 进程独立部署,可单独扩容
- RabbitMQ 作为中间件解耦生产者和消费者

#### 1.2.2 异步处理
- 用户提交请求后立即返回任务ID
- 实际生成过程在 Worker 中异步执行
- WebSocket 实时推送任务进度

#### 1.2.3 高可用设计
- RabbitMQ 持久化队列,防止消息丢失
- 死信队列处理失败任务
- Worker 支持优雅重启
- 数据库主从复制

#### 1.2.4 可扩展性
- Worker 支持水平扩容
- 支持多种 AI 模型提供商
- 模块化设计,便于添加新功能

---

## 二、数据库设计

### 2.1 核心表设计

#### 2.1.1 用户表 (已存在,无需修改)
```sql
-- 使用现有的 table_user 表
```

#### 2.1.2 生图任务表 (generation_tasks)

```sql
CREATE TABLE `generation_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_id` VARCHAR(64) NOT NULL COMMENT '任务UUID (对外暴露)',
  `user_uuid` VARCHAR(36) NOT NULL COMMENT '用户UUID (关联 table_user.uuid)',

  -- 任务基本信息
  `task_type` TINYINT UNSIGNED NOT NULL COMMENT '任务类型: 1-文生图, 2-图生图, 3-放大, 4-局部重绘',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '任务状态: 0-待处理, 1-队列中, 2-处理中, 3-已完成, 4-失败, 5-已取消',
  `priority` TINYINT UNSIGNED NOT NULL DEFAULT 5 COMMENT '优先级: 1-最高, 10-最低 (VIP用户优先级高)',

  -- 输入参数 (JSON存储灵活配置)
  `prompt` TEXT NOT NULL COMMENT '正向提示词 (最长10000字符)',
  `negative_prompt` TEXT COMMENT '负向提示词',
  `model_id` VARCHAR(64) NOT NULL COMMENT '模型ID (关联 generation_models.model_id)',
  `style_preset` VARCHAR(32) COMMENT '风格预设: photographic/digital-art/anime 等',
  `aspect_ratio` VARCHAR(16) DEFAULT '1:1' COMMENT '宽高比: 1:1, 16:9, 9:16, 3:2, 2:3 等',
  `seed` BIGINT UNSIGNED COMMENT '随机种子 (用户指定或自动生成)',
  `output_format` VARCHAR(10) DEFAULT 'png' COMMENT '输出格式: png/jpeg/webp',

  -- 图生图专用参数
  `input_image_url` VARCHAR(512) COMMENT '输入图片URL (图生图/放大/重绘场景)',
  `strength` DECIMAL(3,2) COMMENT '强度 0.00-1.00 (图生图场景, 越高变化越大)',
  `mask_image_url` VARCHAR(512) COMMENT '蒙版图片URL (局部重绘场景)',

  -- 高级参数 (存储为JSON,便于扩展)
  `advanced_params` JSON COMMENT '高级参数: {"steps": 30, "cfg_scale": 7.0, "sampler": "k_dpmpp_2m", "clip_guidance": 0}',

  -- 输出结果
  `output_image_url` VARCHAR(512) COMMENT '生成的图片URL',
  `output_width` INT UNSIGNED COMMENT '输出图片宽度(像素)',
  `output_height` INT UNSIGNED COMMENT '输出图片高度(像素)',
  `output_seed` BIGINT UNSIGNED COMMENT '实际使用的种子值',
  `finish_reason` VARCHAR(32) COMMENT '完成原因: SUCCESS/CONTENT_FILTERED/ERROR',

  -- 错误处理
  `error_code` VARCHAR(32) COMMENT '错误代码',
  `error_message` TEXT COMMENT '错误详情',
  `retry_count` TINYINT UNSIGNED DEFAULT 0 COMMENT '已重试次数',
  `max_retry` TINYINT UNSIGNED DEFAULT 3 COMMENT '最大重试次数',

  -- 费用与性能
  `cost_credits` INT UNSIGNED NOT NULL COMMENT '消耗积分',
  `api_cost_usd` DECIMAL(10,6) COMMENT 'API实际费用(美元)',
  `generation_time_ms` INT UNSIGNED COMMENT '生成耗时(毫秒)',
  `queue_time_ms` INT UNSIGNED COMMENT '队列等待时长(毫秒)',

  -- RabbitMQ 相关
  `mq_message_id` VARCHAR(64) COMMENT 'RabbitMQ消息ID',
  `mq_delivery_tag` BIGINT UNSIGNED COMMENT 'RabbitMQ投递标签',

  -- 审核与安全
  `content_filtered` BOOLEAN DEFAULT FALSE COMMENT '是否触发内容过滤',
  `is_public` BOOLEAN DEFAULT FALSE COMMENT '是否公开展示',
  `is_deleted` BOOLEAN DEFAULT FALSE COMMENT '软删除标记',

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
  KEY `idx_created_at` (`created_at`),
  KEY `idx_user_status` (`user_uuid`, `status`),
  KEY `idx_mq_message_id` (`mq_message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI生图任务表';
```

#### 2.1.3 用户积分表 (user_credits)

```sql
CREATE TABLE `user_credits` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_uuid` VARCHAR(36) NOT NULL COMMENT '用户UUID',

  -- 积分账户
  `balance` INT NOT NULL DEFAULT 0 COMMENT '当前积分余额',
  `frozen_balance` INT NOT NULL DEFAULT 0 COMMENT '冻结积分 (处理中任务占用)',
  `total_earned` BIGINT NOT NULL DEFAULT 0 COMMENT '累计获得积分',
  `total_spent` BIGINT NOT NULL DEFAULT 0 COMMENT '累计消费积分',
  `total_recharged` BIGINT NOT NULL DEFAULT 0 COMMENT '累计充值积分',

  -- 等级与权益
  `level` TINYINT UNSIGNED DEFAULT 1 COMMENT '用户等级: 1-普通, 2-VIP, 3-SVIP',
  `daily_free_quota` INT DEFAULT 10 COMMENT '每日免费额度',
  `daily_used_quota` INT DEFAULT 0 COMMENT '今日已用免费额度',
  `quota_reset_at` DATE COMMENT '额度重置日期',

  -- 统计信息
  `total_tasks` INT UNSIGNED DEFAULT 0 COMMENT '累计生图次数',
  `success_tasks` INT UNSIGNED DEFAULT 0 COMMENT '成功生图次数',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_uuid` (`user_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户积分账户表';
```

#### 2.1.4 积分流水表 (credit_transactions)

```sql
CREATE TABLE `credit_transactions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `transaction_id` VARCHAR(64) NOT NULL COMMENT '流水号UUID',
  `user_uuid` VARCHAR(36) NOT NULL COMMENT '用户UUID',

  -- 交易类型
  `type` TINYINT UNSIGNED NOT NULL COMMENT '交易类型: 1-充值, 2-消费, 3-退款, 4-赠送, 5-过期扣除, 6-系统调整',
  `amount` INT NOT NULL COMMENT '变动数量 (正数=增加, 负数=减少)',
  `balance_before` INT NOT NULL COMMENT '变动前余额',
  `balance_after` INT NOT NULL COMMENT '变动后余额',

  -- 关联信息
  `related_task_id` VARCHAR(64) COMMENT '关联任务ID (type=2时)',
  `related_order_id` VARCHAR(64) COMMENT '关联订单ID (type=1时)',

  -- 描述信息
  `title` VARCHAR(128) NOT NULL COMMENT '流水标题',
  `description` VARCHAR(512) COMMENT '详细描述',
  `remark` VARCHAR(255) COMMENT '备注',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_transaction_id` (`transaction_id`),
  KEY `idx_user_uuid` (`user_uuid`),
  KEY `idx_created_at` (`created_at`),
  KEY `idx_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='积分流水表';
```

#### 2.1.5 充值订单表 (recharge_orders)

```sql
CREATE TABLE `recharge_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `order_id` VARCHAR(64) NOT NULL COMMENT '订单号 (自生成)',
  `user_uuid` VARCHAR(36) NOT NULL COMMENT '用户UUID',

  -- 订单信息
  `package_id` INT UNSIGNED COMMENT '充值套餐ID (关联 recharge_packages)',
  `credits_amount` INT NOT NULL COMMENT '充值积分数量',
  `payment_amount` DECIMAL(10,2) NOT NULL COMMENT '支付金额 (元)',
  `currency` VARCHAR(10) DEFAULT 'CNY' COMMENT '货币: CNY/USD',

  -- 支付信息
  `payment_method` VARCHAR(32) NOT NULL COMMENT '支付方式: alipay/wechat/stripe',
  `payment_channel` VARCHAR(32) COMMENT '支付渠道: web/app/h5',
  `third_party_order_id` VARCHAR(128) COMMENT '第三方订单号',

  -- 订单状态
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '订单状态: 0-待支付, 1-支付中, 2-支付成功, 3-支付失败, 4-已退款, 5-已取消',
  `paid_at` DATETIME COMMENT '支付完成时间',
  `refund_at` DATETIME COMMENT '退款时间',

  -- 回调信息
  `notify_url` VARCHAR(512) COMMENT '异步通知URL',
  `notify_count` TINYINT UNSIGNED DEFAULT 0 COMMENT '通知次数',
  `notify_success` BOOLEAN DEFAULT FALSE COMMENT '通知成功标记',

  -- IP与设备
  `client_ip` VARCHAR(64) COMMENT '客户端IP',
  `user_agent` VARCHAR(512) COMMENT '用户代理',

  -- 备注
  `remark` VARCHAR(512) COMMENT '备注',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `expired_at` DATETIME COMMENT '订单过期时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_id` (`order_id`),
  KEY `idx_user_uuid` (`user_uuid`),
  KEY `idx_status` (`status`),
  KEY `idx_third_party_order_id` (`third_party_order_id`),
  KEY `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='充值订单表';
```

#### 2.1.6 充值套餐表 (recharge_packages)

```sql
CREATE TABLE `recharge_packages` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `package_id` VARCHAR(32) NOT NULL COMMENT '套餐标识',

  -- 套餐内容
  `name` VARCHAR(64) NOT NULL COMMENT '套餐名称',
  `credits_amount` INT NOT NULL COMMENT '积分数量',
  `price` DECIMAL(10,2) NOT NULL COMMENT '价格 (元)',
  `original_price` DECIMAL(10,2) COMMENT '原价 (用于显示折扣)',
  `bonus_credits` INT DEFAULT 0 COMMENT '赠送积分',

  -- 套餐属性
  `description` VARCHAR(512) COMMENT '套餐描述',
  `tags` JSON COMMENT '标签: ["热门", "限时优惠"]',
  `sort_order` INT DEFAULT 0 COMMENT '排序权重 (数字越大越靠前)',

  -- 限制条件
  `min_level` TINYINT UNSIGNED DEFAULT 1 COMMENT '最低用户等级要求',
  `is_first_recharge_only` BOOLEAN DEFAULT FALSE COMMENT '是否仅限首充',
  `stock` INT COMMENT '库存数量 (NULL=无限)',
  `daily_limit` INT COMMENT '每日限购数量',

  -- 状态
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否启用',
  `is_recommended` BOOLEAN DEFAULT FALSE COMMENT '是否推荐',

  -- 时间限制
  `valid_from` DATETIME COMMENT '生效开始时间',
  `valid_until` DATETIME COMMENT '生效结束时间',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_package_id` (`package_id`),
  KEY `idx_is_active` (`is_active`),
  KEY `idx_sort_order` (`sort_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='充值套餐表';
```

#### 2.1.7 模型配置表 (generation_models)

```sql
CREATE TABLE `generation_models` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `model_id` VARCHAR(64) NOT NULL COMMENT '模型标识 (如: sd3-large)',

  -- 基本信息
  `model_name` VARCHAR(128) NOT NULL COMMENT '模型显示名称',
  `provider` VARCHAR(32) NOT NULL DEFAULT 'stability' COMMENT '提供商: stability/midjourney/dalle',
  `version` VARCHAR(32) COMMENT '版本号',

  -- 模型类型
  `type` VARCHAR(32) NOT NULL COMMENT '类型: text2img/img2img/upscale/inpaint',
  `category` VARCHAR(32) COMMENT '分类: general/anime/realistic/artistic',

  -- 能力参数
  `max_resolution` VARCHAR(16) COMMENT '最大分辨率 (如: 1024x1024)',
  `supported_aspect_ratios` JSON COMMENT '支持的宽高比: ["1:1", "16:9", "9:16"]',
  `supported_output_formats` JSON COMMENT '支持的输出格式: ["png", "jpeg", "webp"]',

  -- 定价
  `cost_per_image` INT NOT NULL COMMENT '每张图消耗积分',
  `api_cost_usd` DECIMAL(10,6) COMMENT 'API成本 (美元/张)',

  -- 性能指标
  `avg_generation_time_ms` INT UNSIGNED COMMENT '平均生成耗时(毫秒)',
  `quality_score` DECIMAL(3,2) COMMENT '质量评分 0.00-10.00',

  -- 显示与排序
  `description` TEXT COMMENT '模型描述',
  `preview_image_url` VARCHAR(512) COMMENT '预览图URL',
  `tags` JSON COMMENT '标签: ["快速", "高质量", "动漫风格"]',
  `sort_order` INT DEFAULT 0 COMMENT '排序权重',

  -- 限制条件
  `min_user_level` TINYINT UNSIGNED DEFAULT 1 COMMENT '最低用户等级',
  `is_beta` BOOLEAN DEFAULT FALSE COMMENT '是否为测试版',

  -- 状态
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否启用',
  `is_recommended` BOOLEAN DEFAULT FALSE COMMENT '是否推荐',

  -- 统计信息
  `total_usage` BIGINT UNSIGNED DEFAULT 0 COMMENT '累计使用次数',
  `success_rate` DECIMAL(5,2) COMMENT '成功率百分比',

  -- 时间戳
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_model_id` (`model_id`),
  KEY `idx_type` (`type`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='生图模型配置表';
```

### 2.2 表关系图

```
table_user (现有用户表)
    │
    │ 1:1
    ▼
user_credits (积分账户)
    │
    │ 1:N
    ├──────────────────┬──────────────────┐
    │                  │                  │
    ▼                  ▼                  ▼
generation_tasks  credit_transactions  recharge_orders
    │                                     │
    │ N:1                                 │ N:1
    ▼                                     ▼
generation_models                   recharge_packages
```

### 2.3 索引优化策略

#### 高频查询场景:
1. **按用户查询任务列表**: `idx_user_status (user_uuid, status)`
2. **按状态查询待处理任务**: `idx_status (status)`
3. **按时间查询历史记录**: `idx_created_at (created_at)`
4. **积分流水查询**: `idx_user_uuid (user_uuid)` + `idx_created_at (created_at)`

---

## 三、RabbitMQ队列设计

### 3.1 Exchange 设计

```
类型: Topic Exchange
名称: generation.topic
持久化: true
自动删除: false
```

### 3.2 Routing Key 规范

```
格式: generation.{task_type}.{priority}

示例:
- generation.text2img.high      (文生图, 高优先级)
- generation.text2img.normal    (文生图, 普通优先级)
- generation.img2img.high       (图生图, 高优先级)
- generation.upscale.normal     (放大, 普通优先级)
```

### 3.3 队列设计

#### 3.3.1 主业务队列

```yaml
queue.text2img:
  durable: true                    # 持久化
  auto_delete: false
  exclusive: false
  arguments:
    x-max-priority: 10             # 支持优先级 1-10
    x-message-ttl: 1800000         # 消息TTL: 30分钟
    x-max-length: 10000            # 最大队列长度
    x-dead-letter-exchange: generation.dlx
    x-dead-letter-routing-key: dead_letter

queue.img2img:
  # 配置同上

queue.upscale:
  # 配置同上
```

#### 3.3.2 死信队列 (Dead Letter Queue)

```yaml
exchange: generation.dlx (Direct Exchange)

queue.dead_letter:
  durable: true
  arguments:
    x-message-ttl: 604800000       # 7天后自动删除
```

**死信场景:**
1. 消息被拒绝 (basic.reject / basic.nack, requeue=false)
2. 消息TTL过期
3. 队列达到最大长度

#### 3.3.3 延迟重试队列

```yaml
queue.retry_delay:
  arguments:
    x-message-ttl: 60000           # 延迟60秒后重新投递
    x-dead-letter-exchange: generation.topic
    x-dead-letter-routing-key: generation.text2img.normal
```

### 3.4 消息格式

```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_uuid": "user-uuid-here",
  "task_type": 1,
  "retry_count": 0,
  "created_at": "2025-11-02T15:30:00Z",

  "payload": {
    "prompt": "A majestic cat wearing a wizard hat",
    "negative_prompt": "blurry, low quality",
    "model_id": "sd3-large",
    "aspect_ratio": "1:1",
    "seed": 1234567890
  }
}
```

### 3.5 消息确认机制

```
生产者确认 (Publisher Confirms):
- 启用 channel.ConfirmSelect()
- 等待 ack 后才认为发送成功

消费者确认 (Consumer Acknowledgements):
- 手动确认模式 (auto_ack=false)
- 处理成功: channel.Ack()
- 处理失败但可重试: channel.Nack(requeue=true)
- 处理失败不可重试: channel.Nack(requeue=false) → 进入死信队列
```

---

## 四、业务流程设计

### 4.1 文生图完整流程

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 用户提交请求阶段                                           │
└─────────────────────────────────────────────────────────────┘

  用户 → POST /api/v1/generation/text2img
         {
           "prompt": "...",
           "model": "sd3-large",
           ...
         }
         +
         JWT Token (从中间件获取 user_uuid)

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 2. Controller 层处理                                         │
└─────────────────────────────────────────────────────────────┘

  GenerationController.Text2Img()
    │
    ├─ 1. 参数校验 (Gin Validator)
    │    - prompt 长度检查
    │    - model_id 是否有效
    │    - aspect_ratio 格式检查
    │
    ├─ 2. 调用 Service 层
    │
    └─ 3. 返回响应
         {
           "code": 200,
           "data": {
             "task_id": "xxx",
             "status": "queued"
           }
         }

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 3. Service 层处理                                            │
└─────────────────────────────────────────────────────────────┘

  GenerationService.CreateText2ImgTask()
    │
    ├─ 1. 查询用户积分 (Redis缓存 + MySQL)
    │    SELECT balance FROM user_credits WHERE user_uuid = ?
    │
    ├─ 2. 查询模型定价
    │    SELECT cost_per_image FROM generation_models WHERE model_id = ?
    │
    ├─ 3. 检查积分是否足够
    │    IF balance < cost_per_image THEN
    │      RETURN Error("积分不足")
    │
    ├─ 4. 生成 task_id (UUID)
    │
    ├─ 5. 【事务开始】
    │    │
    │    ├─ 5.1 创建任务记录
    │    │    INSERT INTO generation_tasks (...)
    │    │    VALUES (task_id, user_uuid, status=1, ...)
    │    │
    │    ├─ 5.2 冻结积分
    │    │    UPDATE user_credits
    │    │    SET balance = balance - cost,
    │    │        frozen_balance = frozen_balance + cost
    │    │    WHERE user_uuid = ?
    │    │
    │    ├─ 5.3 记录积分流水
    │    │    INSERT INTO credit_transactions (...)
    │    │    VALUES (type=2, amount=-cost, ...)
    │    │
    │    └─ 【事务提交】
    │
    ├─ 6. 发送消息到 RabbitMQ
    │    │
    │    ├─ 构造消息体
    │    │    {
    │    │      "task_id": "xxx",
    │    │      "user_uuid": "yyy",
    │    │      "payload": {...}
    │    │    }
    │    │
    │    ├─ 发布到 Exchange
    │    │    channel.Publish(
    │    │      exchange: "generation.topic",
    │    │      routing_key: "generation.text2img.normal",
    │    │      body: json,
    │    │      properties: {
    │    │        delivery_mode: 2,  // 持久化
    │    │        priority: 5,
    │    │        message_id: task_id
    │    │      }
    │    │    )
    │    │
    │    └─ 等待 Publisher Confirm (ack)
    │
    ├─ 7. 更新任务状态
    │    UPDATE generation_tasks
    │    SET status=1, queued_at=NOW()
    │    WHERE task_id = ?
    │
    └─ 8. 缓存任务状态到 Redis
         SET task:status:{task_id} "queued" EX 3600

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 4. Worker 消费消息                                           │
└─────────────────────────────────────────────────────────────┘

  Text2ImgWorker.HandleMessage(msg)
    │
    ├─ 1. 反序列化消息
    │    task := json.Unmarshal(msg.Body)
    │
    ├─ 2. 从数据库获取完整任务信息
    │    SELECT * FROM generation_tasks WHERE task_id = ?
    │
    ├─ 3. 检查任务状态
    │    IF status != 1 THEN  // 不是"队列中"状态
    │      msg.Ack()          // 确认消息(防止重复处理)
    │      RETURN
    │
    ├─ 4. 更新状态为"处理中"
    │    UPDATE generation_tasks
    │    SET status=2, started_at=NOW()
    │    WHERE task_id = ?
    │
    ├─ 5. WebSocket 推送进度 (可选)
    │    ws.Send(user_uuid, {
    │      "task_id": "xxx",
    │      "status": "processing",
    │      "progress": 10
    │    })
    │
    ├─ 6. 调用 Stability AI API
    │    │
    │    ├─ 构造请求参数
    │    │
    │    ├─ 发送 HTTP POST 请求
    │    │    POST https://api.stability.ai/v2beta/stable-image/generate/sd3
    │    │    Headers: Authorization: Bearer {api_key}
    │    │    Body: multipart/form-data {...}
    │    │
    │    ├─ 处理响应
    │    │    IF status != 200 THEN
    │    │      → 进入错误处理流程
    │    │
    │    └─ 获取生成的图片 (Base64)
    │
    ├─ 7. 保存图片到存储
    │    │
    │    ├─ 解码 Base64
    │    │    imageData := base64.Decode(response.Image)
    │    │
    │    ├─ 生成文件名
    │    │    filename := fmt.Sprintf("%s.png", task_id)
    │    │
    │    ├─ 上传到 MinIO / 保存到本地
    │    │    url := storage.SaveImage(filename, imageData)
    │    │
    │    └─ 获取可访问的 URL
    │         output_url := "https://cdn.example.com/xxx.png"
    │
    ├─ 8. 更新任务为"已完成"
    │    UPDATE generation_tasks
    │    SET status=3,
    │        output_image_url=?,
    │        output_seed=?,
    │        generation_time_ms=?,
    │        completed_at=NOW()
    │    WHERE task_id = ?
    │
    ├─ 9. 解冻积分 (已在创建时扣除,这里只是更新状态)
    │    UPDATE user_credits
    │    SET frozen_balance = frozen_balance - cost
    │    WHERE user_uuid = ?
    │
    ├─ 10. WebSocket 推送完成通知
    │     ws.Send(user_uuid, {
    │       "task_id": "xxx",
    │       "status": "completed",
    │       "output_url": "https://..."
    │     })
    │
    └─ 11. 确认消息
         msg.Ack()

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 5. 用户查询结果                                              │
└─────────────────────────────────────────────────────────────┘

  方式1: 轮询
    GET /api/v1/generation/tasks/{task_id}

    GenerationService.GetTaskStatus()
      │
      ├─ 1. 先查 Redis 缓存
      │    status := Redis.Get("task:status:" + task_id)
      │
      ├─ 2. 缓存未命中则查数据库
      │    SELECT * FROM generation_tasks WHERE task_id = ?
      │
      └─ 3. 返回任务详情

  方式2: WebSocket 推送 (推荐)
    用户建立 WebSocket 连接
    Worker 完成后主动推送结果
    无需轮询,实时性更好
```

### 4.2 错误处理流程

```
┌─────────────────────────────────────────────────────────────┐
│ Worker 处理失败场景                                          │
└─────────────────────────────────────────────────────────────┘

IF API调用失败 THEN
  │
  ├─ 判断错误类型
  │
  ├─ 可重试错误 (网络超时、429限流、5xx服务器错误)
  │   │
  │   ├─ retry_count < max_retry (3次)
  │   │   │
  │   │   ├─ 更新重试次数
  │   │   │    UPDATE generation_tasks
  │   │   │    SET retry_count = retry_count + 1
  │   │   │
  │   │   ├─ Nack消息 (requeue=false)
  │   │   │
  │   │   ├─ 发送到延迟重试队列
  │   │   │    Publish to queue.retry_delay
  │   │   │    (60秒后自动重新投递到主队列)
  │   │   │
  │   │   └─ RETURN
  │   │
  │   └─ retry_count >= max_retry
  │       │
  │       └─ 标记为"失败"
  │
  └─ 不可重试错误 (400参数错误、CONTENT_FILTERED)
      │
      ├─ 更新任务状态
      │    UPDATE generation_tasks
      │    SET status=4,
      │        error_code=?,
      │        error_message=?,
      │        completed_at=NOW()
      │
      ├─ 退款积分
      │    UPDATE user_credits
      │    SET balance = balance + cost,
      │        frozen_balance = frozen_balance - cost
      │
      ├─ 记录退款流水
      │    INSERT INTO credit_transactions (type=3, ...)
      │
      ├─ WebSocket 推送失败通知
      │
      └─ 消息进入死信队列
           msg.Nack(requeue=false)
```

### 4.3 充值流程

```
┌─────────────────────────────────────────────────────────────┐
│ 1. 用户选择充值套餐                                          │
└─────────────────────────────────────────────────────────────┘

用户 → GET /api/v1/credit/packages
       返回套餐列表
  │
  ▼
用户选择套餐 → POST /api/v1/credit/recharge
               {
                 "package_id": "pkg_100",
                 "payment_method": "alipay"
               }
  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 2. 创建充值订单                                              │
└─────────────────────────────────────────────────────────────┘

PaymentService.CreateRechargeOrder()
  │
  ├─ 1. 查询套餐信息
  │    SELECT * FROM recharge_packages WHERE package_id = ?
  │
  ├─ 2. 生成订单号
  │    order_id := "RO" + timestamp + random
  │
  ├─ 3. 创建订单记录
  │    INSERT INTO recharge_orders (
  │      order_id,
  │      user_uuid,
  │      package_id,
  │      credits_amount,
  │      payment_amount,
  │      payment_method,
  │      status=0,  // 待支付
  │      expired_at=NOW() + 30分钟
  │    )
  │
  ├─ 4. 调用支付接口
  │    │
  │    ├─ 支付宝
  │    │    AlipayClient.TradePagePay(
  │    │      out_trade_no: order_id,
  │    │      total_amount: payment_amount,
  │    │      subject: "积分充值",
  │    │      notify_url: "https://api.xxx.com/callback/alipay"
  │    │    )
  │    │    → 返回支付页面URL
  │    │
  │    └─ 微信支付
  │         WechatPayClient.UnifiedOrder(...)
  │         → 返回支付二维码
  │
  └─ 5. 返回支付信息给前端
       {
         "order_id": "ROxxx",
         "payment_url": "https://openapi.alipay.com/...",
         "qr_code": "weixin://wxpay/...",
         "expired_at": "2025-11-02T16:00:00Z"
       }

  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 3. 用户完成支付                                              │
└─────────────────────────────────────────────────────────────┘

用户扫码/跳转支付 → 第三方支付平台
  │
  ▼
支付成功 → 支付宝/微信 异步回调
           POST https://api.xxx.com/callback/alipay
           {
             "out_trade_no": "ROxxx",
             "trade_no": "2025110222001...",
             "trade_status": "TRADE_SUCCESS",
             ...
           }
  │
  ▼

┌─────────────────────────────────────────────────────────────┐
│ 4. 处理支付回调                                              │
└─────────────────────────────────────────────────────────────┘

PaymentController.AlipayCallback()
  │
  ├─ 1. 验证签名
  │    IF !VerifySign(params, sign) THEN
  │      RETURN "fail"
  │
  ├─ 2. 查询订单
  │    SELECT * FROM recharge_orders WHERE order_id = ?
  │
  ├─ 3. 检查订单状态
  │    IF status != 0 THEN  // 已处理过
  │      RETURN "success"   // 幂等性
  │
  ├─ 4. 【分布式锁】防止并发
  │    lock := Redis.SetNX("lock:order:" + order_id, 1, 10s)
  │    IF !lock THEN RETURN "processing"
  │
  ├─ 5. 【事务开始】
  │    │
  │    ├─ 5.1 更新订单状态
  │    │    UPDATE recharge_orders
  │    │    SET status=2,
  │    │        third_party_order_id=?,
  │    │        paid_at=NOW()
  │    │    WHERE order_id = ?
  │    │
  │    ├─ 5.2 增加用户积分
  │    │    UPDATE user_credits
  │    │    SET balance = balance + credits_amount,
  │    │        total_earned = total_earned + credits_amount,
  │    │        total_recharged = total_recharged + credits_amount
  │    │    WHERE user_uuid = ?
  │    │
  │    ├─ 5.3 记录积分流水
  │    │    INSERT INTO credit_transactions (
  │    │      type=1,  // 充值
  │    │      amount=+credits_amount,
  │    │      related_order_id=order_id,
  │    │      ...
  │    │    )
  │    │
  │    └─ 【事务提交】
  │
  ├─ 6. 释放分布式锁
  │    Redis.Del("lock:order:" + order_id)
  │
  ├─ 7. 清除积分缓存
  │    Redis.Del("credits:" + user_uuid)
  │
  ├─ 8. WebSocket 推送通知
  │    ws.Send(user_uuid, {
  │      "type": "recharge_success",
  │      "credits": credits_amount
  │    })
  │
  └─ 9. 返回成功
       RETURN "success"
```

### 4.4 每日免费额度重置流程

```
定时任务 (Cron: 每天00:00执行)
  │
  ├─ 1. 更新所有用户的每日额度
  │    UPDATE user_credits
  │    SET daily_used_quota = 0,
  │        quota_reset_at = CURDATE()
  │    WHERE quota_reset_at < CURDATE()
  │
  └─ 2. 记录日志
       log.Info("Daily quota reset completed")
```

---

## 五、充值模块设计

### 5.1 充值套餐设计

#### 初始套餐配置 (建议)

| 套餐ID | 名称 | 积分数量 | 价格(元) | 赠送积分 | 总积分 | 性价比 |
|--------|------|----------|---------|----------|--------|--------|
| pkg_trial | 体验包 | 50 | 5 | 0 | 50 | 10积分/元 |
| pkg_basic | 基础包 | 100 | 10 | 10 | 110 | 11积分/元 |
| pkg_plus | 增强包 | 300 | 28 | 50 | 350 | 12.5积分/元 |
| pkg_pro | 专业包 | 500 | 45 | 100 | 600 | 13.3积分/元 |
| pkg_vip | VIP包 | 1000 | 80 | 300 | 1300 | 16.25积分/元 |
| pkg_first | 首充特惠 | 100 | 6.8 | 100 | 200 | 29.4积分/元 |

#### 套餐策略
- **阶梯定价**: 充值越多单价越低,引导用户大额充值
- **首充优惠**: 吸引新用户体验
- **限时活动**: 节假日推出特惠套餐
- **VIP专属**: 高等级用户独享套餐

### 5.2 支付方式集成

#### 5.2.1 支付宝 (Alipay)

```
使用场景: Web端、H5端
集成方式: alipay-sdk-go
支付方式:
  - 电脑网站支付 (alipay.trade.page.pay)
  - 手机网站支付 (alipay.trade.wap.pay)
  - 扫码支付 (alipay.trade.precreate)

回调处理:
  - 同步回调 (return_url): 支付完成跳转,仅用于展示
  - 异步回调 (notify_url): 真正的业务处理,需验签
```

#### 5.2.2 微信支付 (WeChat Pay)

```
使用场景: App、H5、小程序
集成方式: wechatpay-go
支付方式:
  - Native支付 (扫码)
  - JSAPI支付 (公众号/小程序)
  - H5支付

回调处理:
  - 统一异步通知 (notify_url)
  - 需验证签名和解密
```

### 5.3 支付安全设计

#### 5.3.1 防重复支付
```
创建订单时:
  - 订单号唯一索引
  - 30分钟过期时间
  - 前端防重复提交

处理回调时:
  - 分布式锁 (Redis SETNX)
  - 检查订单状态 (幂等性)
  - 金额校验
```

#### 5.3.2 签名验证
```
支付宝:
  - RSA2签名算法
  - 验证 sign 参数
  - 使用支付宝公钥验证

微信:
  - SHA256withRSA
  - 验证平台证书序列号
  - 解密敏感信息
```

#### 5.3.3 异常处理
```
订单超时:
  - 定时任务扫描过期订单
  - 自动取消未支付订单

支付失败:
  - 记录错误日志
  - 不扣积分
  - 允许重新支付

退款机制:
  - 仅支持未使用积分退款
  - 退款到原支付账户
  - 扣除已使用积分
```

### 5.4 积分体系设计

#### 5.4.1 积分获取方式
```
1. 充值获得 (主要来源)
2. 新用户注册赠送 (50积分)
3. 每日签到 (5-10积分)
4. 邀请好友 (被邀请人首充后奖励100积分)
5. 活动赠送
6. 系统补偿
```

#### 5.4.2 积分消费规则
```
不同任务消耗不同积分:
- SD3 Large: 15积分/张
- SD3 Medium: 10积分/张
- SDXL 1.0: 8积分/张
- 图片放大: 5积分/张

VIP用户享折扣:
- 普通用户: 原价
- VIP: 9折
- SVIP: 8折

每日免费额度:
- 普通用户: 10积分/天
- VIP: 30积分/天
- SVIP: 100积分/天
```

#### 5.4.3 积分有效期
```
充值积分: 永久有效
赠送积分: 1年有效期
活动积分: 按活动规则

过期处理:
  - 定时任务每天扫描
  - 扣除过期积分
  - 记录流水 (type=5)
```

---

## 六、接口设计

### 6.1 生图相关接口

#### 6.1.1 文生图
```
POST /api/v1/generation/text2img
Headers: Authorization: Bearer {jwt_token}
Body:
{
  "prompt": "A majestic cat wearing a wizard hat, digital art",
  "negative_prompt": "blurry, low quality, distorted",
  "model_id": "sd3-large",
  "aspect_ratio": "1:1",
  "style_preset": "photographic",
  "seed": 1234567890,
  "output_format": "png"
}

Response 200:
{
  "code": 200,
  "message": "任务创建成功",
  "data": {
    "task_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "queued",
    "estimated_time_seconds": 30,
    "cost_credits": 15,
    "remaining_credits": 485,
    "created_at": "2025-11-02T15:30:00Z"
  }
}

Response 400:
{
  "code": 400,
  "message": "参数错误: prompt不能为空"
}

Response 402:
{
  "code": 402,
  "message": "积分不足",
  "data": {
    "required": 15,
    "current": 5,
    "shortage": 10
  }
}
```

#### 6.1.2 图生图
```
POST /api/v1/generation/img2img
Content-Type: multipart/form-data

Fields:
  - init_image: File (required)
  - prompt: String (required)
  - negative_prompt: String
  - model_id: String (required)
  - strength: Float (0.0-1.0, default: 0.7)
  - seed: Integer

Response: 同 text2img
```

#### 6.1.3 查询任务状态
```
GET /api/v1/generation/tasks/{task_id}

Response 200:
{
  "code": 200,
  "data": {
    "task_id": "xxx",
    "status": "completed",
    "task_type": "text2img",
    "prompt": "...",
    "model_id": "sd3-large",
    "output_image_url": "https://cdn.example.com/xxx.png",
    "output_width": 1024,
    "output_height": 1024,
    "seed": 1234567890,
    "cost_credits": 15,
    "generation_time_ms": 3500,
    "created_at": "2025-11-02T15:30:00Z",
    "completed_at": "2025-11-02T15:30:03Z"
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
```
GET /api/v1/generation/tasks?page=1&page_size=20&status=completed

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
        "output_image_url": "...",
        "created_at": "2025-11-02T15:30:00Z"
      },
      ...
    ]
  }
}
```

#### 6.1.5 取消任务
```
DELETE /api/v1/generation/tasks/{task_id}

Response 200:
{
  "code": 200,
  "message": "任务已取消",
  "data": {
    "refund_credits": 15
  }
}

限制:
- 只能取消 queued 状态的任务
- processing/completed 状态不可取消
```

### 6.2 积分相关接口

#### 6.2.1 查询积分余额
```
GET /api/v1/credit/balance

Response 200:
{
  "code": 200,
  "data": {
    "balance": 500,
    "frozen_balance": 30,
    "available_balance": 470,
    "total_earned": 1000,
    "total_spent": 500,
    "level": 2,
    "daily_free_quota": 30,
    "daily_used_quota": 10,
    "daily_remaining_quota": 20
  }
}
```

#### 6.2.2 查询积分流水
```
GET /api/v1/credit/transactions?page=1&page_size=20&type=2

Response 200:
{
  "code": 200,
  "data": {
    "total": 85,
    "transactions": [
      {
        "transaction_id": "xxx",
        "type": "consume",
        "type_name": "消费",
        "amount": -15,
        "balance_after": 485,
        "title": "文生图消费",
        "description": "任务ID: xxx",
        "created_at": "2025-11-02T15:30:00Z"
      },
      ...
    ]
  }
}

type参数:
- 1: 充值
- 2: 消费
- 3: 退款
- 4: 赠送
```

### 6.3 充值相关接口

#### 6.3.1 获取充值套餐列表
```
GET /api/v1/credit/packages

Response 200:
{
  "code": 200,
  "data": {
    "packages": [
      {
        "package_id": "pkg_basic",
        "name": "基础包",
        "credits_amount": 100,
        "bonus_credits": 10,
        "total_credits": 110,
        "price": 10.00,
        "original_price": 12.00,
        "discount": "83折",
        "tags": ["推荐"],
        "is_recommended": true
      },
      ...
    ]
  }
}
```

#### 6.3.2 创建充值订单
```
POST /api/v1/credit/recharge
Body:
{
  "package_id": "pkg_basic",
  "payment_method": "alipay",
  "payment_channel": "web"
}

Response 200:
{
  "code": 200,
  "data": {
    "order_id": "RO202511021530001",
    "payment_url": "https://openapi.alipay.com/...",
    "qr_code": "https://api.xxx.com/qr/{order_id}",
    "amount": 10.00,
    "credits_amount": 110,
    "expired_at": "2025-11-02T16:00:00Z"
  }
}
```

#### 6.3.3 查询订单状态
```
GET /api/v1/credit/orders/{order_id}

Response 200:
{
  "code": 200,
  "data": {
    "order_id": "xxx",
    "status": "paid",
    "status_name": "已支付",
    "credits_amount": 110,
    "payment_amount": 10.00,
    "payment_method": "alipay",
    "created_at": "2025-11-02T15:30:00Z",
    "paid_at": "2025-11-02T15:31:20Z"
  }
}

status:
- pending: 待支付
- paying: 支付中
- paid: 已支付
- failed: 支付失败
- cancelled: 已取消
- refunded: 已退款
```

#### 6.3.4 支付回调 (内部接口)
```
POST /api/v1/callback/alipay
POST /api/v1/callback/wechat

处理逻辑见 4.3 充值流程
```

### 6.4 模型相关接口

#### 6.4.1 获取可用模型列表
```
GET /api/v1/generation/models?type=text2img

Response 200:
{
  "code": 200,
  "data": {
    "models": [
      {
        "model_id": "sd3-large",
        "model_name": "Stable Diffusion 3 Large",
        "type": "text2img",
        "category": "general",
        "cost_per_image": 15,
        "max_resolution": "1024x1024",
        "supported_aspect_ratios": ["1:1", "16:9", "9:16", "3:2", "2:3"],
        "description": "最新的SD3大模型,生成质量极高",
        "preview_image_url": "...",
        "tags": ["推荐", "高质量"],
        "is_recommended": true,
        "avg_generation_time_seconds": 3.5
      },
      ...
    ]
  }
}
```

### 6.5 WebSocket 接口

#### 6.5.1 建立连接
```
WebSocket: ws://api.example.com/ws?token={jwt_token}

连接成功后服务端推送:
{
  "type": "connected",
  "user_uuid": "xxx",
  "timestamp": "2025-11-02T15:30:00Z"
}
```

#### 6.5.2 实时消息格式
```
任务进度推送:
{
  "type": "task_progress",
  "task_id": "xxx",
  "status": "processing",
  "progress": 50,
  "message": "正在生成图片..."
}

任务完成推送:
{
  "type": "task_completed",
  "task_id": "xxx",
  "status": "completed",
  "output_image_url": "https://...",
  "generation_time_ms": 3500
}

充值成功推送:
{
  "type": "recharge_success",
  "order_id": "xxx",
  "credits_amount": 110,
  "new_balance": 610
}
```

---

## 七、目录结构

```
AI-Nexus/
├── cmd/
│   └── ainexus.go                    # 主程序入口
│
├── internal/
│   ├── controller/
│   │   ├── generation/
│   │   │   └── generation-controller.go    # 生图控制器
│   │   └── credit/
│   │       ├── credit-controller.go        # 积分控制器
│   │       └── payment-controller.go       # 支付控制器
│   │
│   ├── service/
│   │   ├── generation/
│   │   │   ├── generation-service.go       # 生图服务
│   │   │   ├── stability-client.go         # Stability AI SDK封装
│   │   │   └── worker-service.go           # Worker服务
│   │   ├── credit/
│   │   │   ├── credit-service.go           # 积分服务
│   │   │   └── transaction-service.go      # 流水服务
│   │   └── payment/
│   │       ├── payment-service.go          # 支付服务
│   │       ├── alipay-client.go            # 支付宝客户端
│   │       └── wechat-client.go            # 微信支付客户端
│   │
│   ├── dao/
│   │   ├── generation/
│   │   │   ├── task-dao.go                 # 任务DAO
│   │   │   └── model-dao.go                # 模型DAO
│   │   └── credit/
│   │       ├── credit-dao.go               # 积分DAO
│   │       ├── transaction-dao.go          # 流水DAO
│   │       └── order-dao.go                # 订单DAO
│   │
│   ├── domain/
│   │   ├── do/
│   │   │   ├── generation/
│   │   │   │   ├── task-do.go              # 任务DO
│   │   │   │   └── model-do.go             # 模型DO
│   │   │   └── credit/
│   │   │       ├── credit-do.go            # 积分DO
│   │   │       ├── transaction-do.go       # 流水DO
│   │   │       ├── order-do.go             # 订单DO
│   │   │       └── package-do.go           # 套餐DO
│   │   │
│   │   ├── dto/
│   │   │   ├── generation/
│   │   │   │   └── generation-dto.go       # 生图请求DTO
│   │   │   └── credit/
│   │   │       └── payment-dto.go          # 支付请求DTO
│   │   │
│   │   ├── vo/
│   │   │   ├── generation/
│   │   │   │   └── generation-vo.go        # 生图响应VO
│   │   │   └── credit/
│   │   │       ├── credit-vo.go            # 积分响应VO
│   │   │       └── payment-vo.go           # 支付响应VO
│   │   │
│   │   └── query/
│   │       └── generation/
│   │           └── generation-query.go     # 查询Query
│   │
│   ├── routes/
│   │   ├── generation/
│   │   │   └── generation-routes.go        # 生图路由
│   │   └── credit/
│   │       └── credit-routes.go            # 积分路由
│   │
│   ├── pkg/
│   │   ├── queue/
│   │   │   ├── rabbitmq.go                 # RabbitMQ客户端
│   │   │   ├── producer.go                 # 消息生产者
│   │   │   └── consumer.go                 # 消息消费者
│   │   ├── storage/
│   │   │   ├── storage.go                  # 存储接口
│   │   │   ├── local.go                    # 本地存储实现
│   │   │   └── minio.go                    # MinIO实现
│   │   └── websocket/
│   │       └── hub.go                      # WebSocket Hub
│   │
│   └── worker/
│       └── main.go                         # Worker独立进程入口
│
├── configs/
│   ├── config.yaml                         # 配置文件
│   └── db.sql                              # 数据库脚本(新增表)
│
├── docs/
│   ├── design/
│   │   └── ai-generation-design.md         # 本设计文档
│   └── api/
│       └── generation-api.md               # API文档
│
└── static/
    └── generations/                        # 生成图片存储目录
```

---

## 八、技术选型说明

### 8.1 核心技术栈

| 组件 | 技术选型 | 版本要求 | 说明 |
|------|---------|---------|------|
| 编程语言 | Go | 1.25+ | 高性能,并发能力强 |
| Web框架 | Gin | v1.11+ | 轻量级,路由性能好 |
| ORM | GORM | v1.31+ | 功能完善,生态好 |
| 消息队列 | RabbitMQ | 3.13+ | 成熟稳定,功能丰富 |
| 数据库 | MySQL | 8.0+ | 事务支持,成熟可靠 |
| 缓存 | Redis | 7.0+ | 高性能KV存储 |
| 对象存储 | MinIO | 可选 | S3兼容,私有化部署 |
| AI接口 | Stability AI | v2beta | 官方API,质量好 |

### 8.2 Go依赖包

```go
// go.mod 新增依赖
require (
    // RabbitMQ
    github.com/rabbitmq/amqp091-go v1.10.0

    // 图片处理
    github.com/disintegration/imaging v1.6.2

    // 对象存储
    github.com/minio/minio-go/v7 v7.0.70

    // 支付SDK
    github.com/smartwalle/alipay/v3 v3.2.22
    github.com/wechatpay-apiv3/wechatpay-go v0.2.18

    // WebSocket
    github.com/gorilla/websocket v1.5.1

    // 工具库
    github.com/google/uuid v1.6.0            // 已有
    github.com/spf13/viper v1.18.2           // 配置管理
)
```

### 8.3 环境要求

#### 开发环境
```
- Go 1.25+
- MySQL 8.0+
- Redis 7.0+
- RabbitMQ 3.13+
- MinIO (可选)
```

#### 生产环境 (建议)
```
- 2核4G (最低)
- 4核8G (推荐)
- SSD硬盘
- 1Mbps+ 带宽
```

---

## 九、扩展性设计

### 9.1 水平扩展

#### 9.1.1 HTTP服务扩展
```
多实例部署:
- Nginx负载均衡
- 无状态设计
- Session存储在Redis
- 支持动态扩缩容
```

#### 9.1.2 Worker扩展
```
独立部署Worker:
- 可单独增加Worker实例
- 不同任务类型独立扩容
- 支持跨机器部署

配置示例:
  机器1: 5个Text2Img Worker
  机器2: 3个Img2Img Worker
  机器3: 2个Upscale Worker
```

#### 9.1.3 RabbitMQ集群
```
镜像队列:
- 3节点集群
- 自动故障转移
- 消息高可用
```

### 9.2 功能扩展

#### 9.2.1 多AI提供商支持
```
接口抽象:
type AIProvider interface {
    Text2Img(params) (*Image, error)
    Img2Img(params) (*Image, error)
}

实现:
- StabilityAIProvider
- MidjourneyProvider (通过API)
- OpenAI DALL-E Provider
- 自部署模型 (ComfyUI)

动态切换:
- 根据模型ID选择Provider
- 失败自动切换备用Provider
- 成本优化
```

#### 9.2.2 新功能模块
```
未来可扩展:
1. ControlNet支持 (精准控制)
2. 视频生成 (Runway/Pika)
3. 3D模型生成
4. AI训练 (Lora/Dreambooth)
5. 图片编辑 (抠图/换脸/风格迁移)
```

### 9.3 监控与告警

#### 9.3.1 指标监控
```
业务指标:
- 任务成功率
- 平均生成时长
- 队列积压数量
- API调用费用

系统指标:
- CPU/内存使用率
- RabbitMQ队列长度
- Redis缓存命中率
- 数据库慢查询
```

#### 9.3.2 日志系统
```
日志级别:
- Error: 错误日志(需告警)
- Warn: 警告日志
- Info: 业务日志
- Debug: 调试日志

日志内容:
- RequestID追踪
- 任务生命周期
- API调用耗时
- 错误堆栈
```

#### 9.3.3 告警规则
```
触发条件:
- 任务失败率 > 10%
- 队列积压 > 1000
- API调用失败率 > 5%
- 支付回调失败

通知方式:
- 企业微信
- 邮件
- 短信 (紧急)
```

---

## 附录

### A. 数据库初始化脚本

见 `configs/db-migration.sql`

### B. RabbitMQ初始化脚本

见 `scripts/rabbitmq-init.sh`

### C. 配置文件示例

见 `configs/config.example.yaml`

### D. API文档

见 `docs/api/generation-api.md`

---

## 总结

本设计方案完整覆盖了AI生图模块的核心功能:

✅ **生图功能**: 文生图/图生图/放大/重绘
✅ **积分体系**: 充值/消费/流水/等级
✅ **支付系统**: 支付宝/微信/订单管理
✅ **异步处理**: RabbitMQ队列/Worker池
✅ **高可用**: 消息持久化/失败重试/监控告警
✅ **可扩展**: 模块化设计/水平扩展/多Provider

通过RabbitMQ学习:
- Topic Exchange路由机制
- 优先级队列
- 死信队列处理
- 消息确认机制
- 持久化与可靠性

后续可根据此设计逐步实现代码！
