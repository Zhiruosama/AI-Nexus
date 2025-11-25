package queue

// TaskMessage 任务消息结构
type TaskMessage struct {
	TaskID   string `json:"task_id"`
	UserUUID string `json:"user_uuid"`
	Payload  any    `json:"payload"`
}

// Text2ImgPayload 文生图任务负载
type Text2ImgPayload struct {
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt"`
	ModelID           string  `json:"model_id"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	NumInferenceSteps int     `json:"num_inference_steps"`
	GuidanceScale     float64 `json:"guidance_scale"`
	Seed              int64   `json:"seed"`
}

// Img2ImgPayload 图生图任务负载
type Img2ImgPayload struct {
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt"`
	ModelID           string  `json:"model_id"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	NumInferenceSteps int     `json:"num_inference_steps"`
	GuidanceScale     float64 `json:"guidance_scale"`
	Seed              int64   `json:"seed"`
	InputImageURL     string  `json:"input_image_url"`
	Strength          float64 `json:"strength"`
}
