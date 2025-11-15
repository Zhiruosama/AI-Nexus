// Package imagegeneration 图片生成controller
package imagegeneration

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	image_generation_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/image-generation"
	image_generation_service "github.com/Zhiruosama/ai_nexus/internal/service/image-generation"
	"github.com/gin-gonic/gin"
)

// Controller 对应 Controller 结构，有一个 Service 成员
type Controller struct {
	ImageGenerationService *image_generation_service.Service
}

// NewController 对应 Controller 的工厂方法
func NewController(igs *image_generation_service.Service) *Controller {
	return &Controller{
		ImageGenerationService: igs,
	}
}

// CreateModel 添加模型
func (c *Controller) CreateModel(ctx *gin.Context) {
	var req image_generation_dto.ModelCreateDTO

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	// 校验参数
	if err := validateModelCreateRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
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
		"message": "create model success",
	})
}

// BatchCreateModels 批量添加模型
func (c *Controller) BatchCreateModels(ctx *gin.Context) {
	var req image_generation_dto.BatchCreateModelsDTO

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	// 校验每个模型参数
	for _, model := range req.Models {
		if err := validateModelCreateRequest(&model); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
		}
	}

	// 批量添加模型
	for _, model := range req.Models {
		if err := c.ImageGenerationService.CreateModel(ctx, &model); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "create model success",
	})
}

// DeleteModel 删除模型
func (c *Controller) DeleteModel(ctx *gin.Context) {
	idsParam := ctx.Query("ids")
	if idsParam != "" {
		parts := strings.Split(idsParam, ",")
		seen := make(map[string]struct{})
		ids := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				ids = append(ids, p)
			}
		}
		if len(ids) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "ids is required",
			})
			return
		}
		for _, model_id := range ids {
			if err := c.ImageGenerationService.DeleteModel(ctx, model_id); err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{
					"code":    http.StatusBadRequest,
					"message": err.Error(),
				})
				return
			}
		}
		ctx.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "delete models success",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "delete model success",
	})
}

// UpdateModel 更新模型数据
func (c *Controller) UpdateModel(ctx *gin.Context) {
	var req image_generation_dto.ModelUpdateDTO
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "The input data does not meet the requirements.",
		})
		return
	}
	if req.ModelID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "model_id is required",
		})
		return
	}
	if err := c.ImageGenerationService.UpdateModel(ctx, &req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "update model success",
	})
}

// GetModelInfo 获取模型数据
func (c *Controller) GetModelInfo(ctx *gin.Context) {
	modelID := ctx.Query("model_id")
	if modelID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "model_id is required",
		})
		return
	}

	model, err := c.ImageGenerationService.GetModelInfo(ctx, modelID)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "get modelinfo success",
		"data":    model,
	})
}

// QueryModelIDs 根据具体信息查询模型ID
func (c *Controller) QueryModelIDs(ctx *gin.Context) {
	modelName := ctx.Query("model_name")
	modelType := ctx.Query("model_type")
	isActiveStr := ctx.Query("is_active")
	isRecommendedStr := ctx.Query("is_recommended")
	// q 全文关键字（在 model_name/description/tags 上做 OR 模糊）
	q := ctx.Query("q")

	filters := map[string]interface{}{}
	if modelName != "" {
		filters["model_name"] = modelName
	}
	if modelType != "" {
		filters["model_type"] = modelType
	}
	if isActiveStr != "" {
		b, _ := strconv.ParseBool(isActiveStr)
		filters["is_active"] = b
	}
	if isRecommendedStr != "" {
		b, _ := strconv.ParseBool(isRecommendedStr)
		filters["is_recommended"] = b
	}

	modelIDs, err := c.ImageGenerationService.QueryModelIDs(ctx, filters, q)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "query modelids success",
		"data":    modelIDs,
	})
}

// validateModelCreateRequest 校验创建模型请求参数
func validateModelCreateRequest(req *image_generation_dto.ModelCreateDTO) error {
	// 参数校验
	if req.ModelID == "" {
		return fmt.Errorf("model_id is required")
	}
	if len(req.ModelID) > 64 {
		return fmt.Errorf("model_id must not exceed 64 characters")
	}

	if req.ModelName == "" {
		return fmt.Errorf("model_name is required")
	}
	if len(req.ModelName) > 128 {
		return fmt.Errorf("model_name must not exceed 128 characters")
	}

	if req.ModelType == "" {
		return fmt.Errorf("model_type is required")
	}
	if req.ModelType != "text2img" && req.ModelType != "img2img" {
		return fmt.Errorf("model_type must be either 'text2img' or 'img2img'")
	}

	if req.Provider == "" {
		return fmt.Errorf("provider is required")
	}
	if len(req.Provider) > 32 {
		return fmt.Errorf("provider must not exceed 32 characters")
	}

	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(req.Description) > 500 {
		return fmt.Errorf("description must not exceed 500 characters")
	}

	if req.Tags == "" {
		return fmt.Errorf("tags is required")
	}
	if len(req.Tags) > 128 {
		return fmt.Errorf("tags must not exceed 128 characters")
	}

	if req.SortOrder != 0 && (req.SortOrder < 0 || req.SortOrder > 100) {
		return fmt.Errorf("sort_order must be between 0 and 100")
	}

	// is_active 默认为 true, is_recommended 默认为 false

	if req.ThirdPartyModelID == "" {
		return fmt.Errorf("third_party_model_id is required")
	}
	if len(req.ThirdPartyModelID) > 128 {
		return fmt.Errorf("third_party_model_id must not exceed 128 characters")
	}

	if req.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}
	if len(req.BaseURL) > 512 {
		return fmt.Errorf("base_url must not exceed 512 characters")
	}

	// 校验尺寸参数
	if req.DefaultWidth != 0 && (req.DefaultWidth < 512 || req.DefaultWidth > 2048) {
		return fmt.Errorf("default_width must be between 512 and 2048")
	}
	if req.DefaultHeight != 0 && (req.DefaultHeight < 512 || req.DefaultHeight > 2048) {
		return fmt.Errorf("default_height must be between 512 and 2048")
	}
	if req.MaxWidth != 0 && (req.MaxWidth < 512 || req.MaxWidth > 2048) {
		return fmt.Errorf("max_width must be between 512 and 2048")
	}
	if req.MaxHeight != 0 && (req.MaxHeight < 512 || req.MaxHeight > 2048) {
		return fmt.Errorf("max_height must be between 512 and 2048")
	}

	// 校验步数参数
	if req.MinSteps != 0 && (req.MinSteps < 1 || req.MinSteps > 100) {
		return fmt.Errorf("min_steps must be between 1 and 100")
	}
	if req.MaxSteps != 0 && (req.MaxSteps < 1 || req.MaxSteps > 100) {
		return fmt.Errorf("max_steps must be between 1 and 100")
	}
	if req.MinSteps != 0 && req.MaxSteps != 0 && req.MinSteps > req.MaxSteps {
		return fmt.Errorf("min_steps must not be greater than max_steps")
	}

	// 校验逻辑关系
	if req.DefaultWidth != 0 && req.MaxWidth != 0 && req.DefaultWidth > req.MaxWidth {
		return fmt.Errorf("default_width must not be greater than max_width")
	}
	if req.DefaultHeight != 0 && req.MaxHeight != 0 && req.DefaultHeight > req.MaxHeight {
		return fmt.Errorf("default_height must not be greater than max_height")
	}

	return nil
}
