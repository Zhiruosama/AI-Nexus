// Package imagegeneration 此模块下的dto请求
package imagegeneration

// ModelCreateDTO 添加模型请求
type ModelCreateDTO struct {
	ModelID           string `json:"model_id"`
	ModelName         string `json:"model_name"`
	ModelType         string `json:"model_type"`
	Provider          string `json:"provider"`
	Description       string `json:"description"`
	Tags              string `json:"tags,omitempty"`
	SortOrder         int    `json:"sort_order"`
	IsActive          bool   `json:"is_active"`
	IsRecommended     bool   `json:"is_recommended"`
	ThirdPartyModelID string `json:"third_party_model_id"`
	BaseURL           string `json:"base_url"`
	DefaultWidth      int    `json:"default_width"`
	DefaultHeight     int    `json:"default_height"`
	MaxWidth          int    `json:"max_width"`
	MaxHeight         int    `json:"max_height"`
	MinSteps          int    `json:"min_steps"`
	MaxSteps          int    `json:"max_steps"`
}

// BatchCreateModelsDTO 批量添加模型
type BatchCreateModelsDTO struct {
	Models []ModelCreateDTO `json:"models"`
}

// ModelUpdateDTO 更新模型
type ModelUpdateDTO struct {
	ModelID           string  `json:"model_id"`
	ModelName         *string `json:"model_name"`
	ModelType         *string `json:"model_type"`
	Provider          *string `json:"provider"`
	Description       *string `json:"description"`
	Tags              *string `json:"tags,omitempty"`
	SortOrder         *int    `json:"sort_order"`
	IsActive          *bool   `json:"is_active"`
	IsRecommended     *bool   `json:"is_recommended"`
	ThirdPartyModelID *string `json:"third_party_model_id"`
	BaseURL           *string `json:"base_url"`
	DefaultWidth      *int    `json:"default_width"`
	DefaultHeight     *int    `json:"default_height"`
	MaxWidth          *int    `json:"max_width"`
	MaxHeight         *int    `json:"max_height"`
	MinSteps          *int    `json:"min_steps"`
	MaxSteps          *int    `json:"max_steps"`
}
