// Package middleware 自定义错误码
package middleware

const (
	// ParamEmpty 请求参数为空
	ParamEmpty int = iota + 3000
	// PasswordMismatch 密码不一致
	PasswordMismatch
	// PasswordInvalid 密码正则匹配错误
	PasswordInvalid
	// VerifyCodeExist 验证码已存在
	VerifyCodeExist
	// RPCSendCodeFailed 验证码发送失败
	RPCSendCodeFailed
	// EmailInvalid 邮箱正则错误
	EmailInvalid
	// RegisterFailed 注册失败
	RegisterFailed
	// VerifyCodeInvalid 验证码无效
	VerifyCodeInvalid
	// UserAlreadyExist 用户已存在
	UserAlreadyExist
	// LoginParamError 登录参数错误
	LoginParamError
	// UserInformationEmpty 用户信息为空
	UserInformationEmpty
	// PasswordEmpty 密码或者验证码不能为空
	PasswordEmpty
	// LoginFailed 登陆失败
	LoginFailed
	// Loginsuccess 登陆成功
	Loginsuccess
	// LogoutFailed 登出失败
	LogoutFailed
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
