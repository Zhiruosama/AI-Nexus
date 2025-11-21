CREATE DATABASE IF NOT EXISTS ai_nexus;

USE ai_nexus;

CREATE TABLE IF NOT EXISTS `test` (
  `id` INT NOT NULL AUTO_INCREMENT,
  message VARCHAR(255),
  PRIMARY KEY (`id`)
);

INSERT INTO `test` (`message`) VALUES ('helloworld'), ('helloworld2');

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `uuid` CHAR(36) NOT NULL UNIQUE COMMENT '每个用户自带的唯一id',
  `nickname` VARCHAR(64) NOT NULL UNIQUE COMMENT '用户昵称',
  `avatar` VARCHAR(64) NOT NULL COMMENT '用户头像',
  `email` VARCHAR(255) NOT NULL UNIQUE COMMENT '用户邮箱',
  `password_hash` VARCHAR(255) NOT NULL COMMENT 'Argon2id加密密码',
  `last_login` DATETIME DEFAULT NULL COMMENT '上次登录的时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '账号创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '信息更新时间, 如修改密码等',

  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_verification_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `email` VARCHAR(255) NOT NULL COMMENT '用户邮箱',
  `code` VARCHAR(16) NOT NULL COMMENT '验证码',
  `purpose` TINYINT NOT NULL COMMENT '1=register, 2=reset_password, 3=login' COMMENT '发送验证码目的',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '验证码创建时间',

  PRIMARY KEY (`id`),
  INDEX `idx_email_purpose_created_at` (`email`, `purpose`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `image_generation_tasks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `task_id` CHAR(36) NOT NULL COMMENT '任务UUID,对外暴露',
  `user_uuid` CHAR(36) NOT NULL COMMENT '用户UUID, 关联 users.uuid',

  -- 任务基本信息
  `task_type` TINYINT UNSIGNED NOT NULL COMMENT '任务类型: 1-文生图, 2-图生图',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '任务状态: 0-待处理, 1-队列中, 2-处理中, 3-已完成, 4-失败, 5-已取消',

  -- 输入参数
  `prompt` TEXT NOT NULL COMMENT '正向提示词',
  `negative_prompt` TEXT COMMENT '负向提示词',
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
  KEY `idx_user_status` (`user_uuid`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI生图任务表';

CREATE TABLE IF NOT EXISTS `image_generation_models` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `model_id` VARCHAR(64) NOT NULL COMMENT '模型标识 (如: sd-v1.5, sd-v2.1)',

  -- 基本信息
  `model_name` VARCHAR(128) NOT NULL COMMENT '模型显示名称',
  `model_type` VARCHAR(32) NOT NULL COMMENT '类型: text2img/img2img',
  `provider` VARCHAR(32) DEFAULT 'modelscope' COMMENT '提供商: modelscope',

  -- 显示与排序
  `description` TEXT COMMENT '模型描述',
  `tags` VARCHAR(128) COMMENT '标签: ["快速", "高质量"]',
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

CREATE TABLE dead_letter_tasks (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id VARCHAR(64) NOT NULL COMMENT '用户UUID',
  task_id VARCHAR(64) NOT NULL UNIQUE COMMENT '任务ID',
  task_type TINYINT NOT NULL COMMENT '任务类型 1:text2img 2:img2img',
  dead_reason TEXT COMMENT '死信原因',
  original_status TINYINT COMMENT '进入死信时的原始状态',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '死信记录时间',

  INDEX idx_user_id (user_id),
  INDEX idx_created_at (created_at)
) COMMENT='死信任务记录表';