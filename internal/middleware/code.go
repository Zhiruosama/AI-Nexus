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
	// PurposeInvalid 验证码目的错误
	PurposeInvalid
	// UserAlreadyExist 用户已存在
	UserAlreadyExist
	// UserNotExists 用户不存在
	UserNotExists
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
	// GetUserInfoFailed 获取用户信息失败
	GetUserInfoFailed
	// GetAllUserInfoFailed 获取所有用户信息失败
	GetAllUserInfoFailed
	// UpdateUserInfoFailed 更新用户信息失败
	UpdateUserInfoFailed
	// DestoryUserFailed 删除用户失败
	DestoryUserFailed
	// LoginPurposeError 登录目的错误
	LoginPurposeError
	// ResetPasswordPurposeError 重置密码目的错误
	ResetPasswordPurposeError
	// ParseTokenFailed 解析 token 失败
	ParseTokenFailed
	// RedisError Redis 错误
	RedisError
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
