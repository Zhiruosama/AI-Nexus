// Package imagegeneration 图像生成dao
package imagegeneration

import (
	imagegeneration_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// DAO 作为 imagegeneration 模块的 dao 结构体
type DAO struct {
}

// CheckModelExists 检查模型是否存在
func (d *DAO) CheckModelExists(ctx *gin.Context, modelID string) (bool, error) {
	var count int64
	sql := `SELECT COUNT(*) FROM image_generation_models WHERE id = ?`
	result := db.GlobalDB.Raw(sql, modelID).Scan(&count)

	if result.Error != nil {
		logger.Error(ctx, "GetModelByID error: %s", result.Error.Error())
		return false, result.Error
	}
	return count > 0, nil
}

// CreateModel 创建新模型
func (d *DAO) CreateModel(ctx *gin.Context, model *imagegeneration_do.TableImageGenerationModelsDO) error {
	sql := `INSERT INTO image_generation_models (model_id, model_name, model_type, provider, description, tags, sort_order, third_party_model_id, base_url, default_width, default_height, max_width, max_height, min_steps, max_steps) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result := db.GlobalDB.Exec(sql, model.ModelID, model.ModelName, model.ModelType, model.Provider, model.Description, model.Tags, model.SortOrder, model.ThirdPartyModelID, model.BaseUrl, model.DefaultWidth, model.DefaultHeight, model.MaxWidth, model.MaxHeight, model.MinSteps, model.MaxSteps)

	if result.Error != nil {
		logger.Error(ctx, "CreateModel error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}
