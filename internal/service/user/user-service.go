// Package user 用户服务
package user

import (
	"fmt"

	user_dao "github.com/Zhiruosama/ai_nexus/internal/dao/user"
	user_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/user"
	user_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/user"
	"github.com/Zhiruosama/ai_nexus/internal/grpc"
	"github.com/Zhiruosama/ai_nexus/internal/pkg"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Service 对应 user 模块的 Service 结构
type Service struct {
	UserDao *user_dao.DAO
}

// NewService 对应 user 模块的 Service 工厂方法
func NewService() *Service {
	return &Service{
		UserDao: &user_dao.DAO{},
	}
}

const (
	codePrefix = "code_"
)

// SendEmailCode 发送邮箱服务
func (s *Service) SendEmailCode(ctx *gin.Context, dto *user_dto.SendEmailCode) error {
	do := &user_do.TableUserVerificationCodesDO{}

	do.Email = dto.Email
	_, _, code, err := grpc.GetVerificationCode(dto.Email)
	if err != nil {
		logger.Error(ctx, "RPC send code error: %s", err.Error())
		return err
	}
	do.Code = code
	do.Purpose = 1

	err = s.UserDao.SendEmailCode(ctx, do)
	if err != nil {
		return err
	}
	return nil
}

// Register 注册服务
func (s *Service) Register(ctx *gin.Context, dto *user_dto.RegisterRequest) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	code, err := rdbClient.Get(rCtx, codePrefix+dto.Email).Result()
	if err == redis.Nil {
		logger.Error(ctx, "Verify email code error: %s", err.Error())
		return fmt.Errorf("the verification code has expired")
	}

	if code != dto.VerifyCode {
		return fmt.Errorf("verify code error")
	}

	// 检查用户是否存在
	exists, err := s.UserDao.CheckUserExists(ctx, dto.Email)
	if err != nil {
		logger.Error(ctx, "check user exists error: %s", err.Error())
		return err
	}
	if exists {
		return fmt.Errorf("user exists")
	}

	passwordHash, err := pkg.HashPassword(dto.PassWord)
	if err != nil {
		logger.Error(ctx, "Hash password error: %s", err.Error())
		return err
	}

	// 角色DO
	userDO := &user_do.TableUserDO{
		UUID:         uuid.New().String(),
		Avatar:       "/static/avatar/default.png",
		Nickname:     dto.NickName,
		Email:        dto.Email,
		PasswordHash: passwordHash,
	}

	err = s.UserDao.CreateUser(ctx, userDO)
	if err != nil {
		logger.Error(ctx, "Create user error: %s", err.Error())
		return err
	}

	return nil
}
