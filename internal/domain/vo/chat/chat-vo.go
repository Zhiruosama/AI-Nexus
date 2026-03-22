// Package chat 对话模块 VO 结构
package chat

import "time"

// APIKeyVO API Key 响应
type APIKeyVO struct {
	ID         uint64    `json:"id"`
	Provider   string    `json:"provider"`
	Name       string    `json:"name"`
	IsActive   bool      `json:"is_active"`
	BaseURL    string    `json:"base_url"`
	APIKeyMask string    `json:"api_key_mask"`
	CreatedAt  time.Time `json:"created_at"`
}

// ConversationVO 对话响应
type ConversationVO struct {
	ConvID       string    `json:"conv_id"`
	APIKeyID     uint64    `json:"api_key_id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ConversationDetailVO 对话详情（含消息）
type ConversationDetailVO struct {
	ConversationVO
	Messages []MessageVO `json:"messages"`
}

// MessageVO 消息响应
type MessageVO struct {
	ID               uint64    `json:"id"`
	Role             string    `json:"role"`
	Content          string    `json:"content"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CreatedAt        time.Time `json:"created_at"`
}

// PresetVO 预设响应
type PresetVO struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
