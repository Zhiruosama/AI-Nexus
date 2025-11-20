// Package imagegeneration 此模块下的dto请求
package imagegeneration

import "mime/multipart"

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

// Text2ImgDTO 文生图负载
type Text2ImgDTO struct {
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt,omitempty"`
	ModelID           string  `json:"model_id"`
	Width             int     `json:"width,omitempty"`
	Height            int     `json:"height,omitempty"`
	NumInferenceSteps int     `json:"num_inference_steps,omitempty"`
	GuidanceScale     float64 `json:"guidance_scale,omitempty"`
	Seed              int64   `json:"seed,omitempty"`
}

// Img2ImgDTO 图生图负载
type Img2ImgDTO struct {
	Prompt            string                `json:"prompt" form:"prompt"`
	InputImage        *multipart.FileHeader `json:"input_image" form:"input_image"`
	Strength          float64               `json:"strength,omitempty" form:"strength,omitempty"`
	Sha256            string                `json:"sha256,omitempty" form:"sha256,omitempty"`
	NegativePrompt    string                `json:"negative_prompt,omitempty" form:"negative_prompt"`
	ModelID           string                `json:"model_id" form:"model_id"`
	Width             int                   `json:"width,omitempty" form:"width,omitempty"`
	Height            int                   `json:"height,omitempty" form:"height,omitempty"`
	NumInferenceSteps int                   `json:"num_inference_steps,omitempty" form:"num_inference_steps,omitempty"`
	GuidanceScale     float64               `json:"guidance_scale,omitempty" form:"guidance_scale,omitempty"`
	Seed              int64                 `json:"seed,omitempty" form:"seed,omitempty"`
}
