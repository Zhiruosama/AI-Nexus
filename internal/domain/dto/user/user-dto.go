// Package user 此模块下的dto请求
package user

// SendEmailCode 请求发送验证码
type SendEmailCode struct {
	NickName       string `json:"nickname,omitempty" form:"nickname,omitempty"`
	Email          string `json:"email" form:"email"`
	PassWord       string `json:"password" form:"password"`
	RepeatPassWord string `json:"repeatpassword" form:"repeat_password"`
}
