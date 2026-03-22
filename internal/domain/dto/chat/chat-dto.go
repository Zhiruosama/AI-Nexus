// Package chat 对话模块 DTO 结构
package chat

// CreateAPIKeyDTO 创建 API Key
type CreateAPIKeyDTO struct {
	Provider string `json:"provider" binding:"required"`
	Name     string `json:"name" binding:"required"`
	APIKey   string `json:"api_key" binding:"required"`
	BaseURL  string `json:"base_url"`
}

// UpdateAPIKeyDTO 更新 API Key
type UpdateAPIKeyDTO struct {
	Provider *string `json:"provider"`
	Name     *string `json:"name"`
	APIKey   *string `json:"api_key"`
	BaseURL  *string `json:"base_url"`
}

// CreateConversationDTO 创建对话
type CreateConversationDTO struct {
	APIKeyID     uint64 `json:"api_key_id" binding:"required"`
	Title        string `json:"title"`
	Model        string `json:"model" binding:"required"`
	SystemPrompt string `json:"system_prompt"`
}

// SendMessageDTO 发送消息
type SendMessageDTO struct {
	Content     string  `json:"content" binding:"required"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	MaxTokens   int     `json:"max_tokens"`
}

// CreatePresetDTO 创建预设
type CreatePresetDTO struct {
	Name    string `json:"name" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// UpdatePresetDTO 更新预设
type UpdatePresetDTO struct {
	Name    *string `json:"name"`
	Content *string `json:"content"`
}
