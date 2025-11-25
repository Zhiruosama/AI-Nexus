// Package imagegeneration 图像生成服务
package imagegeneration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/Zhiruosama/ai_nexus/configs"
	image_generation_dao "github.com/Zhiruosama/ai_nexus/internal/dao/image-generation"
	image_generation_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
	image_generation_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/image-generation"
	image_generation_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/image-generation"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	rabbitmq "github.com/Zhiruosama/ai_nexus/internal/pkg/queue"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	updates := make(map[string]any)

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

// QueryModels 根据具体信息查询模型列表
func (s *Service) QueryModels(ctx *gin.Context, query *image_generation_query.ModelsQuery) ([]*image_generation_do.TableImageGenerationModelsDO, int64, error) {
	models, total, err := s.ImageGenerationDAO.QueryModels(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	if models == nil {
		models = []*image_generation_do.TableImageGenerationModelsDO{}
	}

	return models, total, nil
}

// Text2Img 文生图
func (s *Service) Text2Img(ctx *gin.Context, dto *image_generation_dto.Text2ImgDTO) (string, error) {
	ok, err := s.ImageGenerationDAO.CheckModelExists(ctx, dto.ModelID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("model_id '%s' does not exist", dto.ModelID)
	}

	taskID := uuid.New().String()
	uuid, _ := ctx.Get("user_id")

	do := image_generation_do.TableImageGenerationTaskDO{
		TaskID:            taskID,
		UserUUID:          uuid.(string),
		TaskType:          1,
		Status:            0,
		Prompt:            dto.Prompt,
		NegativePrompt:    dto.NegativePrompt,
		ModelID:           dto.ModelID,
		Width:             dto.Width,
		Height:            dto.Height,
		NumInferenceSteps: dto.NumInferenceSteps,
		GuidanceScale:     dto.GuidanceScale,
		Seed:              dto.Seed,
	}

	if err := s.ImageGenerationDAO.CreateText2ImgTask(ctx, &do); err != nil {
		return "", err
	}

	message := rabbitmq.TaskMessage{
		TaskID:   taskID,
		UserUUID: uuid.(string),
		Payload: rabbitmq.Text2ImgPayload{
			Prompt:            dto.Prompt,
			NegativePrompt:    dto.NegativePrompt,
			ModelID:           dto.ModelID,
			Width:             dto.Width,
			Height:            dto.Height,
			NumInferenceSteps: dto.NumInferenceSteps,
			GuidanceScale:     dto.GuidanceScale,
			Seed:              dto.Seed,
		},
	}

	c, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := rabbitmq.Publish(c, 1, &message); err != nil {
		logger.Error(ctx, "Publish to mq error(text2img): %s", err.Error())

		errs := s.ImageGenerationDAO.DeleteTask(ctx, taskID)
		if errs != nil {
			logger.Error(ctx, "DeleteTask error: %s", errs.Error())
		}

		return "", err
	}

	now := time.Now()
	mysqlDatetime := now.Format("2006-01-02 15:04:05")
	if err := s.ImageGenerationDAO.UpdateTaskParams("queued_at", mysqlDatetime, taskID); err != nil {
		return "", err
	}

	if err := s.ImageGenerationDAO.UpdateTaskParams("status", 1, taskID); err != nil {
		return "", err
	}

	return taskID, nil
}

// Img2Img 图生图
func (s *Service) Img2Img(ctx *gin.Context, dto *image_generation_dto.Img2ImgDTO) (string, error) {
	ok, err := s.ImageGenerationDAO.CheckModelExists(ctx, dto.ModelID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("model_id '%s' does not exist", dto.ModelID)
	}

	taskID := uuid.New().String()
	uuid, _ := ctx.Get("user_id")

	// 落盘
	ext := filepath.Ext(dto.InputImage.Filename)
	allowedExts := []string{".png", ".jpg", ".jpeg", ".webp"}
	isValid := slices.Contains(allowedExts, ext)
	if !isValid {
		return "", fmt.Errorf("unsupported file format: %s", ext)
	}

	if err = os.MkdirAll(filepath.Join("static", "images"), 0755); err != nil {
		return "", err
	}

	filename := "img2img-" + taskID + ext
	dst := filepath.Join("static", "images", filename)

	if err = ctx.SaveUploadedFile(dto.InputImage, dst); err != nil {
		return "", err
	}

	// 落盘后进行校验
	file, err := os.Open(dst)
	if err != nil {
		errs := os.Remove(dst)
		if errs != nil {
			logger.Error(ctx, "Remove uploaded file error: %s", errs.Error())
		}
		return "", err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Error(ctx, "Close uploaded file error: %s", closeErr.Error())
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		logger.Error(ctx, "calcute sha256 error: %s", err.Error())
		errs := os.Remove(dst)
		if errs != nil {
			logger.Error(ctx, "Remove uploaded file error: %s", errs.Error())
		}
		return "", err
	}

	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if calculatedHash != dto.Sha256 {
		errs := os.Remove(dst)
		if errs != nil {
			logger.Error(ctx, "Remove uploaded file error: %s", errs.Error())
		}
		return "", fmt.Errorf("the file destroyed")
	}

	inputImageURL := fmt.Sprintf("http://%s/static/images/%s", configs.GlobalConfig.Server.SerialStringPublic(), filename)

	do := image_generation_do.TableImageGenerationTaskDO{
		TaskID:            taskID,
		UserUUID:          uuid.(string),
		TaskType:          2,
		Status:            0,
		Prompt:            dto.Prompt,
		NegativePrompt:    dto.NegativePrompt,
		ModelID:           dto.ModelID,
		Width:             dto.Width,
		Height:            dto.Height,
		NumInferenceSteps: dto.NumInferenceSteps,
		GuidanceScale:     dto.GuidanceScale,
		Seed:              dto.Seed,
		InputImageURL:     inputImageURL,
		Strength:          dto.Strength,
	}

	if err := s.ImageGenerationDAO.CreateImg2ImgTask(ctx, &do); err != nil {
		return "", err
	}

	message := rabbitmq.TaskMessage{
		TaskID:   taskID,
		UserUUID: uuid.(string),
		Payload: rabbitmq.Img2ImgPayload{
			Prompt:            dto.Prompt,
			NegativePrompt:    dto.NegativePrompt,
			ModelID:           dto.ModelID,
			Width:             dto.Width,
			Height:            dto.Height,
			NumInferenceSteps: dto.NumInferenceSteps,
			GuidanceScale:     dto.GuidanceScale,
			Seed:              dto.Seed,
			InputImageURL:     inputImageURL,
			Strength:          dto.Strength,
		},
	}

	c, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	if err := rabbitmq.Publish(c, 2, &message); err != nil {
		logger.Error(ctx, "Publish to mq error(img2img): %s", err.Error())

		errs := s.ImageGenerationDAO.DeleteTask(ctx, taskID)
		if errs != nil {
			logger.Error(ctx, "DeleteTask error: %s", errs.Error())
		}

		return "", err
	}

	now := time.Now()
	mysqlDatetime := now.Format("2006-01-02 15:04:05")
	if err := s.ImageGenerationDAO.UpdateTaskParams("queued_at", mysqlDatetime, taskID); err != nil {
		return "", err
	}
	if err := s.ImageGenerationDAO.UpdateTaskParams("status", 1, taskID); err != nil {
		return "", err
	}

	return taskID, nil
}

// CancelTask 取消任务
func (s *Service) CancelTask(ctx *gin.Context, taskID string) error {
	ok, err := s.ImageGenerationDAO.CheckTaskExists(ctx, taskID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("task_id '%s' does not exist", taskID)
	}

	// 如果是img2img任务 则删除前面保存到服务器里的图片
	taskType, err := image_generation_dao.GetTaskInfo[int8](s.ImageGenerationDAO, "task_type", taskID)
	if err != nil {
		return err
	}

	if taskType == 2 {
		inputURL, err := image_generation_dao.GetTaskInfo[string](s.ImageGenerationDAO, "input_image_url", taskID)
		if err == nil && inputURL != "" {
			u, err := url.Parse(inputURL)
			if err != nil {
				return err
			}
			filename := filepath.Base(u.Path)
			dst := filepath.Join("static", "images", filename)
			if err = os.Remove(dst); err != nil && !os.IsNotExist(err) {
				logger.Error(ctx, "Remove img2img input image error: %s", err.Error())
				return err
			}
		}
	}

	if err := s.ImageGenerationDAO.UpdateTaskParams("status", 5, taskID); err != nil {
		return err
	}
	if err := s.ImageGenerationDAO.UpdateTaskParams("input_image_url", "", taskID); err != nil {
		return err
	}

	return nil
}
