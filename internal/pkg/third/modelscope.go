package third

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
)

const (
	// TaskStatusSucceed 任务成功
	TaskStatusSucceed = "SUCCEED"
	// TaskStatusFailed 任务失败
	TaskStatusFailed = "FAILED"
	// TaskStatusPending 任务等待中
	TaskStatusPending = "PENDING"
	// TaskStatusProcessing 任务处理中
	TaskStatusProcessing = "PROCESSING"
)

// ModelScopeCreateRequest ModelScope 创建任务请求
type ModelScopeCreateRequest struct {
	Model             string  `json:"model"`
	Prompt            string  `json:"prompt"`
	NegativePrompt    string  `json:"negative_prompt,omitempty"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	NumInferenceSteps int     `json:"num_inference_steps"`
	GuidanceScale     float64 `json:"guidance_scale"`
	Seed              int64   `json:"seed"`
}

// ModelScopeCreateResponse ModelScope 创建任务响应
type ModelScopeCreateResponse struct {
	TaskID     string `json:"task_id"`
	TaskStatus string `json:"task_status"`
	RequestID  string `json:"request_id"`
}

// ModelScopeTaskResponse ModelScope 任务状态响应
type ModelScopeTaskResponse struct {
	TaskID       string   `json:"task_id"`
	TaskStatus   string   `json:"task_status"`
	OutputImages []string `json:"output_images,omitempty"`
	Message      string   `json:"message,omitempty"`
	TimeTaken    float64  `json:"time_taken"`
	RequestID    string   `json:"request_id"`
}

// ModelScopeClient ModelScope API 客户端
type ModelScopeClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewModelScopeClient 创建 ModelScope 客户端
func NewModelScopeClient(baseURL, apiKey string) *ModelScopeClient {
	return &ModelScopeClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// CreateTask 创建图像生成任务
func (c *ModelScopeClient) CreateTask(thirdPartyModelID string, payload rabbitmq.Text2ImgPayload) (string, error) {
	reqPayload := ModelScopeCreateRequest{
		Model:             thirdPartyModelID,
		Prompt:            payload.Prompt,
		NegativePrompt:    payload.NegativePrompt,
		Width:             payload.Width,
		Height:            payload.Height,
		NumInferenceSteps: payload.NumInferenceSteps,
		GuidanceScale:     payload.GuidanceScale,
		Seed:              payload.Seed,
	}

	reqBody, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("marshal request payload: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"v1/images/generations", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ModelScope-Async-Mode", "true")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("task submit failed with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp ModelScopeCreateResponse
	if err := json.Unmarshal(body, &createResp); err != nil {
		return "", fmt.Errorf("unmarshal create response: %w", err)
	}

	return createResp.TaskID, nil
}

// WaitForTaskCompletion 等待任务完成
func (c *ModelScopeClient) WaitForTaskCompletion(taskID string, maxAttempts int, pollInterval time.Duration) (*ModelScopeTaskResponse, error) {
	for attempts := 1; attempts <= maxAttempts; attempts++ {
		taskResp, err := c.GetTaskStatus(taskID)
		if err != nil {
			return nil, err
		}

		switch taskResp.TaskStatus {
		case TaskStatusSucceed:
			if len(taskResp.OutputImages) == 0 {
				return nil, fmt.Errorf("task succeed but no output images")
			}
			return taskResp, nil

		case TaskStatusFailed:
			return nil, fmt.Errorf("task failed: %s", taskResp.Message)

		case TaskStatusPending, TaskStatusProcessing:
			if attempts < maxAttempts {
				time.Sleep(pollInterval)
			}

		default:
			return nil, fmt.Errorf("unknown task status: %s", taskResp.TaskStatus)
		}
	}

	return nil, fmt.Errorf("timeout: task not completed within %d attempts", maxAttempts)
}

// GetTaskStatus 获取任务状态
func (c *ModelScopeClient) GetTaskStatus(taskID string) (*ModelScopeTaskResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"v1/tasks/"+taskID, nil)
	if err != nil {
		return nil, fmt.Errorf("create status request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ModelScope-Task-Type", "image_generation")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send status request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read status response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get task status failed with status %d: %s", resp.StatusCode, string(body))
	}

	var taskResp ModelScopeTaskResponse
	if err := json.Unmarshal(body, &taskResp); err != nil {
		return nil, fmt.Errorf("unmarshal status response: %w", err)
	}

	return &taskResp, nil
}

// CallThirdAPI 调用第三方 API 进行图片生成
func CallThirdAPI(baseURL, apiKey, thirdPartyModelID string, payload rabbitmq.Text2ImgPayload) (string, float64, error) {
	client := NewModelScopeClient(baseURL, apiKey)

	// 创建任务
	taskID, err := client.CreateTask(thirdPartyModelID, payload)
	if err != nil {
		return "", 0, fmt.Errorf("create task: %w", err)
	}

	// 轮询任务状态
	taskResp, err := client.WaitForTaskCompletion(taskID, 60, 5*time.Second)
	if err != nil {
		return "", 0, fmt.Errorf("wait for task completion: %w", err)
	}

	return taskResp.OutputImages[0], taskResp.TimeTaken, nil
}
