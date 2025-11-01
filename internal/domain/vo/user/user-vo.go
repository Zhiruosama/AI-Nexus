// Package user 对应usre的VO结构集合
package user

type LoginVO struct {
	Code     int32  `json:"code"`
	Message  string `json:"message"`
	JWTToken string `json:"token"`
}
