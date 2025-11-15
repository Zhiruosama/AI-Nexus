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

// DeleteModel 删除模型
func (s *Service) DeleteModel(ctx *gin.Context, modelID string) error {
	existing, err := s.ImageGenerationDAO.CheckModelExists(ctx, modelID)
	if err != nil {
		return err
	}
	if !existing {
		return fmt.Errorf("model_id '%s' does not exist", modelID)
	}
	if err := s.ImageGenerationDAO.DeleteModel(ctx, modelID); err != nil {
		return err
	}

	return nil
}

// GetModelInfo 获取模型信息
func (s *Service) GetModelInfo(ctx *gin.Context, modelID string) (*image_generation_do.TableImageGenerationModelsDO, error) {
	if modelID == "" {
		return nil, fmt.Errorf("model_id is required")
	}

	existing, err := s.ImageGenerationDAO.CheckModelExists(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if !existing {
		return nil, fmt.Errorf("model_id '%s' does not exist", modelID)
	}

	model, err := s.ImageGenerationDAO.GetModelInfo(ctx, modelID)
	if err != nil {
		return nil, err
	}
	return model, err
}

// UpdateModel 更新模型数据
func (s *Service) UpdateModel(ctx *gin.Context, dto *image_generation_dto.ModelUpdateDTO) error {
	// 唯一性检验
	existing, err := s.ImageGenerationDAO.CheckModelExists(ctx, dto.ModelID)
	if err != nil {
		return err
	}
	if !existing {
		return fmt.Errorf("model_id '%s' does not exist", dto.ModelID)
	}

	updates := make(map[string]interface{})
	if dto.ModelName != nil {
		updates["model_name"] = *dto.ModelName
	}
	if dto.ModelType != nil {
		if *dto.ModelType != "text2img" && *dto.ModelType != "img2img" {
			return fmt.Errorf("model_type must be either 'text2img' or 'img2img'")
		}
		updates["model_type"] = *dto.ModelType
	}
	if dto.Provider != nil {
		updates["provider"] = *dto.Provider
	}
	if dto.Description != nil {
		updates["description"] = *dto.Description
	}
	if dto.Tags != nil {
		updates["tags"] = *dto.Tags
	}
	if dto.SortOrder != nil {
		updates["sort_order"] = *dto.SortOrder
	}
	if dto.IsActive != nil {
		updates["is_active"] = *dto.IsActive
	}
	if dto.IsRecommended != nil {
		updates["is_recommended"] = *dto.IsRecommended
	}
	if dto.ThirdPartyModelID != nil {
		updates["third_party_model_id"] = *dto.ThirdPartyModelID
	}
	if dto.BaseURL != nil {
		updates["base_url"] = *dto.BaseURL
	}
	if dto.DefaultWidth != nil {
		updates["default_width"] = *dto.DefaultWidth
	}
	if dto.DefaultHeight != nil {
		updates["default_height"] = *dto.DefaultHeight
	}
	if dto.MaxWidth != nil {
		updates["max_width"] = *dto.MaxWidth
	}
	if dto.MaxHeight != nil {
		updates["max_height"] = *dto.MaxHeight
	}
	if dto.MinSteps != nil {
		updates["min_steps"] = *dto.MinSteps
	}
	if dto.MaxSteps != nil {
		updates["max_steps"] = *dto.MaxSteps
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	if dto.DefaultWidth != nil && dto.MaxWidth != nil && *dto.DefaultWidth > *dto.MaxWidth {
		return fmt.Errorf("default_width must not be greater than max_width")
	}
	if dto.DefaultHeight != nil && dto.MaxHeight != nil && *dto.DefaultHeight > *dto.MaxHeight {
		return fmt.Errorf("default_height must not be greater than max_height")
	}
	if dto.MinSteps != nil && dto.MaxSteps != nil && *dto.MinSteps > *dto.MaxSteps {
		return fmt.Errorf("min_steps must not be greater than max_steps")
	}

	if err := s.ImageGenerationDAO.UpdateModel(ctx, dto.ModelID, updates); err != nil {
		return err
	}

	return nil
}

// QueryModelIDs 根据具体信息查询模型
func (s *Service) QueryModelIDs(ctx *gin.Context, filters map[string]interface{}, q string) ([]string, error) {
	return s.ImageGenerationDAO.QueryModelIDs(ctx, filters, q)
}
