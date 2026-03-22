// Package chat 对话模块 Controller
package chat

import (
	"net/http"
	"strconv"

	chat_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/chat"
	chat_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/chat"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	chat_service "github.com/Zhiruosama/ai_nexus/internal/service/chat"
	"github.com/gin-gonic/gin"
)

// Controller 对话模块 Controller
type Controller struct {
	ChatService *chat_service.Service
}

// NewController 创建 Controller
func NewController(s *chat_service.Service) *Controller {
	return &Controller{ChatService: s}
}

func getUserUUID(ctx *gin.Context) string {
	uid, ok := ctx.Get(middleware.UserIDKey)
	if !ok {
		return ""
	}
	return uid.(string)
}

// ==================== API Key ====================

// GetAPIKeys 获取 API Key 列表
func (c *Controller) GetAPIKeys(ctx *gin.Context) {
	keys, err := c.ChatService.GetAPIKeys(ctx, getUserUUID(ctx))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "data": keys})
}

// CreateAPIKey 创建 API Key
func (c *Controller) CreateAPIKey(ctx *gin.Context) {
	var dto chat_dto.CreateAPIKeyDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	validProviders := map[string]bool{"openai": true, "anthropic": true, "gemini": true, "custom": true}
	if !validProviders[dto.Provider] {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "provider 必须是 openai/anthropic/gemini/custom"})
		return
	}

	if dto.Provider == "custom" && dto.BaseURL == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "custom provider 必须填写 base_url"})
		return
	}

	if err := c.ChatService.CreateAPIKey(ctx, getUserUUID(ctx), &dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "创建成功"})
}

// UpdateAPIKey 更新 API Key
func (c *Controller) UpdateAPIKey(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "无效的 ID"})
		return
	}

	var dto chat_dto.UpdateAPIKeyDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	if err := c.ChatService.UpdateAPIKey(ctx, id, getUserUUID(ctx), &dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "更新成功"})
}

// DeleteAPIKey 删除 API Key
func (c *Controller) DeleteAPIKey(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "无效的 ID"})
		return
	}

	if err := c.ChatService.DeleteAPIKey(ctx, id, getUserUUID(ctx)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "删除成功"})
}

// ==================== Conversation ====================

// GetConversations 获取对话列表
func (c *Controller) GetConversations(ctx *gin.Context) {
	var query chat_query.ConversationListQuery
	_ = ctx.ShouldBindQuery(&query)

	convs, total, err := c.ChatService.GetConversations(ctx, getUserUUID(ctx), query.PageIndex, query.PageSize)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "data": convs, "total": total})
}

// CreateConversation 创建对话
func (c *Controller) CreateConversation(ctx *gin.Context) {
	var dto chat_dto.CreateConversationDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	conv, err := c.ChatService.CreateConversation(ctx, getUserUUID(ctx), &dto)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "data": conv})
}

// GetConversationDetail 获取对话详情
func (c *Controller) GetConversationDetail(ctx *gin.Context) {
	convID := ctx.Param("conv_id")
	if convID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "conv_id 不能为空"})
		return
	}

	detail, err := c.ChatService.GetConversationDetail(ctx, convID, getUserUUID(ctx))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "data": detail})
}

// DeleteConversation 删除对话
func (c *Controller) DeleteConversation(ctx *gin.Context) {
	convID := ctx.Param("conv_id")

	if err := c.ChatService.DeleteConversation(ctx, convID, getUserUUID(ctx)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "删除成功"})
}

// UpdateConversationTitle 修改对话标题
func (c *Controller) UpdateConversationTitle(ctx *gin.Context) {
	convID := ctx.Param("conv_id")
	var body struct {
		Title string `json:"title" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	if err := c.ChatService.UpdateConversationTitle(ctx, convID, getUserUUID(ctx), body.Title); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "更新成功"})
}

// ==================== Chat SSE ====================

// SendMessage 发送消息（SSE 流式返回）
func (c *Controller) SendMessage(ctx *gin.Context) {
	convID := ctx.Param("conv_id")
	if convID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "conv_id 不能为空"})
		return
	}

	var dto chat_dto.SendMessageDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	// 设置 SSE 响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	writer := ctx.Writer
	flusher := func() { writer.Flush() }

	if err := c.ChatService.StreamChat(ctx, convID, getUserUUID(ctx), &dto, writer, flusher); err != nil {
		// 如果 SSE 头已发送则通过 data 返回错误
		if writer.Written() {
			ctx.SSEvent("error", err.Error())
			writer.Flush()
		} else {
			ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		}
	}
}

// ==================== Preset ====================

// CreatePreset 创建预设
func (c *Controller) CreatePreset(ctx *gin.Context) {
	var dto chat_dto.CreatePresetDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	if err := c.ChatService.CreatePreset(ctx, getUserUUID(ctx), &dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "创建成功"})
}

// GetPresets 获取预设列表
func (c *Controller) GetPresets(ctx *gin.Context) {
	presets, err := c.ChatService.GetPresets(ctx, getUserUUID(ctx))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "data": presets})
}

// UpdatePreset 更新预设
func (c *Controller) UpdatePreset(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "无效的 ID"})
		return
	}

	var dto chat_dto.UpdatePresetDTO
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "参数错误"})
		return
	}

	if err := c.ChatService.UpdatePreset(ctx, id, getUserUUID(ctx), &dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "更新成功"})
}

// DeletePreset 删除预设
func (c *Controller) DeletePreset(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": "无效的 ID"})
		return
	}

	if err := c.ChatService.DeletePreset(ctx, id, getUserUUID(ctx)); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"code": middleware.ParamEmpty, "message": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": http.StatusOK, "message": "删除成功"})
}
