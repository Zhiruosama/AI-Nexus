// Package chat 对话模块 DO 结构
package chat

import "time"

// TableUserAPIKeyDO 对应 user_api_keys 表
type TableUserAPIKeyDO struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	UserUUID  string    `gorm:"column:user_uuid"`
	Provider  string    `gorm:"column:provider"`
	Name      string    `gorm:"column:name"`
	IsActive  bool      `gorm:"column:is_active"`
	BaseURL   string    `gorm:"column:base_url"`
	APIKeyEnc []byte    `gorm:"column:api_key_enc"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名
func (TableUserAPIKeyDO) TableName() string { return "user_api_keys" }

// TableConversationDO 对应 conversations 表
type TableConversationDO struct {
	ID           uint64    `gorm:"column:id;primaryKey"`
	ConvID       string    `gorm:"column:conv_id"`
	UserUUID     string    `gorm:"column:user_uuid"`
	APIKeyID     uint64    `gorm:"column:api_key_id"`
	Title        string    `gorm:"column:title"`
	Model        string    `gorm:"column:model"`
	SystemPrompt string    `gorm:"column:system_prompt"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名
func (TableConversationDO) TableName() string { return "conversations" }

// TableConversationMessageDO 对应 conversation_messages 表
type TableConversationMessageDO struct {
	ID               uint64    `gorm:"column:id;primaryKey"`
	ConvID           string    `gorm:"column:conv_id"`
	Role             string    `gorm:"column:role"`
	Content          string    `gorm:"column:content"`
	PromptTokens     int       `gorm:"column:prompt_tokens"`
	CompletionTokens int       `gorm:"column:completion_tokens"`
	CreatedAt        time.Time `gorm:"column:created_at"`
}

// TableName 指定表名
func (TableConversationMessageDO) TableName() string { return "conversation_messages" }

// TableChatPresetDO 对应 chat_presets 表
type TableChatPresetDO struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	UserUUID  string    `gorm:"column:user_uuid"`
	Name      string    `gorm:"column:name"`
	Content   string    `gorm:"column:content"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName 指定表名
func (TableChatPresetDO) TableName() string { return "chat_presets" }
