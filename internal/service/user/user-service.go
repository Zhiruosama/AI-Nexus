// Package user 用户服务
package user

import (
	user_dao "github.com/Zhiruosama/ai_nexus/internal/dao/user"
	user_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/user"
	user_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/user"
	"github.com/Zhiruosama/ai_nexus/internal/grpc"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/logger"
	"github.com/gin-gonic/gin"
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
