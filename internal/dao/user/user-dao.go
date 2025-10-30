// Package user 用户dao
package user

import (
	"fmt"

	user_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/user"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

// DAO 作为 user 模块的 dao 结构体
type DAO struct {
}

// SendEmailCode 发送邮箱方法
func (d DAO) SendEmailCode(ctx *gin.Context, do *user_do.TableUserVerificationCodesDO) error {
	sql := fmt.Sprintf("INSERT INTO user_verification_codes (`email`, `code`, `purpose`) VALUES ('%s', '%s', %d)", do.Email, do.Code, do.Purpose)
	result := db.GlobalDB.Exec(sql)

	if result.Error != nil {
		logger.Error(ctx, "SendEmailCodeDAO insert error: %s", result.Error.Error())
		return result.Error
	}
	return nil
}
