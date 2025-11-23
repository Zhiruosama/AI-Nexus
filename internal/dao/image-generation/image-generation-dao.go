// Package imagegeneration 图像生成dao
package imagegeneration

import (
	"log"
	"strings"

	image_generation_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
	image_generation_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/image-generation"
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
func (d *DAO) UpdateModel(ctx *gin.Context, modelID string, updates map[string]any) error {
	set := make([]string, 0, len(updates))
	args := make([]any, 0, len(updates)+1)

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
	var model image_generation_do.TableImageGenerationModelsDO
	sql := `SELECT * FROM image_generation_models WHERE model_id = ?`

	result := db.GlobalDB.Raw(sql, modelID).Scan(&model)
	if result.Error != nil {
		logger.Error(ctx, "GetModelByID error: %s", result.Error.Error())
		return nil, result.Error
	}
	return &model, nil
}

// QueryModels 根据具体信息查询模型列表
func (d *DAO) QueryModels(ctx *gin.Context, query *image_generation_query.ModelsQuery) ([]*image_generation_do.TableImageGenerationModelsDO, int64, error) {
	base := `SELECT * FROM image_generation_models WHERE 1=1`
	countBase := `SELECT COUNT(*) FROM image_generation_models WHERE 1=1`

	whereClause, args := buildQueryCondition(query)

	var total int64
	countSQL := countBase + whereClause
	result := db.GlobalDB.Raw(countSQL, args...).Scan(&total)
	if result.Error != nil {
		logger.Error(ctx, "QueryModels count error: %s", result.Error.Error())
		return nil, 0, result.Error
	}

	offset := query.PageIndex * query.PageSize
	sql := base + whereClause + " ORDER BY sort_order DESC, created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, query.PageSize, offset)

	var models []*image_generation_do.TableImageGenerationModelsDO
	result = db.GlobalDB.Raw(sql, queryArgs...).Scan(&models)
	if result.Error != nil {
		logger.Error(ctx, "QueryModels error: %s", result.Error.Error())
		return nil, 0, result.Error
	}

	return models, total, nil
}

// GetInfoFromModel 获取模型信息
func GetInfoFromModel[T any](_ *DAO, key, modelID string) (T, error) {
	sql := "SELECT " + key + " FROM image_generation_models WHERE model_id = ?"

	var val T
	result := db.GlobalDB.Raw(sql, modelID).Scan(&val)

	if result.Error != nil {
		log.Printf("GetInfoFromModel error: %s\n", result.Error.Error())
		var zero T
		return zero, result.Error
	}

	return val, nil
}

// UpdateModelUsage 更新模型使用量
func (d *DAO) UpdateModelUsage(success bool, modelID string) error {
	var successIncrement int
	if success {
		successIncrement = 1
	}

	sql := `UPDATE image_generation_models
		SET total_usage = total_usage + 1,
		    success_count = success_count + ?,
		    success_rate = (success_count + ?) / (total_usage + 1) * 100
		WHERE model_id = ?`

	result := db.GlobalDB.Exec(sql, successIncrement, successIncrement, modelID)
	if result.Error != nil {
		log.Printf("UpdateModelUsage error: %s\n", result.Error.Error())
		return result.Error
	}

	return nil
}

// CreateText2ImgTask 创建文生图任务
func (d *DAO) CreateText2ImgTask(ctx *gin.Context, do *image_generation_do.TableImageGenerationTaskDO) error {
	sql := `INSERT INTO image_generation_tasks (task_id, user_uuid, task_type, status, prompt, negative_prompt, model_id, width, height, num_inference_steps, guidance_scale, seed) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result := db.GlobalDB.Exec(sql, do.TaskID, do.UserUUID, do.TaskType, do.Status, do.Prompt, do.NegativePrompt, do.ModelID, do.Width, do.Height, do.NumInferenceSteps, do.GuidanceScale, do.Seed)

	if result.Error != nil {
		logger.Error(ctx, "Create text2img task error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// CreateImg2ImgTask 创建图生图任务
func (d *DAO) CreateImg2ImgTask(ctx *gin.Context, do *image_generation_do.TableImageGenerationTaskDO) error {
	sql := `INSERT INTO image_generation_tasks (task_id, user_uuid, task_type, status, prompt, negative_prompt, model_id, width, height, num_inference_steps, guidance_scale, seed, input_image_url, strength) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	result := db.GlobalDB.Exec(sql, do.TaskID, do.UserUUID, do.TaskType, do.Status, do.Prompt, do.NegativePrompt, do.ModelID, do.Width, do.Height, do.NumInferenceSteps, do.GuidanceScale, do.Seed, do.InputImageURL, do.Strength)

	if result.Error != nil {
		logger.Error(ctx, "Create img2img task error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// DeleteTask 删除任务
func (d *DAO) DeleteTask(ctx *gin.Context, taskID string) error {
	sql := "DELETE FROM image_generation_tasks WHERE task_id = ?"
	result := db.GlobalDB.Exec(sql, taskID)

	if result.Error != nil {
		logger.Error(ctx, "DeleteTask error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// UpdateTaskParams 更新任务表的参数
func (d *DAO) UpdateTaskParams(key string, val any, taskID string) error {
	sql := "UPDATE image_generation_tasks SET " + key + " = ? WHERE task_id = ?"
	result := db.GlobalDB.Exec(sql, val, taskID)

	if result.Error != nil {
		log.Fatalf("UpdateTaskParams error: %s\n", result.Error.Error())
		return result.Error
	}

	return nil
}

// GetTaskInfo 获取任务信息
func GetTaskInfo[T any](_ *DAO, key, taskID string) (T, error) {
	sql := "SELECT " + key + " FROM image_generation_tasks WHERE task_id = ?"

	var val T
	result := db.GlobalDB.Raw(sql, taskID).Scan(&val)

	if result.Error != nil {
		log.Printf("GetTaskInfo error: %s\n", result.Error.Error())
		var zero T
		return zero, result.Error
	}

	return val, nil
}

// CheckDeadLetterExists 判断是否存在死信任务
func (d *DAO) CheckDeadLetterExists(taskID string) (bool, error) {
	var count int64
	sql := `SELECT COUNT(*) FROM dead_letter_tasks WHERE task_id = ?`
	result := db.GlobalDB.Raw(sql, taskID).Scan(&count)

	if result.Error != nil {
		log.Printf("GetDeadLetterByID error: %s", result.Error.Error())
		return false, result.Error
	}
	return count > 0, nil
}

// InsertDeadLetterTask 插入死信任务
func (d *DAO) InsertDeadLetterTask(do *image_generation_do.TableDeadLetterTasksDO) error {
	sql := `INSERT INTO dead_letter_tasks (user_id, task_id, task_type, dead_reason, original_status) VALUES (?, ?, ?, ?, ?)`
	result := db.GlobalDB.Exec(sql, do.UserID, do.TaskID, do.TaskType, do.DeadReason, do.OriginalStatus)

	if result.Error != nil {
		log.Printf("CreateModel error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// buildQueryCondition 构建查询条件
func buildQueryCondition(query *image_generation_query.ModelsQuery) (string, []any) {
	where := make([]string, 0)
	args := make([]any, 0)

	if query.ModelType != nil && *query.ModelType != "" {
		where = append(where, "model_type = ?")
		args = append(args, *query.ModelType)
	}

	if query.Provider != nil && *query.Provider != "" {
		where = append(where, "provider = ?")
		args = append(args, *query.Provider)
	}

	if query.TotalUsage != nil {
		where = append(where, "total_usage >= ?")
		args = append(args, *query.TotalUsage)
	}

	if query.SuccessRate != nil {
		where = append(where, "success_rate >= ?")
		args = append(args, *query.SuccessRate)
	}

	if query.IsActive != nil {
		where = append(where, "is_active = ?")
		args = append(args, *query.IsActive)
	}

	if query.IsRecommended != nil {
		where = append(where, "is_recommended = ?")
		args = append(args, *query.IsRecommended)
	}

	if query.ThirdPartyModelID != nil && *query.ThirdPartyModelID != "" {
		where = append(where, "third_party_model_id = ?")
		args = append(args, *query.ThirdPartyModelID)
	}

	if query.Width != nil {
		where = append(where, "default_width >= ? OR max_width >= ?")
		args = append(args, *query.Width, *query.Width)
	}

	if query.Height != nil {
		where = append(where, "default_height >= ? OR max_height >= ?")
		args = append(args, *query.Height, *query.Height)
	}

	if query.Steps != nil {
		where = append(where, "min_steps <= ? AND max_steps >= ?")
		args = append(args, *query.Steps, *query.Steps)
	}

	if query.CreateAt != nil && *query.CreateAt != "" {
		where = append(where, "created_at >= ?")
		args = append(args, *query.CreateAt)
	}

	if query.Q != nil && *query.Q != "" {
		where = append(where, "(model_name LIKE ? OR description LIKE ? OR tags LIKE ?)")
		searchPattern := "%" + *query.Q + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " AND " + strings.Join(where, " AND ")
	}

	return whereClause, args
}
