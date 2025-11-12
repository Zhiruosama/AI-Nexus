// Package imagegeneration 图像生成路由
package imagegeneration

import (
	imagegeneration_controller "github.com/Zhiruosama/ai_nexus/internal/controller/image-generation"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	imagegeneration_service "github.com/Zhiruosama/ai_nexus/internal/service/image-generation"
	"github.com/gin-gonic/gin"
)

// InitImageGenerationRoutes 初始化图像生成模块的路由
func InitImageGenerationRoutes(r *gin.Engine) {
	igs := imagegeneration_service.NewService()
	igc := imagegeneration_controller.NewController(igs)

	imagegeneration := r.Group("/image-generation")
	{
		imagegeneration.POST("/create-model", middleware.RateLimitingMiddleware(), middleware.DeduplicationMiddleware(), igc.CreateModel)
	}
}
