// Package chat OpenAI 兼容 Provider 实现
package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1/"

// OpenAIProvider OpenAI 兼容 Provider
type OpenAIProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewOpenAIProvider 创建 OpenAI Provider
func NewOpenAIProvider(apiKey, baseURL string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type openAIRequest struct {
	Model         string               `json:"model"`
	Messages      []openAIMessage      `json:"messages"`
	Stream        bool                 `json:"stream"`
	Temperature   float64              `json:"temperature,omitempty"`
	TopP          float64              `json:"top_p,omitempty"`
	MaxTokens     int                  `json:"max_tokens,omitempty"`
	StreamOptions *openAIStreamOptions `json:"stream_options,omitempty"`
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIStreamResponse struct {
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage         `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Delta        openAIDelta `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

type openAIDelta struct {
	Content string `json:"content"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// ChatStream 发起流式对话
func (p *OpenAIProvider) ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error) {
	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openAIMessage(m)
	}

	body := openAIRequest{
		Model:         req.Model,
		Messages:      messages,
		Stream:        true,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		MaxTokens:     req.MaxTokens,
		StreamOptions: &openAIStreamOptions{IncludeUsage: true},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			log.Printf("[OpenAI] close error response body: %v\n", err)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan StreamChunk, 32)
	go func() {
		defer close(ch)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("[OpenAI] close stream body: %v\n", err)
			}
		}()
		p.readSSE(ctx, resp.Body, ch)
	}()
	return ch, nil
}

func (p *OpenAIProvider) readSSE(ctx context.Context, body io.Reader, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- StreamChunk{Err: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}

		var resp openAIStreamResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("unmarshal SSE: %w", err)}
			return
		}

		chunk := StreamChunk{}

		if resp.Usage != nil {
			chunk.Usage = &Usage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
			}
		}

		if len(resp.Choices) > 0 {
			choice := resp.Choices[0]
			chunk.Delta = choice.Delta.Content
			if choice.FinishReason != nil {
				chunk.FinishReason = *choice.FinishReason
			}
		}

		ch <- chunk
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Err: fmt.Errorf("read SSE: %w", err)}
	}
}
