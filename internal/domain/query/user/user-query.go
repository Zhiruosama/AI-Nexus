// Package user 此模块下的query请求
package user

// SendEmailCode 请求发送验证码
type SendEmailCode struct {
	NickName       string `json:"nickname,omitempty"`
	Email          string `json:"email"`
	PassWord       string `json:"password"`
	RepeatPassWord string `json:"repeatpassword"`
}
