CREATE DATABASE ai_nexus;

USE ai_nexus;

CREATE TABLE test (
  id INT NOT NULL AUTO_INCREMENT,
  message VARCHAR(255),
  PRIMARY KEY (id)
)

INSERT INTO test (message)
VALUES ('helloworld'), ('helloworld2');

CREATE TABLE IF NOT EXISTS `users` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `uuid` CHAR(36) NOT NULL '每个用户自带的唯一id',
  `nickname` VARCHAR(64) NOT NULL COMMENT '用户昵称',
  `email` VARCHAR(255) NOT NULL COMMENT '用户邮箱',
  `password_hash` VARCHAR(255) NOT NULL 'Argon2id加密密码',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '1=active, 0=pending, -1=blocked',
  `last_login` DATETIME NOT NULL DEFAULT NULL COMMENT '上次登录的时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '账号创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '信息更新时间, 如修改密码等'

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uuid` (`uuid`),
  UNIQUE KEY `uk_account` (`nickname`),
  UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `user_verification_codes` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT UNSIGNED NULL,
  `email` VARCHAR(255) NOT NULL,
  `code` VARCHAR(16) NOT NULL,
  `purpose` TINYINT NOT NULL COMMENT '1=register, 2=reset_password',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  PRIMARY KEY (`id`),
  INDEX `idx_email_purpose` (`email`, `purpose`),
  CONSTRAINT `fk_user_verification_codes_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;