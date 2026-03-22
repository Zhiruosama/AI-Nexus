// Package chat 提供多模型对话的 Provider 抽象
package chat

import "context"

// Provider 定义对话模型提供商的统一接口
type Provider interface {
	// ChatStream 发起流式对话，通过 channel 逐块返回
	ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error)
}

// Request 对话请求
type Request struct {
	Model       string
	Messages    []Message
	Temperature float64
	TopP        float64
	MaxTokens   int
}

// StreamChunk 流式响应块
type StreamChunk struct {
	Delta        string
	FinishReason string
	Usage        *Usage
	Err          error
}

// Message 对话消息
type Message struct {
	Role    string
	Content string
}

// Usage token 用量
type Usage struct {
	PromptTokens     int
	CompletionTokens int
}

// NewProvider 根据 provider 类型创建对应实例
func NewProvider(provider, apiKey, baseURL string) Provider {
	switch provider {
	case "anthropic":
		return NewAnthropicProvider(apiKey, baseURL)
	case "gemini":
		return NewGeminiProvider(apiKey, baseURL)
	default:
		return NewOpenAIProvider(apiKey, baseURL)
	}
}
