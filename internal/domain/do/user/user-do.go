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
