// Package chat Anthropic Claude Provider 实现
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

const defaultAnthropicBaseURL = "https://api.anthropic.com/"

// AnthropicProvider Anthropic Claude Provider
type AnthropicProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewAnthropicProvider 创建 Anthropic Provider
func NewAnthropicProvider(apiKey, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = defaultAnthropicBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Stream      bool               `json:"stream"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicEvent struct {
	Type  string          `json:"type"`
	Delta json.RawMessage `json:"delta,omitempty"`
	Usage json.RawMessage `json:"usage,omitempty"`
}

type anthropicContentDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicErrorEvent struct {
	Error *anthropicAPIError `json:"error,omitempty"`
}

type anthropicAPIError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

// ChatStream 发起流式对话
func (p *AnthropicProvider) ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error) {
	var systemPrompt string
	var messages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemPrompt = m.Content
			continue
		}
		messages = append(messages, anthropicMessage(m))
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	body := anthropicRequest{
		Model:       req.Model,
		Messages:    messages,
		System:      systemPrompt,
		MaxTokens:   maxTokens,
		Stream:      true,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			log.Printf("[Anthropic] close error response body: %v\n", err)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan StreamChunk, 32)
	go func() {
		defer close(ch)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("[Anthropic] close stream body: %v\n", err)
			}
		}()
		p.readSSE(ctx, resp.Body, ch)
	}()
	return ch, nil
}

func (p *AnthropicProvider) readSSE(ctx context.Context, body io.Reader, ch chan<- StreamChunk) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var eventType string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- StreamChunk{Err: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()

		if after, found := strings.CutPrefix(line, "event: "); found {
			eventType = after
			continue
		}

		data, found := strings.CutPrefix(line, "data: ")
		if !found {
			continue
		}

		switch eventType {
		case "ping", "content_block_start", "content_block_stop":
			continue

		case "content_block_delta":
			var evt anthropicEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("unmarshal delta: %w", err)}
				return
			}
			var delta anthropicContentDelta
			if err := json.Unmarshal(evt.Delta, &delta); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("unmarshal content delta: %w", err)}
				return
			}
			ch <- StreamChunk{Delta: delta.Text}

		case "message_delta":
			// 可能包含 stop_reason
			var raw map[string]any
			if err := json.Unmarshal([]byte(data), &raw); err == nil {
				chunk := StreamChunk{}
				if deltaMap, ok := raw["delta"].(map[string]any); ok {
					if sr, ok := deltaMap["stop_reason"].(string); ok {
						chunk.FinishReason = sr
					}
				}
				if usageRaw, ok := raw["usage"]; ok {
					usageBytes, _ := json.Marshal(usageRaw)
					var usage anthropicUsage
					if json.Unmarshal(usageBytes, &usage) == nil {
						chunk.Usage = &Usage{
							PromptTokens:     usage.InputTokens,
							CompletionTokens: usage.OutputTokens,
						}
					}
				}
				ch <- chunk
			}

		case "message_start":
			// 提取 input_tokens
			var raw map[string]any
			if err := json.Unmarshal([]byte(data), &raw); err == nil {
				if msg, ok := raw["message"].(map[string]any); ok {
					if usageRaw, ok := msg["usage"]; ok {
						usageBytes, _ := json.Marshal(usageRaw)
						var usage anthropicUsage
						if json.Unmarshal(usageBytes, &usage) == nil && usage.InputTokens > 0 {
							ch <- StreamChunk{
								Usage: &Usage{PromptTokens: usage.InputTokens},
							}
						}
					}
				}
			}

		case "message_stop":
			return

		case "error":
			var evt anthropicErrorEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				ch <- StreamChunk{Err: fmt.Errorf("anthropic stream error event: %s", data)}
				return
			}
			if evt.Error != nil {
				if evt.Error.Type != "" {
					ch <- StreamChunk{Err: fmt.Errorf("anthropic %s: %s", evt.Error.Type, evt.Error.Message)}
				} else {
					ch <- StreamChunk{Err: fmt.Errorf("anthropic error: %s", evt.Error.Message)}
				}
				return
			}
			ch <- StreamChunk{Err: fmt.Errorf("anthropic stream error event: %s", data)}
			return

		default:
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Err: fmt.Errorf("read SSE: %w", err)}
	}
}
