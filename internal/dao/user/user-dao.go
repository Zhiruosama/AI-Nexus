// Package user 用户dao
package user

import (
	"time"

	user_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/user"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DAO 作为 user 模块的 dao 结构体
type DAO struct {
}

// SendEmailCode 发送邮箱方法
func (d *DAO) SendEmailCode(ctx *gin.Context, do *user_do.TableUserVerificationCodesDO) error {
	sql := `INSERT INTO user_verification_codes (email, code, purpose) VALUES (?, ?, ?)`
	result := db.GlobalDB.Exec(sql, do.Email, do.Code, do.Purpose)

	if result.Error != nil {
		logger.Error(ctx, "SendEmailCodeDAO insert error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}

// CheckUserExists 检查用户是否存在
func (d *DAO) CheckUserExists(ctx *gin.Context, email string) (bool, error) {
	var count int64
	sql := `SELECT COUNT(*) FROM users WHERE email = ?`
	result := db.GlobalDB.Raw(sql, email).Scan(&count)

	if result.Error != nil {
		logger.Error(ctx, "CheckUserExits query error: %s", result.Error.Error())
		return false, result.Error
	}

	return count > 0, nil
}

// CreateUser 创建用户
func (d *DAO) CreateUser(ctx *gin.Context, userDO *user_do.TableUserDO) error {
	sql := `INSERT INTO users (uuid, avatar, nickname, email, password_hash) VALUES (?, ?, ?, ?, ?)`
	result := db.GlobalDB.Exec(sql,
		userDO.UUID,
		userDO.Avatar,
		userDO.Nickname,
		userDO.Email,
		userDO.PasswordHash,
	)

	if result.Error != nil {
		logger.Error(ctx, "CreateUser insert error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// GetPasswordByNickname 根据用户名获取用户密码
func (d *DAO) GetPasswordByNickname(ctx *gin.Context, nickname string) (uuid string, password string, err error) {
	sql := `SELECT uuid, password_hash FROM users WHERE nickname = ?`
	var creds userCredentials
	result := db.GlobalDB.Raw(sql, nickname).Scan(&creds)

	if result.Error != nil {
		logger.Error(ctx, "GetPasswordByNickname query error: %s", result.Error.Error())
		return "", "", result.Error
	}

	return creds.UUID, creds.PasswordHash, nil
}

// GetPasswordByEmail 根据用户邮箱获取用户密码
func (d *DAO) GetPasswordByEmail(ctx *gin.Context, email string) (uuid string, password string, err error) {
	sql := `SELECT uuid,password_hash FROM users WHERE email = ?`
	var creds userCredentials
	result := db.GlobalDB.Raw(sql, email).Scan(&creds)

	if result.Error != nil {
		logger.Error(ctx, "GetPasswordByEmail query error: %s", result.Error.Error())
		return "", "", result.Error
	}

	return creds.UUID, creds.PasswordHash, nil
}

// GetUserByID 根据UUID获取用户
func (d *DAO) GetUserByID(ctx *gin.Context, userid string) (userDO *user_do.TableUserDO, err error) {
	userDO = &user_do.TableUserDO{}
	sql := `SELECT uuid,nickname,email,avatar from users WHERE uuid = ?`

	result := db.GlobalDB.Raw(sql, userid).Scan(userDO)
	if result.Error != nil {
		logger.Error(ctx, "GetUserByID query error: %s", result.Error.Error())
		return nil, result.Error
	}
	return userDO, nil
}

// UpdateLoginTime 更新登录时间戳
func (d *DAO) UpdateLoginTime(ctx *gin.Context, userid string) error {
	sql := `UPDATE users SET last_login=? WHERE uuid=?`
	result := db.GlobalDB.Exec(sql, time.Now(), userid)

	if result.Error != nil {
		logger.Error(ctx, "UpdateLoginTime update error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// GetAllUsers 查询所有用户信息
func (d *DAO) GetAllUsers(ctx *gin.Context) ([]*user_do.TableUserDO, int, error) {
	var users = make([]*user_do.TableUserDO, 0)
	sql := `SELECT id, uuid, nickname, avatar, email, last_login, created_at, updated_at FROM users`

	result := db.GlobalDB.Raw(sql).Scan(&users)
	if result.Error != nil {
		logger.Error(ctx, "GetAllUsers query error: %s", result.Error.Error())
		return nil, 0, result.Error
	}

	return users, len(users), nil
}

// UpdateUserInfo 更新用户信息
func (d *DAO) UpdateUserInfo(ctx *gin.Context, userid string, nickname string, avatarpath string) error {
	var sql string
	var result *gorm.DB

	if avatarpath == "" {
		sql = `UPDATE users SET nickname=? WHERE uuid=?`
		result = db.GlobalDB.Exec(sql, nickname, userid)
	} else if nickname == "" {
		sql = `UPDATE users SET avatar=? WHERE uuid=?`
		result = db.GlobalDB.Exec(sql, avatarpath, userid)
	} else {
		sql = `UPDATE users SET nickname=?,avatar=? WHERE uuid=?`
		result = db.GlobalDB.Exec(sql, nickname, avatarpath, userid)
	}

	if result.Error != nil {
		logger.Error(ctx, "UpdateUserInfo update error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// GetUserAvatar 根据UUID获取用户头像路径
func (d *DAO) GetUserAvatar(ctx *gin.Context, uuid string) (string, error) {
	var avatar string
	sql := `SELECT avatar FROM users WHERE uuid = ?`

	result := db.GlobalDB.Raw(sql, uuid).Scan(&avatar)
	if result.Error != nil {
		logger.Error(ctx, "GetUserAvatar query error: %s", result.Error.Error())
		return "", result.Error
	}

	return avatar, nil
}

// DestroyUser 注销用户
func (d *DAO) DestroyUser(ctx *gin.Context, uuid string) error {
	sql := `DELETE uvc FROM user_verification_codes uvc
  INNER JOIN users u ON uvc.email = u.email
  WHERE u.uuid = ?`

	result := db.GlobalDB.Exec(sql, uuid)
	if result.Error != nil {
		logger.Error(ctx, "destroy uvc error: %s", result.Error.Error())
		return result.Error
	}

	sql = `DELETE FROM users WHERE uuid = ?`

	result = db.GlobalDB.Exec(sql, uuid)
	if result.Error != nil {
		logger.Error(ctx, "destroy user error: %s", result.Error.Error())
		return result.Error
	}

	return nil
}

// userCredentials 接收查询结果
type userCredentials struct {
	UUID         string `gorm:"column:uuid"`
	PasswordHash string `gorm:"column:password_hash"`
}
