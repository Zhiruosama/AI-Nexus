// Package user 对应user表中的DO结构
package user

// TableUserDO 对应 user 表中的 DO 结构
type TableUserDO struct {
	ID           int64  `gorm:"column:id"`
	UUID         string `gorm:"column:uuid"`
	Nickname     string `gorm:"column:nickname"`
	Avatar       string `gorm:"column:avatar"`
	Email        string `gorm:"column:email"`
	PasswordHash string `gorm:"column:password_hash"`
	LastLogin    string `gorm:"column:last_login"`
	CreatedAt    string `gorm:"column:created_at"`
	UpdatedAt    string `gorm:"column:updated_at"`
}

// TableUserVerificationCodesDO 对应user_verification_codes表中的DO结构
type TableUserVerificationCodesDO struct {
	ID       int64  `gorm:"column:id"`
	Email    string `gorm:"column:email"`
	Code     string `gorm:"column:code"`
	Purpose  int8   `gorm:"column:purpose"`
	CreateAt string `gorm:"column:created_at"`
}
