// Package middleware Code 自定义错误码
package middleware

const (
	// 照片生成API限制错误码
	IMAGE_APILIMIt int = iota + 50000
	// 照片生成RPC调用错误 错误码
	IMAGE_RPCERROR
)

const (
	// 视频生成API限制错误码
	VIDEO_APILIMIT int = iota + 60000
	// 视频生成RPC调用错误 错误码
	VIDEO_RPCERROR
)
