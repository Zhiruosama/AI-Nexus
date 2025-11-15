// Package imagegeneration 图片生成vo
package imagegeneration

import (
	image_generation_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/image-generation"
)

// GetModelInfoVO 获取模型数据
type GetModelInfoVO struct {
	Code    int                                               `json:"code"`
	Message string                                            `json:"message"`
	Model   *image_generation_do.TableImageGenerationModelsDO `json:"model"`
}

// QueryModelsVO 查询模型
type QueryModelsVO struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    dataForQueryModels `json:"data"`
}

type dataForQueryModels struct {
	PageIndex int                                                 `json:"pageIndex"`
	PageSize  int                                                 `json:"pageSize"`
	Total     int                                                 `json:"total"`
	Models    []*image_generation_do.TableImageGenerationModelsDO `json:"models"`
}
