// Package middleware 自定义错误码
package middleware

const (
	// ParamEmpty 请求参数为空
	ParamEmpty int = iota + 3000
	// PasswordMismatch 密码不一致
	PasswordMismatch
	// PasswordInvalid 密码正则匹配错误
	PasswordInvalid
	// RPCSendCodeFailed 验证码发送失败
	RPCSendCodeFailed
	// EmailInvalid 邮箱正则错误
	EmailInvalid
)

const (
	// ImageAPILimit 图像API限制错误码
	ImageAPILimit int = iota + 5000
	// ImageRPCError 图像RPC错误码
	ImageRPCError
)

const (
	// VideoAPILimit 视频API限制错误码
	VideoAPILimit int = iota + 6000
	// VideoRPCError 视频RPC错误码
	VideoRPCError
)
