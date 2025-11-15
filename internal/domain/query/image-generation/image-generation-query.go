// Package imagegeneration 此模块下的query请求
package imagegeneration

// ModelsQuery 模型查询结构体
type ModelsQuery struct {
	// 分页参数
	PageIndex int `form:"pageIndex"`
	PageSize  int `form:"pageSize"`

	// 基本字段
	ModelType         *string  `form:"model_type"`
	Provider          *string  `form:"provider"`
	TotalUsage        *uint64  `form:"total_usage"`
	SuccessRate       *float64 `form:"success_rate"`
	IsActive          *bool    `form:"is_active"`
	IsRecommended     *bool    `form:"is_recommended"`
	ThirdPartyModelID *string  `form:"third_party_model_id"`
	Width             *int     `form:"width"`
	Height            *int     `form:"height"`
	Steps             *int     `form:"steps"`
	CreateAt          *string  `form:"create_at"`

	// 全文搜索关键字，在 model_name/description/tags 上做 OR 模糊
	Q *string `form:"q"`
}
