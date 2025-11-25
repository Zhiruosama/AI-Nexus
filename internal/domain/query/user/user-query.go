// Package user 此模块下的query请求
package user

// GetAllUsersQuery 获取用户信息
type GetAllUsersQuery struct {
	PageIndex int
	PageSize  int
}
