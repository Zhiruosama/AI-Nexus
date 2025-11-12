// Package imagegeneration 图像生成服务
package imagegeneration

import (
	"fmt"

	imagegeneration_dao "github.com/Zhiruosama/ai_nexus/internal/dao/image-generation"
	imagegeneration_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
	imagegeneration_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/image-generation"
	"github.com/gin-gonic/gin"
)

// Service 对应 imagegeneration 模块的 Service 结构
type Service struct {
	ImageGenerationDAO *imagegeneration_dao.DAO
}

// NewService 对应 imagegeneration 模块的 Service 工厂方法
func NewService() *Service {
	return &Service{
		ImageGenerationDAO: &imagegeneration_dao.DAO{},
	}
}

// CreateModel 创建模型
func (s *Service) CreateModel(ctx *gin.Context, dto *imagegeneration_dto.ModelCreateRequest) error {
	// 唯一性检验
	existing, err := s.ImageGenerationDAO.CheckModelExists(ctx, dto.ModelID)
	if err != nil {
		return err
	}
	if existing {
		return fmt.Errorf("model_id '%s' already exists", dto.ModelID)
	}

	if dto.DefaultHeight <= 0 || dto.DefaultWidth <= 0 {
		return fmt.Errorf("DeafultHeight or DefaultWidth must be greater than or equal to 0")
	}
	if dto.MaxWidth > 0 || dto.MaxHeight > 0 {
		if dto.DefaultWidth > dto.MaxWidth || dto.DefaultHeight > dto.MaxHeight {
			return fmt.Errorf("DefaultWidth and DefaultHeight cannot exceed the MaxWidth and MaxHeight")
		}
	}
	if dto.MinSteps > 0 && dto.MaxSteps > 0 && dto.MinSteps > dto.MaxSteps {
		return fmt.Errorf("MinSteps cannot be greater than MaxSteps")
	}

	model := &imagegeneration_do.TableImageGenerationModelsDO{
		ModelID:           dto.ModelID,
		ModelName:         dto.ModelName,
		ModelType:         dto.ModelType,
		Provider:          dto.Provider,
		Description:       dto.Description,
		Tags:              dto.Tags,
		SortOrder:         dto.SortOrder,
		ThirdPartyModelID: dto.ThirdPartyModelID,
		BaseUrl:           dto.BaseUrl,
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
