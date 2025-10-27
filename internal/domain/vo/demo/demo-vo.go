// Package demo 对应 demo 的 VO 结构集合
package demo

// GetMessageByIDVO 对应函数的 VO 视图
type GetMessageByIDVO struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
}
