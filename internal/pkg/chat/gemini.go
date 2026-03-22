// Package chat Google Gemini Provider 实现
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

const defaultGeminiBaseURL = "https://generativelanguage.googleapis.com/"

// GeminiProvider Google Gemini Provider
type GeminiProvider struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeminiProvider 创建 Gemini Provider
func NewGeminiProvider(apiKey, baseURL string) *GeminiProvider {
	if baseURL == "" {
		baseURL = defaultGeminiBaseURL
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &GeminiProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

type geminiRequest struct {
	Contents          []geminiContent         `json:"contents"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"topP,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiStreamResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
}

// ChatStream 发起流式对话
func (p *GeminiProvider) ChatStream(ctx context.Context, req *Request) (<-chan StreamChunk, error) {
	var contents []geminiContent
	var systemInstruction *geminiContent

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemInstruction = &geminiContent{
				Parts: []geminiPart{{Text: m.Content}},
			}
			continue
		}
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	body := geminiRequest{
		Contents:          contents,
		SystemInstruction: systemInstruction,
	}

	if req.Temperature > 0 || req.TopP > 0 || req.MaxTokens > 0 {
		body.GenerationConfig = &geminiGenerationConfig{
			Temperature:     req.Temperature,
			TopP:            req.TopP,
			MaxOutputTokens: req.MaxTokens,
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%sv1beta/models/%s:streamGenerateContent?alt=sse", p.baseURL, req.Model)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("x-goog-api-key", p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		if err := resp.Body.Close(); err != nil {
			log.Printf("[Gemini] close error response body: %v\n", err)
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	ch := make(chan StreamChunk, 32)
	go func() {
		defer close(ch)
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Printf("[Gemini] close stream body: %v\n", err)
			}
		}()
		p.readSSE(ctx, resp.Body, ch)
	}()
	return ch, nil
}

func (p *GeminiProvider) readSSE(ctx context.Context, body io.Reader, ch chan<- StreamChunk) {
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

		var resp geminiStreamResponse
		if err := json.Unmarshal([]byte(data), &resp); err != nil {
			ch <- StreamChunk{Err: fmt.Errorf("unmarshal SSE: %w", err)}
			return
		}

		chunk := StreamChunk{}

		if len(resp.Candidates) > 0 {
			candidate := resp.Candidates[0]
			if len(candidate.Content.Parts) > 0 {
				chunk.Delta = candidate.Content.Parts[0].Text
			}
			if candidate.FinishReason != "" && candidate.FinishReason != "STOP" {
				chunk.FinishReason = strings.ToLower(candidate.FinishReason)
			} else if candidate.FinishReason == "STOP" {
				chunk.FinishReason = "stop"
			}
		}

		if resp.UsageMetadata != nil {
			chunk.Usage = &Usage{
				PromptTokens:     resp.UsageMetadata.PromptTokenCount,
				CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			}
		}

		ch <- chunk
	}
	if err := scanner.Err(); err != nil {
		ch <- StreamChunk{Err: fmt.Errorf("read SSE: %w", err)}
	}
}
