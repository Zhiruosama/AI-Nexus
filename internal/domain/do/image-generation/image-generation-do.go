// Package imagegeneration 对应image_generation_models表
package imagegeneration

// TableImageGenerationTaskDO 对应 image_generation_tasks 表中的 DO 结构
type TableImageGenerationTaskDO struct {
	ID               int64   `gorm:"column:id"`
	TaskID           string  `gorm:"column:task_id"`
	UserID           string  `gorm:"column:user_uuid"`
	TaskType         int8    `gorm:"column:task_type"`
	TaskStatus       int8    `gorm:"column:task_status"`
	Prompt           string  `gorm:"column:prompt"`
	NegativePrompt   string  `gorm:"column:negative_prompt"`
	ModelID          string  `gorm:"column:model_id"`
	Width            int     `gorm:"column:width"`
	Height           int     `gorm:"column:height"`
	Steps            int     `gorm:"column:num_inference_steps"`
	GuidanceScale    float64 `gorm:"column:guidance_scale"`
	Seed             int64   `gorm:"column:seed"`
	InputImageUrl    string  `gorm:"column:input_image_url"`
	Strength         float64 `gorm:"column:strength"`
	OutputImageUrl   string  `gorm:"column:output_image_url"`
	ActualSeed       int64   `gorm:"column:actual_seed"`
	ErrorMessage     string  `gorm:"column:error_message"`
	RetryCount       int8    `gorm:"column:retry_count"`
	MaxRetry         int8    `gorm:"column:max_retry"`
	GenerationTimeMs int     `gorm:"column:generation_time_ms"`
	QueueTimeMs      int     `gorm:"column:queue_time_ms"`
	MqMessageID      string  `gorm:"column:mq_message_id"`
	CreatedAt        string  `gorm:"column:created_at"`
	QueuedAt         string  `gorm:"column:queued_at"`
	StartedAt        string  `gorm:"column:started_at"`
	CompletedAt      string  `gorm:"column:completed_at"`
	UpdatedAt        string  `gorm:"column:updated_at"`
}

// TableImageGenerationModelsDO 对应 image_generation_models 表中的 DO 结构
type TableImageGenerationModelsDO struct {
	ID                int64   `gorm:"column:id"`
	ModelID           string  `gorm:"column:model_id"`
	ModelName         string  `gorm:"column:model_name"`
	ModelType         string  `gorm:"column:model_type"`
	Provider          string  `gorm:"column:provider"`
	Description       string  `gorm:"column:description"`
	Tags              string  `gorm:"column:tags"`
	SortOrder         int     `gorm:"column:sort_order"`
	TotalUsage        int64   `gorm:"column:total_usage"`
	SuccessRate       float64 `gorm:"column:success_rate"`
	IsActive          bool    `gorm:"column:is_active"`
	IsRecommended     bool    `gorm:"column:is_recommended"`
	ThirdPartyModelID string  `gorm:"column:third_party_model_id"`
	BaseUrl           string  `gorm:"column:base_url"`
	DefaultWidth      int     `gorm:"column:default_width"`
	DefaultHeight     int     `gorm:"column:default_height"`
	MaxWidth          int     `gorm:"column:max_width"`
	MaxHeight         int     `gorm:"column:max_height"`
	MinSteps          int     `gorm:"column:min_steps"`
	MaxSteps          int     `gorm:"column:max_steps"`
	CreateAt          string  `gorm:"column:created_at"`
	UpdatedAt         string  `gorm:"column:updated_at"`
}
