// Package imagegeneration 此模块下的dto请求
package imagegeneration

// ModelCreateRequest 添加模型请求
type ModelCreateRequest struct {
	ModelID           string `json:"model_id" binding:"required"`
	ModelName         string `json:"model_name" binding:"required"`
	ModelType         string `json:"model_type" binding:"required"`
	Provider          string `json:"provider" binding:"required"`
	Description       string `json:"description" binding:"required"`
	Tags              string `json:"tags,omitempty"`
	SortOrder         int    `json:"sort_order"`
	ThirdPartyModelID string `json:"third_party_model_id"`
	BaseURL           string `json:"base_url"`
	DefaultWidth      int    `json:"default_width"`
	DefaultHeight     int    `json:"default_height"`
	MaxWidth          int    `json:"max_width"`
	MaxHeight         int    `json:"max_height"`
	MinSteps          int    `json:"min_steps"`
	MaxSteps          int    `json:"max_steps"`
}
