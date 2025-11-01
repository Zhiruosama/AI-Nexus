// Package user 对应usre的VO结构集合
package user

// LoginVO 用户登录信息VO
type LoginVO struct {
	Code     int32  `json:"code"`
	Message  string `json:"message"`
	JWTToken string `json:"token"`
}

// InfoVO 用户信息VO
type InfoVO struct {
	UUID     string `json:"uuid"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}
