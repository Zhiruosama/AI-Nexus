// Package imagegeneration 图片生成controller
package imagegeneration

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"

	image_generation_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/image-generation"
	image_generation_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/image-generation"
	image_generation_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/image-generation"
	"github.com/Zhiruosama/ai_nexus/internal/pkg"
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
		"message": "batch create model success",
	})
}

// DeleteModel 删除模型
func (c *Controller) DeleteModel(ctx *gin.Context) {
	idsParam := ctx.DefaultQuery("ids", "")

	if idsParam == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "ids is required",
		})
		return
	}

	ids := strings.SplitSeq(idsParam, ",")

	for modelID := range ids {
		if err := c.ImageGenerationService.DeleteModel(ctx, modelID); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": err.Error(),
			})
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "delete all models success",
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
	vo := image_generation_vo.GetModelInfoVO{}

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
		vo.Code = http.StatusBadRequest
		vo.Message = err.Error()
		ctx.JSON(http.StatusBadRequest, vo)
		return
	}

	vo.Code = http.StatusOK
	vo.Message = "get modelinfo success"
	vo.Model = model
	ctx.JSON(http.StatusOK, vo)
}

// QueryModels 根据具体信息查询模型列表
func (c *Controller) QueryModels(ctx *gin.Context) {
	var query image_generation_query.ModelsQuery
	vo := image_generation_vo.QueryModelsVO{}

	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "invalid query parameters: " + err.Error(),
		})
		return
	}

	if query.PageSize == 0 {
		query.PageSize = 20
	}
	if query.PageIndex < 0 {
		query.PageIndex = 0
	}

	vo.Data.PageIndex = query.PageIndex
	vo.Data.PageSize = query.PageSize

	// 调用 Service 层
	models, total, err := c.ImageGenerationService.QueryModels(ctx, &query)
	if err != nil {
		vo.Code = http.StatusInternalServerError
		vo.Message = "failed to query models"
		ctx.JSON(http.StatusInternalServerError, vo)
		return
	}

	vo.Code = http.StatusOK
	vo.Message = "query models success"
	vo.Data.Total = int(total)
	vo.Data.Models = models
	ctx.JSON(http.StatusOK, vo)
}

// Text2Img 文生图
func (c *Controller) Text2Img(ctx *gin.Context) {
	dto := &image_generation_dto.Text2ImgDTO{}

	if err := ctx.ShouldBindJSON(dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if dto.Prompt == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "prompt is required",
		})
		return
	}
	if err := pkg.ValidatePrompt(dto.Prompt); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	if dto.NegativePrompt == "" {
		dto.NegativePrompt = "lowres, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, worst quality, low quality, normal quality, jpeg artifacts, signature, watermark, username, blurry"
	}

	if dto.ModelID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "model_id is required",
		})
		return
	}

	if dto.Width == 0 {
		dto.Width = 760
	}

	if dto.Height == 0 {
		dto.Height = 1280
	}

	if dto.NumInferenceSteps == 0 {
		dto.NumInferenceSteps = 20
	}

	if dto.GuidanceScale == 0 {
		dto.GuidanceScale = 7.5
	}

	if dto.Seed == 0 {
		dto.Seed = rand.Int64N(2147483649) - 1
	}

	taskID, err := c.ImageGenerationService.Text2Img(ctx, dto)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "text2img create task success",
		"data": gin.H{
			"task_id": taskID,
			"status":  "queued",
		},
	})
}

// Img2Img 图生图
func (c *Controller) Img2Img(ctx *gin.Context) {
	dto := image_generation_dto.Img2ImgDTO{}

	if err := ctx.ShouldBind(&dto); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "The input data does not meet the requirements.",
		})
	}

	if dto.InputImage == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "input_image is required",
		})
		return
	}

	if dto.Prompt == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "prompt is required",
		})
		return
	}
	if err := pkg.ValidatePrompt(dto.Prompt); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": err.Error(),
		})
		return
	}

	if dto.NegativePrompt == "" {
		dto.NegativePrompt = "lowres, bad anatomy, bad hands, text, error, missing fingers, extra digit, fewer digits, cropped, worst quality, low quality, normal quality, jpeg artifacts, signature, watermark, username, blurry"
	}

	if dto.Strength == 0 {
		dto.Strength = 0.8
	}

	if dto.Width == 0 {
		dto.Width = 760
	}

	if dto.Height == 0 {
		dto.Height = 1280
	}

	if dto.NumInferenceSteps == 0 {
		dto.NumInferenceSteps = 20
	}

	if dto.GuidanceScale == 0 {
		dto.GuidanceScale = 7.5
	}

	if dto.Seed == 0 {
		dto.Seed = rand.Int64N(2147483649) - 1
	}

	taskID, err := c.ImageGenerationService.Img2Img(ctx, &dto)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "img2img create task success",
		"data": gin.H{
			"task_id": taskID,
			"status":  "queued",
		},
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
