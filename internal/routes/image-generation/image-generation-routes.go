// Package imagegeneration 图像生成路由
package imagegeneration

import (
	image_generation_controller "github.com/Zhiruosama/ai_nexus/internal/controller/image-generation"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	image_generation_service "github.com/Zhiruosama/ai_nexus/internal/service/image-generation"
	"github.com/gin-gonic/gin"
)

// InitImageGenerationRoutes 初始化图像生成模块的路由
func InitImageGenerationRoutes(r *gin.Engine) {
	igs := image_generation_service.NewService()
	igc := image_generation_controller.NewController(igs)

	image_generation := r.Group("/image-generation")
	{
		model := image_generation.Group("/model")
		model.Use(middleware.RateLimitingMiddleware(), middleware.DeduplicationMiddleware())
		{
			model.POST("/create", igc.CreateModel)
			model.POST("/batchcreate", igc.BatchCreateModels)
			model.DELETE("/delete", igc.DeleteModel)
			model.PUT("/update", igc.UpdateModel)
			model.GET("/info", igc.GetModelInfo)
			model.GET("/queryids", igc.QueryModelIDs)
		}
	}
}
