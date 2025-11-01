// Package user 用户服务
package user

import (
	"fmt"

	user_dao "github.com/Zhiruosama/ai_nexus/internal/dao/user"
	user_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/user"
	user_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/user"
	user_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/user"
	user_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/user"
	"github.com/Zhiruosama/ai_nexus/internal/grpc"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
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

	// 注册成功清除redis
	_, err = rdbClient.Del(rCtx, codePrefix+dto.Email).Result()
	if err != nil {
		logger.Error(ctx, "Delete verify code error: %s", err.Error())
		return err
	}

	return nil
}

// LoginWithNicknamePassword 用户名密码登录
func (s *Service) LoginWithNicknamePassword(ctx *gin.Context, query *user_query.LoginQuery, vo *user_vo.LoginVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx
	uuid, password, err := s.UserDao.GetPasswordByNickname(ctx, query.Nickname)
	if err != nil {
		return err
	}

	_, err = rdbClient.Get(rCtx, uuid).Result()
	if err == redis.Nil {
		ok, errs := pkg.VerifyPassword(query.PassWord, password)
		if errs != nil {
			return fmt.Errorf("VerifyPassword error")
		} else if !ok {
			return fmt.Errorf("password error")
		}

		vo.JWTToken, errs = middleware.GenerateToken(uuid)
		if errs != nil {
			logger.Error(ctx, "Generate token error: %s", err.Error())
			return err
		}

		rdbClient.Set(rCtx, uuid, vo.JWTToken, 0)

		return nil
	}
	if err != nil {
		logger.Error(ctx, "redis get error.: %s", err.Error())
		return err
	}
	return fmt.Errorf("this account is logged in")
}

// LoginWithEmailPassword 邮箱密码登录
func (s *Service) LoginWithEmailPassword(ctx *gin.Context, query *user_query.LoginQuery, vo *user_vo.LoginVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx
	uuid, password, err := s.UserDao.GetPasswordByEmail(ctx, query.Email)
	if err != nil {
		return err
	}

	_, err = rdbClient.Get(rCtx, uuid).Result()
	if err == redis.Nil {
		ok, errs := pkg.VerifyPassword(query.PassWord, password)
		if errs != nil {
			return fmt.Errorf("VerifyPassword error")
		} else if !ok {
			return fmt.Errorf("password error")
		}

		vo.JWTToken, errs = middleware.GenerateToken(uuid)
		if errs != nil {
			logger.Error(ctx, "Generate token error: %s", err.Error())
			return err
		}

		rdbClient.Set(rCtx, uuid, vo.JWTToken, 0)

		return nil
	}

	if err != nil {
		logger.Error(ctx, "redis get error.: %s", err.Error())
		return err
	}
	return fmt.Errorf("this account is logged in")
}

// LoginWithEmailVerifyCode 邮箱验证码登录
func (s *Service) LoginWithEmailVerifyCode(ctx *gin.Context, query *user_query.LoginQuery, vo *user_vo.LoginVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx
	code, err := rdbClient.Get(rCtx, codePrefix+query.Email).Result()
	if err != nil {
		logger.Error(ctx, "Get verify code error: %s", err.Error())
		return err
	}
	uuid, _, err := s.UserDao.GetPasswordByEmail(ctx, query.Email)
	if err != nil {
		logger.Error(ctx, "Get uuid error: %s", err.Error())
		return err
	}

	if code != query.VerifyCode {
		return fmt.Errorf("email verification code error")
	}

	vo.JWTToken, err = middleware.GenerateToken(uuid)
	if err != nil {
		logger.Error(ctx, "Generate token error: %s", err.Error())
		return err
	}

	rdbClient.Set(rCtx, uuid, vo.JWTToken, 0)

	return nil
}

// GetUserInfo 获取用户信息
func (s *Service) GetUserInfo(ctx *gin.Context, user_id string) (*user_vo.UserInfoVO, error) {
	userDO, err := s.UserDao.GetUserByID(ctx, user_id)

	if err != nil {
		logger.Error(ctx, "Get user error: %s", err.Error())
		return nil, err
	}

	return &user_vo.UserInfoVO{
		UUID:     userDO.UUID,
		Nickname: userDO.Nickname,
		Email:    userDO.Email,
		Avatar:   userDO.Avatar,
	}, nil
}
