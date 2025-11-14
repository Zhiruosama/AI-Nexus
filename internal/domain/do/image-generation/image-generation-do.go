// Package imagegeneration 对应image_generation_models表
package imagegeneration

// TableImageGenerationTaskDO 对应 image_generation_tasks 表中的 DO 结构
type TableImageGenerationTaskDO struct {
	ID       int64  `gorm:"column:id"`
	TaskID   string `gorm:"column:task_id"`
	UserUUID string `gorm:"column:user_uuid"`

	// 任务基本信息
	TaskType int8 `gorm:"column:task_type"` // 1-文生图, 2-图生图
	Status   int8 `gorm:"column:status"`    // 0-待处理, 1-队列中, 2-处理中, 3-已完成, 4-失败, 5-已取消

	// 输入参数
	Prompt            string  `gorm:"column:prompt"`
	NegativePrompt    string  `gorm:"column:negative_prompt"`
	ModelID           string  `gorm:"column:model_id"`
	Width             int     `gorm:"column:width"`
	Height            int     `gorm:"column:height"`
	NumInferenceSteps int     `gorm:"column:num_inference_steps"`
	GuidanceScale     float64 `gorm:"column:guidance_scale"`
	Seed              int64   `gorm:"column:seed"`

	// 图生图专用参数
	InputImageURL string  `gorm:"column:input_image_url"`
	Strength      float64 `gorm:"column:strength"`

	// 生成结果
	OutputImageURL string `gorm:"column:output_image_url"`
	ActualSeed     int64  `gorm:"column:actual_seed"`

	// 错误处理
	ErrorMessage string `gorm:"column:error_message"`
	RetryCount   int8   `gorm:"column:retry_count"`
	MaxRetry     int8   `gorm:"column:max_retry"`

	// 性能指标
	GenerationTimeMs int `gorm:"column:generation_time_ms"`
	QueueTimeMs      int `gorm:"column:queue_time_ms"`

	// 时间戳
	CreatedAt   string `gorm:"column:created_at"`
	QueuedAt    string `gorm:"column:queued_at"`
	StartedAt   string `gorm:"column:started_at"`
	CompletedAt string `gorm:"column:completed_at"`
	UpdatedAt   string `gorm:"column:updated_at"`
}

// TableImageGenerationModelsDO 对应 image_generation_models 表中的 DO 结构
type TableImageGenerationModelsDO struct {
	ID      int64  `gorm:"column:id"`
	ModelID string `gorm:"column:model_id"`

	// 基本信息
	ModelName string `gorm:"column:model_name"`
	ModelType string `gorm:"column:model_type"`
	Provider  string `gorm:"column:provider"`

	// 显示与排序
	Description string `gorm:"column:description"`
	Tags        string `gorm:"column:tags"`
	SortOrder   int    `gorm:"column:sort_order"`

	// 统计信息
	TotalUsage  int64   `gorm:"column:total_usage"`
	SuccessRate float64 `gorm:"column:success_rate"`

	// 状态
	IsActive      bool `gorm:"column:is_active"`
	IsRecommended bool `gorm:"column:is_recommended"`

	// 第三方平台相关
	ThirdPartyModelID string `gorm:"column:third_party_model_id"`
	BaseURL           string `gorm:"column:base_url"`

	// 能力参数
	DefaultWidth  int `gorm:"column:default_width"`
	DefaultHeight int `gorm:"column:default_height"`
	MaxWidth      int `gorm:"column:max_width"`
	MaxHeight     int `gorm:"column:max_height"`
	MinSteps      int `gorm:"column:min_steps"`
	MaxSteps      int `gorm:"column:max_steps"`

	// 时间戳
	CreatedAt string `gorm:"column:created_at"`
	UpdatedAt string `gorm:"column:updated_at"`
}
