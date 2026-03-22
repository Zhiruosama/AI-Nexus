// Package chat 对话模块 Service
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	chat_dao "github.com/Zhiruosama/ai_nexus/internal/dao/chat"
	chat_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/chat"
	chat_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/chat"
	chat_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/chat"
	"github.com/Zhiruosama/ai_nexus/internal/pkg"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/chat"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Service 对话模块 Service
type Service struct {
	ChatDAO *chat_dao.DAO
}

// NewService 创建 Service
func NewService() *Service {
	return &Service{ChatDAO: &chat_dao.DAO{}}
}

// ==================== API Key ====================

// GetAPIKeys 获取用户所有 API Key
func (s *Service) GetAPIKeys(ctx *gin.Context, userUUID string) ([]*chat_vo.APIKeyVO, error) {
	keys, err := s.ChatDAO.GetAPIKeysByUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	result := make([]*chat_vo.APIKeyVO, len(keys))
	for i, k := range keys {
		plainKey, _ := pkg.Decrypt(k.APIKeyEnc, configs.GlobalConfig.Chat.EncryptionKey)
		result[i] = &chat_vo.APIKeyVO{
			ID:         k.ID,
			Provider:   k.Provider,
			Name:       k.Name,
			IsActive:   k.IsActive,
			BaseURL:    k.BaseURL,
			APIKeyMask: pkg.MaskAPIKey(plainKey),
			CreatedAt:  k.CreatedAt,
		}
	}
	return result, nil
}

// CreateAPIKey 创建 API Key
func (s *Service) CreateAPIKey(ctx *gin.Context, userUUID string, dto *chat_dto.CreateAPIKeyDTO) error {
	encKey, err := pkg.Encrypt(dto.APIKey, configs.GlobalConfig.Chat.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt api key: %w", err)
	}

	do := &chat_do.TableUserAPIKeyDO{
		UserUUID:  userUUID,
		Provider:  dto.Provider,
		Name:      dto.Name,
		IsActive:  true,
		BaseURL:   dto.BaseURL,
		APIKeyEnc: encKey,
	}
	return s.ChatDAO.CreateAPIKey(ctx, do)
}

// UpdateAPIKey 更新 API Key
func (s *Service) UpdateAPIKey(ctx *gin.Context, id uint64, userUUID string, dto *chat_dto.UpdateAPIKeyDTO) error {
	updates := make(map[string]any)
	if dto.Provider != nil {
		updates["provider"] = *dto.Provider
	}
	if dto.Name != nil {
		updates["name"] = *dto.Name
	}
	if dto.BaseURL != nil {
		updates["base_url"] = *dto.BaseURL
	}
	if dto.APIKey != nil {
		encKey, err := pkg.Encrypt(*dto.APIKey, configs.GlobalConfig.Chat.EncryptionKey)
		if err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
		updates["api_key_enc"] = encKey
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	return s.ChatDAO.UpdateAPIKey(ctx, id, userUUID, updates)
}

// DeleteAPIKey 删除 API Key
func (s *Service) DeleteAPIKey(ctx *gin.Context, id uint64, userUUID string) error {
	return s.ChatDAO.DeleteAPIKey(ctx, id, userUUID)
}

// ==================== Conversation ====================

// GetConversations 获取对话列表
func (s *Service) GetConversations(ctx *gin.Context, userUUID string, pageIndex, pageSize int) ([]*chat_vo.ConversationVO, int64, error) {
	offset := 0
	limit := 0
	if pageSize > 0 {
		offset = pageIndex * pageSize
		limit = pageSize
	}

	convs, total, err := s.ChatDAO.GetConversationsByUser(ctx, userUUID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*chat_vo.ConversationVO, len(convs))
	for i, c := range convs {
		result[i] = &chat_vo.ConversationVO{
			ConvID:       c.ConvID,
			APIKeyID:     c.APIKeyID,
			Title:        c.Title,
			Model:        c.Model,
			SystemPrompt: c.SystemPrompt,
			CreatedAt:    c.CreatedAt,
			UpdatedAt:    c.UpdatedAt,
		}
	}
	return result, total, nil
}

// CreateConversation 创建对话
func (s *Service) CreateConversation(ctx *gin.Context, userUUID string, dto *chat_dto.CreateConversationDTO) (*chat_vo.ConversationVO, error) {
	_, err := s.ChatDAO.GetAPIKeyByID(ctx, dto.APIKeyID, userUUID)
	if err != nil {
		return nil, fmt.Errorf("api key not found or not owned by user")
	}

	convID := uuid.New().String()
	title := dto.Title

	// 默认标题为 "新对话"，后续根据首条消息自动生成
	if title == "" {
		title = "新对话"
	}

	do := &chat_do.TableConversationDO{
		ConvID:       convID,
		UserUUID:     userUUID,
		APIKeyID:     dto.APIKeyID,
		Title:        title,
		Model:        dto.Model,
		SystemPrompt: dto.SystemPrompt,
	}
	if err := s.ChatDAO.CreateConversation(ctx, do); err != nil {
		return nil, err
	}

	return &chat_vo.ConversationVO{
		ConvID:       convID,
		APIKeyID:     dto.APIKeyID,
		Title:        title,
		Model:        dto.Model,
		SystemPrompt: dto.SystemPrompt,
		CreatedAt:    do.CreatedAt,
		UpdatedAt:    do.UpdatedAt,
	}, nil
}

// GetConversationDetail 获取对话详情含消息
func (s *Service) GetConversationDetail(ctx *gin.Context, convID, userUUID string) (*chat_vo.ConversationDetailVO, error) {
	conv, err := s.ChatDAO.GetConversationByID(ctx, convID, userUUID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found")
	}

	msgs, err := s.ChatDAO.GetMessagesByConvID(ctx, convID)
	if err != nil {
		return nil, err
	}

	msgVOs := make([]chat_vo.MessageVO, len(msgs))
	for i, m := range msgs {
		msgVOs[i] = chat_vo.MessageVO{
			ID:               m.ID,
			Role:             m.Role,
			Content:          m.Content,
			PromptTokens:     m.PromptTokens,
			CompletionTokens: m.CompletionTokens,
			CreatedAt:        m.CreatedAt,
		}
	}

	return &chat_vo.ConversationDetailVO{
		ConversationVO: chat_vo.ConversationVO{
			ConvID:       conv.ConvID,
			APIKeyID:     conv.APIKeyID,
			Title:        conv.Title,
			Model:        conv.Model,
			SystemPrompt: conv.SystemPrompt,
			CreatedAt:    conv.CreatedAt,
			UpdatedAt:    conv.UpdatedAt,
		},
		Messages: msgVOs,
	}, nil
}

// DeleteConversation 删除对话
func (s *Service) DeleteConversation(ctx *gin.Context, convID, userUUID string) error {
	return s.ChatDAO.DeleteConversation(ctx, convID, userUUID)
}

// UpdateConversationTitle 更新对话标题
func (s *Service) UpdateConversationTitle(ctx *gin.Context, convID, userUUID, title string) error {
	return s.ChatDAO.UpdateConversationTitle(ctx, convID, userUUID, title)
}

// ==================== Chat (SSE) ====================

// StreamChat 流式对话，通过 writer 写入 SSE
func (s *Service) StreamChat(ctx *gin.Context, convID, userUUID string, dto *chat_dto.SendMessageDTO, writer io.Writer, flusher func()) error {
	// 1. 获取对话信息
	conv, err := s.ChatDAO.GetConversationByID(ctx, convID, userUUID)
	if err != nil {
		return fmt.Errorf("conversation not found")
	}

	// 2. 获取并解密 API Key
	apiKeyDO, err := s.ChatDAO.GetAPIKeyByID(ctx, conv.APIKeyID, userUUID)
	if err != nil {
		return fmt.Errorf("api key not found")
	}
	plainKey, err := pkg.Decrypt(apiKeyDO.APIKeyEnc, configs.GlobalConfig.Chat.EncryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt api key: %w", err)
	}

	// 3. 组装历史消息（上下文窗口）
	maxMsg := configs.GlobalConfig.Chat.MaxMessagesPerConv
	if maxMsg <= 0 {
		maxMsg = 200
	}
	historyMsgs, err := s.ChatDAO.GetRecentMessages(convID, maxMsg)
	if err != nil {
		return fmt.Errorf("get history: %w", err)
	}

	var messages []chat.Message
	if conv.SystemPrompt != "" {
		messages = append(messages, chat.Message{Role: "system", Content: conv.SystemPrompt})
	}
	for _, m := range historyMsgs {
		messages = append(messages, chat.Message{Role: m.Role, Content: m.Content})
	}
	// 添加当前用户消息
	messages = append(messages, chat.Message{Role: "user", Content: dto.Content})

	// 4. 保存用户消息
	userMsg := &chat_do.TableConversationMessageDO{
		ConvID:  convID,
		Role:    "user",
		Content: dto.Content,
	}
	if err := s.ChatDAO.CreateMessage(userMsg); err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	// 5. 创建 Provider 并发起流式请求
	provider := chat.NewProvider(apiKeyDO.Provider, plainKey, apiKeyDO.BaseURL)

	temperature := dto.Temperature
	if temperature <= 0 {
		temperature = 0.7
	}
	maxTokens := dto.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	req := &chat.Request{
		Model:       conv.Model,
		Messages:    messages,
		Temperature: temperature,
		TopP:        dto.TopP,
		MaxTokens:   maxTokens,
	}

	streamCtx, cancel := context.WithTimeout(ctx.Request.Context(), 2*60*time.Second)
	defer cancel()

	ch, err := provider.ChatStream(streamCtx, req)
	if err != nil {
		return fmt.Errorf("start stream: %w", err)
	}

	// 6. 读取流并写入 SSE
	var contentBuilder strings.Builder
	var totalUsage chat.Usage

	for chunk := range ch {
		if chunk.Err != nil {
			errPayload, _ := json.Marshal(map[string]string{"error": chunk.Err.Error()})
			_, _ = fmt.Fprintf(writer, "data: %s\n\n", errPayload)
			flusher()
			break
		}

		if chunk.Delta != "" {
			contentBuilder.WriteString(chunk.Delta)
		}

		if chunk.Usage != nil {
			if chunk.Usage.PromptTokens > 0 {
				totalUsage.PromptTokens = chunk.Usage.PromptTokens
			}
			if chunk.Usage.CompletionTokens > 0 {
				totalUsage.CompletionTokens = chunk.Usage.CompletionTokens
			}
		}

		finishReason := ""
		if chunk.FinishReason != "" {
			finishReason = chunk.FinishReason
		}

		if chunk.Delta != "" || finishReason != "" {
			if finishReason != "" && totalUsage.PromptTokens > 0 {
				_, _ = fmt.Fprintf(writer, "data: {\"delta\":%q,\"finish_reason\":%q,\"usage\":{\"prompt_tokens\":%d,\"completion_tokens\":%d}}\n\n",
					chunk.Delta, finishReason, totalUsage.PromptTokens, totalUsage.CompletionTokens)
			} else {
				_, _ = fmt.Fprintf(writer, "data: {\"delta\":%q,\"finish_reason\":%q}\n\n", chunk.Delta, finishReason)
			}
			flusher()
		}
	}

	_, _ = fmt.Fprintf(writer, "data: [DONE]\n\n")
	flusher()

	// 7. 保存助手回复
	assistantMsg := &chat_do.TableConversationMessageDO{
		ConvID:           convID,
		Role:             "assistant",
		Content:          contentBuilder.String(),
		PromptTokens:     totalUsage.PromptTokens,
		CompletionTokens: totalUsage.CompletionTokens,
	}
	if err := s.ChatDAO.CreateMessage(assistantMsg); err != nil {
		log.Printf("[Chat] Save assistant message error: %v\n", err)
	}

	// 8. 更新对话时间
	if err := s.ChatDAO.TouchConversation(convID); err != nil {
		log.Printf("[Chat] Touch conversation error: %v\n", err)
	}

	// 9. 首条消息时自动生成标题
	msgCount, _ := s.ChatDAO.GetMessageCount(convID)
	if msgCount == 2 && conv.Title == "新对话" {
		go s.generateTitle(conv, dto.Content, contentBuilder.String(), plainKey, apiKeyDO.BaseURL, apiKeyDO.Provider)
	}

	return nil
}

// generateTitle 自动生成对话标题
func (s *Service) generateTitle(conv *chat_do.TableConversationDO, userMsg, assistantMsg, apiKey, baseURL, providerStr string) {
	provider := chat.NewProvider(providerStr, apiKey, baseURL)

	req := &chat.Request{
		Model: conv.Model,
		Messages: []chat.Message{
			{Role: "user", Content: userMsg},
			{Role: "assistant", Content: assistantMsg},
			{Role: "user", Content: "请用不超过20个字概括以上对话的主题, 直接输出标题, 不要任何额外内容。"},
		},
		Temperature: 0.3,
		MaxTokens:   60,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ch, err := provider.ChatStream(ctx, req)
	if err != nil {
		log.Printf("[Chat] Generate title error: %v\n", err)
		return
	}

	var title string
	for chunk := range ch {
		if chunk.Err != nil {
			break
		}
		title += chunk.Delta
	}

	if title != "" && len(title) <= 128 {
		if err := s.ChatDAO.TouchConversation(conv.ConvID); err != nil {
			log.Printf("[Chat] Touch conversation for title error: %v\n", err)
		}
		err := s.ChatDAO.UpdateConversationTitle(nil, conv.ConvID, conv.UserUUID, title)
		if err != nil {
			log.Printf("[Chat] Update title error: %v\n", err)
		}
	}
}

// ==================== Preset ====================

// CreatePreset 创建预设
func (s *Service) CreatePreset(ctx *gin.Context, userUUID string, dto *chat_dto.CreatePresetDTO) error {
	do := &chat_do.TableChatPresetDO{
		UserUUID: userUUID,
		Name:     dto.Name,
		Content:  dto.Content,
	}
	return s.ChatDAO.CreatePreset(ctx, do)
}

// GetPresets 获取预设列表
func (s *Service) GetPresets(ctx *gin.Context, userUUID string) ([]*chat_vo.PresetVO, error) {
	presets, err := s.ChatDAO.GetPresetsByUser(ctx, userUUID)
	if err != nil {
		return nil, err
	}

	result := make([]*chat_vo.PresetVO, len(presets))
	for i, p := range presets {
		result[i] = &chat_vo.PresetVO{
			ID:        p.ID,
			Name:      p.Name,
			Content:   p.Content,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
	}
	return result, nil
}

// UpdatePreset 更新预设
func (s *Service) UpdatePreset(ctx *gin.Context, id uint64, userUUID string, dto *chat_dto.UpdatePresetDTO) error {
	updates := make(map[string]any)
	if dto.Name != nil {
		updates["name"] = *dto.Name
	}
	if dto.Content != nil {
		updates["content"] = *dto.Content
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	return s.ChatDAO.UpdatePreset(ctx, id, userUUID, updates)
}

// DeletePreset 删除预设
func (s *Service) DeletePreset(ctx *gin.Context, id uint64, userUUID string) error {
	return s.ChatDAO.DeletePreset(ctx, id, userUUID)
}
