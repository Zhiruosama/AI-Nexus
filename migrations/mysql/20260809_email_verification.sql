CREATE TABLE IF NOT EXISTS `email_verification_challenges` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `request_id` CHAR(36) NOT NULL,
  `message_id` VARCHAR(128) NULL,
  `email_fingerprint` BINARY(32) NOT NULL,
  `purpose` VARCHAR(32) NOT NULL,
  `code_digest` BINARY(32) NOT NULL,
  `state` VARCHAR(32) NOT NULL,
  `valid_for_seconds` INT UNSIGNED NOT NULL,
  `max_attempts` INT UNSIGNED NOT NULL,
  `failed_attempts` INT UNSIGNED NOT NULL DEFAULT 0,
  `latest_delivery_status` INT NOT NULL DEFAULT 0,
  `latest_sequence` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `active_at` DATETIME(6) NULL,
  `expires_at` DATETIME(6) NULL,
  `consumed_at` DATETIME(6) NULL,
  `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  `updated_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_verification_request_id` (`request_id`),
  UNIQUE KEY `uq_verification_message_id` (`message_id`),
  KEY `idx_verification_lookup` (`email_fingerprint`, `purpose`, `state`, `active_at`),
  KEY `idx_verification_reconcile` (`state`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `email_delivery_events` (
  `event_id` VARCHAR(128) NOT NULL,
  `message_id` VARCHAR(128) NOT NULL,
  `request_id` CHAR(36) NOT NULL,
  `sequence` BIGINT UNSIGNED NOT NULL,
  `delivery_status` INT NOT NULL,
  `occurred_at` DATETIME(6) NOT NULL,
  `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (`event_id`),
  KEY `idx_delivery_event_message_sequence` (`message_id`, `sequence`),
  KEY `idx_delivery_event_request` (`request_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- The legacy table stored directly verifiable codes in plaintext and is no
-- longer read by the application. Verification codes are short-lived data, so
-- retaining those rows provides no migration value and creates avoidable risk.
DROP TABLE IF EXISTS `user_verification_codes`;
