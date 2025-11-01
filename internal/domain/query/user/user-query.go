// Package user 此模块下的query请求
package user

type LoginQuery struct {
	Email      string `form:"email,omitempty"`
	Nickname   string `form:"nickname,omitempty"`
	PassWord   string `form:"password,omitempty"`
	VerifyCode string `form:"verify_code,omitempty"`
}
