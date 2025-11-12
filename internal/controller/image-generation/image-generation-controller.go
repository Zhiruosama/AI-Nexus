// Package imagegeneration 图片生成controller
package imagegeneration

import (
	"net/http"

	imagegeneration_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/image-generation"
	imagegeneration_service "github.com/Zhiruosama/ai_nexus/internal/service/image-generation"
	"github.com/gin-gonic/gin"
)

// Controller 对应 Controller 结构，有一个 Service 成员
type Controller struct {
	ImageGenerationService *imagegeneration_service.Service
}

// NewController 对应 Controller 的工厂方法
func NewController(igs *imagegeneration_service.Service) *Controller {
	return &Controller{
		ImageGenerationService: igs,
	}
}

// CreateModel 添加模型
func (c *Controller) CreateModel(ctx *gin.Context) {
	var req imagegeneration_dto.ModelCreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "invalid payload",
		})
		return
	}

	if err := c.ImageGenerationService.CreateModel(ctx, &req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "success",
	})
}
