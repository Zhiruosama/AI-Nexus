// Package user 对应user表中的DO结构
package user

// TableUserDO 对应 user 表中的 DO 结构
type TableUserDO struct {
	ID           int64
	UUID         string
	Nickname     string
	Avatar       string
	Email        string
	PasswordHash string
	LastLogin    string
	UpdatedAt    string
}

// TableUserVerificationCodesDO 对应user_verification_codes表中的DO结构
type TableUserVerificationCodesDO struct {
	ID       int64
	Email    string
	Code     string
	Purpose  int8
	CreateAt string
}
