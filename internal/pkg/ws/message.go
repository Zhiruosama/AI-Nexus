package ws

import "time"

// MessageType 定义消息类型
type MessageType string

const (
	// MessageTypeConnected 连接相关
	MessageTypeConnected MessageType = "connected"
	// MessageTypePing ping
	MessageTypePing MessageType = "ping"
	// MessageTypePong pong
	MessageTypePong MessageType = "pong"

	// MessageTypeTaskQueued 任务已入队
	MessageTypeTaskQueued MessageType = "task_queued"
	// MessageTypeTaskProgress 任务处理中
	MessageTypeTaskProgress MessageType = "task_progress"
	// MessageTypeTaskCompleted 任务完成
	MessageTypeTaskCompleted MessageType = "task_completed"
	// MessageTypeTaskFailed 任务失败
	MessageTypeTaskFailed MessageType = "task_failed"
	// MessageTypeTaskCancelled 任务已取消
	MessageTypeTaskCancelled MessageType = "task_cancelled"
)

// Message WebSocket 消息结构
type Message struct {
	Type      MessageType `json:"type"`
	Data      any         `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserUUID  string      `json:"-"`
}

// TaskProgressData 任务进度数据
type TaskProgressData struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// TaskCompletedData 任务完成数据
type TaskCompletedData struct {
	TaskID           string `json:"task_id"`
	Status           string `json:"status"`
	OutputImageURL   string `json:"output_image_url"`
	GenerationTimeMs int64  `json:"generation_time_ms"`
}

// TaskFailedData 任务失败数据
type TaskFailedData struct {
	TaskID       string `json:"task_id"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

// ConnectedData 连接成功数据
type ConnectedData struct {
	SuccessMsg string `json:"success_msg"`
}
