// Package user 对应usre的VO结构集合
package user

// RegisterResponse 用户注册响应
type RegisterResponse struct {
	UserID   string `json:"user_id"`
	NickName string `json:"nickname,omitempty" form:"nickname,omitempty"`
	Email    string `json:"email" form:"email"`
	Token    string `json:"token" form:"token"`
}

// LoginResponse 用户登录响应
type LoginResponse struct {
	UserID   string `json:"user_id"`
	NickName string `json:"nickname,omitempty" form:"nickname,omitempty"`
	Email    string `json:"email" form:"email"`
	Token    string `json:"token" form:"token"`
}
