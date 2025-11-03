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

// ListUserInfoVO 所有用户信息
type ListUserInfoVO struct {
	Code    int32         `json:"code"`
	Message string        `json:"message"`
	Count   int           `json:"count"`
	Users   []TableUserVO `json:"users"`
}

// TableUserVO 单个用户的全量数据
type TableUserVO struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Email     string `json:"email"`
	LastLogin string `json:"last_login"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"update_at"`
}
