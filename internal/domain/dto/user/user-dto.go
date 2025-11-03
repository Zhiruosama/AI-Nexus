// Package user 此模块下的dto请求
package user

import "mime/multipart"

// SendEmailCode 请求发送验证码
type SendEmailCode struct {
	Purpose int    `json:"purpose" form:"purpose"`
	Email   string `json:"email" form:"email"`
}

// RegisterRequest 用户注册请求
type RegisterRequest struct {
	NickName       string `json:"nickname,omitempty" form:"nickname,omitempty"`
	Email          string `json:"email" form:"email"`
	PassWord       string `json:"password" form:"password"`
	RepeatPassWord string `json:"repeat_password" form:"repeat_password"`
	VerifyCode     string `json:"verify_code" form:"verify_code"`
	Purpose        string `json:"purpose" form:"purpose"`
}

// UpdateInfoRequest 用户更新数据请求
type UpdateInfoRequest struct {
	NickName string                `json:"nickname,omitempty" form:"nickname,omitempty"`
	Avatar   *multipart.FileHeader `json:"avatar,omitempty" form:"avatar,omitempty"`
	Sha256   string                `json:"sha256,omitempty" form:"sha256,omitempty"`
}
