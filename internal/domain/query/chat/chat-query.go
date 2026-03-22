// Package chat 对话模块查询参数
package chat

// ConversationListQuery 对话列表查询
type ConversationListQuery struct {
	PageIndex int `form:"pageIndex"`
	PageSize  int `form:"pageSize"`
}
