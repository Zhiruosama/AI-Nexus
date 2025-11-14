// Package imagegeneration 图像生成服务
package imagegeneration

import (
	"fmt"

	image_generation_dao "github.com/Zhiruosama/ai_nexus/internal/dao/image-generation"
	image_generation_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
	image_generation_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/image-generation"
	"github.com/gin-gonic/gin"
)

// Service 对应 imagegeneration 模块的 Service 结构
type Service struct {
	ImageGenerationDAO *image_generation_dao.DAO
}

// NewService 对应 imagegeneration 模块的 Service 工厂方法
func NewService() *Service {
	return &Service{
		ImageGenerationDAO: &image_generation_dao.DAO{},
	}
}

// CreateModel 创建模型
func (s *Service) CreateModel(ctx *gin.Context, dto *image_generation_dto.ModelCreateDTO) error {
	// 唯一性检验
	existing, err := s.ImageGenerationDAO.CheckModelExists(ctx, dto.ModelID)
	if err != nil {
		return err
	}
	if existing {
		return fmt.Errorf("model_id '%s' already exists", dto.ModelID)
	}

	model := &image_generation_do.TableImageGenerationModelsDO{
		ModelID:           dto.ModelID,
		ModelName:         dto.ModelName,
		ModelType:         dto.ModelType,
		Provider:          dto.Provider,
		Description:       dto.Description,
		Tags:              dto.Tags,
		SortOrder:         dto.SortOrder,
		IsActive:          dto.IsActive,
		IsRecommended:     dto.IsRecommended,
		ThirdPartyModelID: dto.ThirdPartyModelID,
		BaseURL:           dto.BaseURL,
		DefaultWidth:      dto.DefaultWidth,
		DefaultHeight:     dto.DefaultHeight,
		MaxWidth:          dto.MaxWidth,
		MaxHeight:         dto.MaxHeight,
		MinSteps:          dto.MinSteps,
		MaxSteps:          dto.MaxSteps,
	}

	if err := s.ImageGenerationDAO.CreateModel(ctx, model); err != nil {
		return err
	}

	return nil
}
