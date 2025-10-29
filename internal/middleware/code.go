// Package middleware Code 自定义错误码
package middleware

const (
	// ImageAPILimit 照片生成 API 限制错误码。
	ImageAPILimit int = iota + 50000
	// ImageRPCError 照片生成 RPC 调用错误码。
	ImageRPCError
)

const (
	// VideoAPILimit 视频 API 限制错误码。
	VideoAPILimit int = iota + 60000
	// VideoRPCError 视频生成RPC调用错误 错误码
	VideoRPCError
)
