// 自定义错误码
package middleware

const (
	// ImageAPILimit 图像API限制错误码
	ImageAPILimit int = iota + 50000
	// ImageRPCError 图像RPC错误码
	ImageRPCError
)

const (
	// VideoAPILimit 视频API限制错误码
	VideoAPILimit int = iota + 60000
	// VideoRPCError 视频RPC错误码
	VideoRPCError
)
