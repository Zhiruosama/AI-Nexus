// Package user 用户服务
package user

import (
	"fmt"
	"time"

	user_dao "github.com/Zhiruosama/ai_nexus/internal/dao/user"
	user_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/user"
	user_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/user"
	user_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/user"
	"github.com/Zhiruosama/ai_nexus/internal/grpc"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
func (s *Service) Register(ctx *gin.Context, dto *user_dto.RegisterRequest) (*user_vo.RegisterResponse, error) {
	// 验证验证码
	isValid, err := s.UserDao.VerifyEmailCode(ctx, dto.Email, dto.VerifyCode)
	if err != nil {
		logger.Error(ctx, "Verify email code error: %s", err.Error())
		return nil, fmt.Errorf("验证码错误")
	}
	if !isValid {
		return nil, err
	}

	// 检查用户是否存在
	exists, err := s.UserDao.CheckUserExists(ctx, dto.Email)
	if err != nil {
		logger.Error(ctx, "Check user exists error: %s", err.Error())
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("用户已存在")
	}

	passwordHash, err := utils.HashPassword(dto.PassWord)
	if err != nil {
		logger.Error(ctx, "Hash password error: %s", err.Error())
		return nil, err
	}

	// 角色DO
	userDO := &user_do.TableUserDO{
		UUID:         uuid.New().String(),
		Avatar:       "",
		Nickname:     dto.NickName,
		Email:        dto.Email,
		PasswordHash: passwordHash,
		LastLogin:    time.Now().String(),
	}

	err = s.UserDao.CreateUser(ctx, userDO)
	if err != nil {
		logger.Error(ctx, "Create user error: %s", err.Error())
		return nil, err
	}

	token, err := middleware.GenerateToken(userDO.UUID)
	if err != nil {
		logger.Error(ctx, "Generate token error: %s", err.Error())
		return nil, err
	}

	return &user_vo.RegisterResponse{
		Token:    token,
		UserID:   userDO.UUID,
		NickName: userDO.Nickname,
		Email:    userDO.Email,
	}, nil
}
