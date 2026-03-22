// Package chat 对话模块路由
package chat

import (
	chat_controller "github.com/Zhiruosama/ai_nexus/internal/controller/chat"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	chat_service "github.com/Zhiruosama/ai_nexus/internal/service/chat"
	"github.com/gin-gonic/gin"
)

// InitChatRoutes 初始化对话模块路由
func InitChatRoutes(r *gin.Engine) {
	s := chat_service.NewService()
	c := chat_controller.NewController(s)

	chatGroup := r.Group("/chat")
	chatGroup.Use(middleware.AuthMiddleware(), middleware.RateLimitingMiddleware())
	{
		// API Key 管理
		apiKeys := chatGroup.Group("/api-keys")
		{
			apiKeys.GET("", c.GetAPIKeys)
			apiKeys.POST("", c.CreateAPIKey)
			apiKeys.PUT("/:id", c.UpdateAPIKey)
			apiKeys.DELETE("/:id", c.DeleteAPIKey)
		}

		// 对话管理
		convs := chatGroup.Group("/conversations")
		{
			convs.GET("", c.GetConversations)
			convs.POST("", c.CreateConversation)
			convs.GET("/:conv_id", c.GetConversationDetail)
			convs.DELETE("/:conv_id", c.DeleteConversation)
			convs.PUT("/:conv_id/title", c.UpdateConversationTitle)
			convs.POST("/:conv_id/messages", c.SendMessage)
		}

		// 预设管理
		presets := chatGroup.Group("/presets")
		{
			presets.POST("", c.CreatePreset)
			presets.GET("", c.GetPresets)
			presets.PUT("/:id", c.UpdatePreset)
			presets.DELETE("/:id", c.DeletePreset)
		}
	}
}
