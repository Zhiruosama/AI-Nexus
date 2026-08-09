#!/usr/bin/env bash
set -euo pipefail

env_file="${1:-.env}"
example_file="${2:-.env.example}"

if [[ ! -f "$env_file" ]]; then
  cp "$example_file" "$env_file"
fi

ensure_value() {
  local key="$1"
  local value="$2"
  if ! grep -q "^${key}=" "$env_file"; then
    printf '%s=%s\n' "$key" "$value" >> "$env_file"
  fi
}

ensure_secret() {
  local key="$1"
  local placeholder="$2"
  ensure_value "$key" "$placeholder"
  if grep -q "^${key}=${placeholder}$" "$env_file"; then
    sed -i "s|^${key}=${placeholder}$|${key}=$(openssl rand -hex 32)|" "$env_file"
  fi
}

ensure_value MAIL_GRPC_ADDRESS 127.0.0.1:8080
ensure_value MAIL_GRPC_TIMEOUT 3s
ensure_value MAIL_GRPC_ALLOW_INSECURE true
ensure_value MAIL_GRPC_TLS_CA_FILE ""
ensure_value MAIL_GRPC_TLS_SERVER_NAME ""
ensure_value MAIL_CALLBACK_ADDRESS :8081
ensure_value MAIL_CALLBACK_ALLOW_INSECURE true
ensure_value MAIL_CALLBACK_TLS_CERT_FILE ""
ensure_value MAIL_CALLBACK_TLS_KEY_FILE ""
ensure_value MAIL_SENDER_IDENTITY_KEY ainexus.default
ensure_value MAIL_LOCALE zh-CN
ensure_value MAIL_VERIFICATION_TTL 5m
ensure_value MAIL_DISPATCH_DEADLINE 2m
ensure_value MAIL_PENDING_TTL 15m
ensure_value MAIL_VERIFICATION_COOLDOWN 60s
ensure_value MAIL_VERIFICATION_MAX_ATTEMPTS 5
ensure_value MAIL_RECONCILE_INTERVAL 30s

ensure_secret JWT_SECRET replace-with-a-random-secret
ensure_secret CHAT_ENCRYPTION_KEY replace-with-a-64-character-hex-key
ensure_secret VERIFICATION_HMAC_SECRET replace-with-a-random-secret
ensure_secret VERIFICATION_EMAIL_FINGERPRINT_SECRET replace-with-a-random-secret
