// Package user 用户服务
package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	infoPrefix = "info_"
	allinfoKey = "allInfoForUsers"
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
			logger.Error(ctx, "Generate token error: %s", errs.Error())
			return errs
		}

		_, errs = rdbClient.Set(rCtx, uuid, vo.JWTToken, 0).Result()
		if errs != nil {
			logger.Error(ctx, "Set token to redis error: %s", errs.Error())
			return errs
		}

		errs = s.UserDao.UpdateLoginTime(ctx, uuid)
		if errs != nil {
			return errs
		}

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
			logger.Error(ctx, "Generate token error: %s", errs.Error())
			return errs
		}

		_, errs = rdbClient.Set(rCtx, uuid, vo.JWTToken, 0).Result()
		if errs != nil {
			logger.Error(ctx, "Set token to redis error: %s", errs.Error())
			return errs
		}

		errs = s.UserDao.UpdateLoginTime(ctx, uuid)
		if errs != nil {
			return errs
		}

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

	if code != query.VerifyCode {
		return fmt.Errorf("email verification code error")
	}

	uuid, _, err := s.UserDao.GetPasswordByEmail(ctx, query.Email)
	if err != nil {
		logger.Error(ctx, "Get uuid error: %s", err.Error())
		return err
	}

	vo.JWTToken, err = middleware.GenerateToken(uuid)
	if err != nil {
		logger.Error(ctx, "Generate token error: %s", err.Error())
		return err
	}

	_, err = rdbClient.Set(rCtx, uuid, vo.JWTToken, 0).Result()
	if err != nil {
		logger.Error(ctx, "Set token to redis error: %s", err.Error())
		return err
	}

	err = s.UserDao.UpdateLoginTime(ctx, uuid)
	if err != nil {
		return err
	}

	return nil
}

// GetUserInfo 获取用户信息
func (s *Service) GetUserInfo(ctx *gin.Context, userid string) (*user_vo.InfoVO, error) {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx
	var uvo = &user_vo.InfoVO{}

	info, err := rdbClient.Get(rCtx, infoPrefix+userid).Result()
	if err != nil && err != redis.Nil {
		logger.Error(ctx, "Get userinfo from redis error: %s", err.Error())
		return nil, err
	}

	if err == nil && info != "" {
		if err = json.Unmarshal([]byte(info), uvo); err != nil {
			logger.Error(ctx, "Unmarshal error in get user info: %s", err.Error())
			return nil, err
		}
		return uvo, nil
	}

	userDO, err := s.UserDao.GetUserByID(ctx, userid)
	if err != nil {
		return nil, err
	}

	uvo.UUID = userDO.UUID
	uvo.Nickname = userDO.Nickname
	uvo.Email = userDO.Email
	uvo.Avatar = userDO.Avatar

	jsonStr, err := json.Marshal(uvo)
	if err != nil {
		logger.Error(ctx, "Marshal error in get user info: %s", err.Error())
		return nil, err
	}

	if err = rdbClient.Set(rCtx, infoPrefix+userid, jsonStr, time.Minute*10).Err(); err != nil {
		logger.Error(ctx, "Set userinfo to redis error: %s", err.Error())
	}

	return uvo, nil
}

// GetAllUsers 获取所有用户信息
func (s *Service) GetAllUsers(ctx *gin.Context, users *user_vo.ListUserInfoVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	info, err := rdbClient.Get(rCtx, allinfoKey).Result()
	if err != nil && err != redis.Nil {
		logger.Error(ctx, "Get all userinfo from redis error: %s", err.Error())
		return err
	}

	if err == nil && info != "" {
		if err = json.Unmarshal([]byte(info), &users.Users); err != nil {
			logger.Error(ctx, "Unmarshal error in get all user info: %s", err.Error())
			return err
		}
		return nil
	}

	userDos, err := s.UserDao.GetAllUsers(ctx)
	if err != nil {
		return err
	}

	users.Code = 200
	users.Message = "Success get all user info"
	for _, userDo := range userDos {
		users.Users = append(users.Users, user_vo.TableUserVO{
			ID:        userDo.ID,
			UUID:      userDo.UUID,
			Nickname:  userDo.Nickname,
			Avatar:    userDo.Avatar,
			Email:     userDo.Email,
			LastLogin: userDo.LastLogin,
			UpdatedAt: userDo.UpdatedAt,
		})
	}

	jsonStr, err := json.Marshal(users.Users)
	if err != nil {
		logger.Error(ctx, "Marshal error in get all user info: %s", err.Error())
		return err
	}

	if err = rdbClient.Set(rCtx, allinfoKey, jsonStr, time.Minute*10).Err(); err != nil {
		logger.Error(ctx, "Set all userinfo to redis error: %s", err.Error())
	}

	return nil
}

// UpdateUserInfo 更新用户信息(名称 头像)
func (s *Service) UpdateUserInfo(ctx *gin.Context, req *user_dto.UpdateInfoRequest) error {
	UserID, _ := ctx.Get("user_id")
	userid, _ := UserID.(string)

	ext := filepath.Ext(req.Avatar.Filename)
	if ext == "" {
		ext = ".png" //设置默认扩展名
	}

	filname := "avatar" + "-" + userid + ext
	dst := filepath.Join("/static/avatar", filname)
	// 文件落盘
	err := ctx.SaveUploadedFile(req.Avatar, dst)
	if err != nil {
		logger.Error(ctx, "Save uploaded file error: %s", err.Error())
		return err
	}

	err = s.UserDao.UpdateUserInfo(ctx, userid, req.NickName, dst)
	// 如果更新错误则删除文件
	if err != nil {
		//删除文件
		if err = os.Remove(dst); err != nil {
			logger.Error(ctx, "Remove file error: %s", err.Error())
		}
		return err
	}

	// 删除缓存
	if delErr := rdb.Rdb.Del(rdb.Ctx, infoPrefix+userid).Err(); delErr != nil {
		logger.Error(ctx, "Delete redis cache error: %s", delErr.Error())
	}

	return nil
}
