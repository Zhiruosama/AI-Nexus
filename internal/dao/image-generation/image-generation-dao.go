// Package imagegeneration 图像生成dao
package imagegeneration

import (
	"strings"

	image_generation_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
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
	sql := `SELECT COUNT(*) FROM image_generation_models WHERE model_id = ?`
	result := db.GlobalDB.Raw(sql, modelID).Scan(&count)

	if result.Error != nil {
		logger.Error(ctx, "GetModelByID error: %s", result.Error.Error())
		return false, result.Error
	}
	return count > 0, nil
}

// CreateModel 创建新模型
func (d *DAO) CreateModel(ctx *gin.Context, model *image_generation_do.TableImageGenerationModelsDO) error {
	sql := `INSERT INTO image_generation_models (model_id, model_name, model_type, provider, description, tags, sort_order, is_active, is_recommended, third_party_model_id, base_url, default_width, default_height, max_width, max_height, min_steps, max_steps) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result := db.GlobalDB.Exec(sql, model.ModelID, model.ModelName, model.ModelType, model.Provider, model.Description, model.Tags, model.SortOrder, model.IsActive, model.IsRecommended, model.ThirdPartyModelID, model.BaseURL, model.DefaultWidth, model.DefaultHeight, model.MaxWidth, model.MaxHeight, model.MinSteps, model.MaxSteps)

	if result.Error != nil {
		logger.Error(ctx, "CreateModel error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// DeleteModel 删除模型
func (d *DAO) DeleteModel(ctx *gin.Context, modelID string) error {
	sql := "DELETE FROM image_generation_models WHERE model_id = ?"
	result := db.GlobalDB.Exec(sql, modelID)
	if result.Error != nil {
		logger.Error(ctx, "DeleteModel error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// UpdateModel 更新模型数据
func (d *DAO) UpdateModel(ctx *gin.Context, modelID string, updates map[string]interface{}) error {
	set := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	for k, v := range updates {
		set = append(set, k+" = ?")
		args = append(args, v)
	}
	sql := "UPDATE image_generation_models SET " + strings.Join(set, ", ") + " WHERE model_id = ?"
	args = append(args, modelID)
	result := db.GlobalDB.Exec(sql, args...)
	if result.Error != nil {
		logger.Error(ctx, "UpdateModel error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// GetModelInfo 获取模型数据
func (d *DAO) GetModelInfo(ctx *gin.Context, modelID string) (*image_generation_do.TableImageGenerationModelsDO, error) {
	var model *image_generation_do.TableImageGenerationModelsDO
	sql := `SELECT * FROM image_generation_models WHERE model_id = ?`
	result := db.GlobalDB.Raw(sql, modelID).Scan(&model)
	if result.Error != nil {
		logger.Error(ctx, "GetModelByID error: %s", result.Error.Error())
		return nil, result.Error
	}
	return model, nil
}

// QueryModelIDs 根据具体信息查询模型ID
func (d *DAO) QueryModelIDs(ctx *gin.Context, filters map[string]interface{}, q string) ([]string, error) {
	base := `SELECT model_id FROM image_generation_models WHERE 1=1`
	where := make([]string, 0)
	args := make([]interface{}, 0)
	if v, ok := filters["model_name"].(string); ok && v != "" {
		where = append(where, "model_name LIKE ?")
		args = append(args, "%"+v+"%")
	}
	if v, ok := filters["model_type"].(string); ok && v != "" {
		where = append(where, "model_type = ?")
		args = append(args, v)
	}
	if v, ok := filters["provider"].(string); ok && v != "" {
		where = append(where, "provider = ?")
		args = append(args, v)
	}
	if v, ok := filters["is_active"].(bool); ok {
		where = append(where, "is_active = ?")
		args = append(args, v)
	}
	if v, ok := filters["is_recommended"].(bool); ok {
		where = append(where, "is_recommended = ?")
		args = append(args, v)
	}
	if q != "" {
		where = append(where, "(model_name LIKE ? OR description LIKE ? OR tags LIKE ?)")
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	sql := base
	if len(where) > 0 {
		sql += " AND " + strings.Join(where, " AND ")
	}
	var modelIDs []string
	result := db.GlobalDB.Raw(sql, args...).Scan(&modelIDs)
	if result.Error != nil {
		logger.Error(ctx, "QueryModelIDs error: %s", result.Error.Error())
		return nil, result.Error
	}
	return modelIDs, nil
}
