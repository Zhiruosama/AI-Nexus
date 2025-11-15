// Package user 用户服务
package user

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	codePrefix       = "code_"
	infoPrefix       = "info_"
	allinfoKey       = "allInfoForUsers"
	allinfoKeyPrefix = "allInfoForUsers_"
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
	do.Purpose = int8(dto.Purpose)

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

	if delErr := rdb.Rdb.Del(rdb.Ctx, allinfoKey).Err(); delErr != nil {
		logger.Error(ctx, "Delete all users cache error: %s", delErr.Error())
	}

	delUserInfoByPage(ctx)

	return nil
}

// LoginWithNicknamePassword 用户名密码登录
func (s *Service) LoginWithNicknamePassword(ctx *gin.Context, req *user_dto.LoginRequest, vo *user_vo.LoginVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx
	uuid, password, err := s.UserDao.GetPasswordByNickname(ctx, req.NickName)

	if err != nil {
		return err
	}
	if uuid == "" || password == "" {
		return fmt.Errorf("user not exists")
	}

	_, err = rdbClient.Get(rCtx, uuid).Result()
	if err == redis.Nil {
		ok, errs := pkg.VerifyPassword(req.PassWord, password)
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
func (s *Service) LoginWithEmailPassword(ctx *gin.Context, req *user_dto.LoginRequest, vo *user_vo.LoginVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	uuid, password, err := s.UserDao.GetPasswordByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if uuid == "" || password == "" {
		return fmt.Errorf("user not exists")
	}

	_, err = rdbClient.Get(rCtx, uuid).Result()
	if err == redis.Nil {
		ok, errs := pkg.VerifyPassword(req.PassWord, password)
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
func (s *Service) LoginWithEmailVerifyCode(ctx *gin.Context, req *user_dto.LoginRequest, vo *user_vo.LoginVO) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	code, err := rdbClient.Get(rCtx, codePrefix+req.Email).Result()
	if err != nil {
		logger.Error(ctx, "Get verify code error: %s", err.Error())
		return err
	}

	if code != req.VerifyCode {
		return fmt.Errorf("email verification code error")
	}

	uuid, _, err := s.UserDao.GetPasswordByEmail(ctx, req.Email)
	if err != nil {
		return err
	}
	if uuid == "" {
		return fmt.Errorf("user not exists")
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

	if err = rdbClient.Del(rCtx, codePrefix+req.Email).Err(); err != nil {
		logger.Error(ctx, "Remove code in redis error: %s", err.Error())
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

// GetAllUsers 获取全部用户信息
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
		users.Code = 200
		users.Message = "Success get all user info"
		users.Count = len(users.Users)
		return nil
	}

	userDos, count, err := s.UserDao.GetAllUsers(ctx)
	if err != nil {
		return err
	}

	users.Code = 200
	users.Message = "Success get all user info"
	users.Count = count
	for _, userDo := range userDos {
		users.Users = append(users.Users, user_vo.TableUserVO{
			ID:        userDo.ID,
			UUID:      userDo.UUID,
			Nickname:  userDo.Nickname,
			Avatar:    userDo.Avatar,
			Email:     userDo.Email,
			LastLogin: userDo.LastLogin,
			CreatedAt: userDo.CreatedAt,
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

// GetAllUsersByPage 获取所有用户信息-分页
func (s *Service) GetAllUsersByPage(ctx *gin.Context, users *user_vo.ListUserInfoVO, query *user_query.GetAllUsersQuery) error {
	users.PageIndex = query.PageIndex
	users.PageSize = query.PageSize

	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	key := fmt.Sprintf("%s%d_%d", allinfoKeyPrefix, query.PageIndex, query.PageSize)

	info, err := rdbClient.Get(rCtx, key).Result()
	if err != nil && err != redis.Nil {
		logger.Error(ctx, "Get all userinfo by page from redis error: %s", err.Error())
		return err
	}

	if err == nil && info != "" {
		if err = json.Unmarshal([]byte(info), &users.Users); err != nil {
			logger.Error(ctx, "Unmarshal error in get all user info by page: %s", err.Error())
			return err
		}
		users.Code = 200
		users.Message = "Success get all user info by page"
		users.Count = len(users.Users)
		return nil
	}

	userDos, count, err := s.UserDao.GetAllUsersByPage(ctx, query)
	if err != nil {
		return err
	}

	users.Code = 200
	users.Message = "Success get all user info by page"
	users.Count = int(count)
	for _, userDo := range userDos {
		users.Users = append(users.Users, user_vo.TableUserVO{
			ID:        userDo.ID,
			UUID:      userDo.UUID,
			Nickname:  userDo.Nickname,
			Avatar:    userDo.Avatar,
			Email:     userDo.Email,
			LastLogin: userDo.LastLogin,
			CreatedAt: userDo.CreatedAt,
			UpdatedAt: userDo.UpdatedAt,
		})
	}

	jsonStr, err := json.Marshal(users.Users)
	if err != nil {
		logger.Error(ctx, "Marshal error in get all user info by page: %s", err.Error())
		return err
	}

	if err = rdbClient.Set(rCtx, key, jsonStr, time.Minute*10).Err(); err != nil {
		logger.Error(ctx, "Set all userinfo by page to redis error: %s", err.Error())
	}

	return nil
}

// UpdateUserInfo 更新用户信息(名称 头像)
func (s *Service) UpdateUserInfo(ctx *gin.Context, req *user_dto.UpdateInfoRequest) error {
	UserID, _ := ctx.Get("user_id")
	userid, _ := UserID.(string)

	var nickName string
	var path string
	var dst string

	if req.NickName != "" {
		nickName = req.NickName
	}

	if req.Avatar != nil {
		if req.Sha256 == "" {
			return fmt.Errorf("please upload your file sha256 value")
		}

		file, err := req.Avatar.Open()
		if err != nil {
			logger.Error(ctx, "Open uploaded file error: %s", err.Error())
			return err
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				logger.Error(ctx, "Close uploaded file error: %s", closeErr.Error())
			}
		}()

		hash := sha256.New()
		if _, err = io.Copy(hash, file); err != nil {
			logger.Error(ctx, "calcute sha256 error: %s", err.Error())
			return err
		}
		calculatedHash := hex.EncodeToString(hash.Sum(nil))

		if calculatedHash != req.Sha256 {
			return fmt.Errorf("the file destroyed")
		}

		ext := filepath.Ext(req.Avatar.Filename)

		allowedExts := []string{".png", ".jpg", ".jpeg", ".webp"}
		isValid := slices.Contains(allowedExts, ext)
		if !isValid {
			return fmt.Errorf("unsupported file format: %s, only png, jpg, jpeg, webp are allowed", ext)
		}

		filname := "avatar-" + userid + ext
		dst = filepath.Join("static", "avatar", filname)

		err = ctx.SaveUploadedFile(req.Avatar, dst)
		if err != nil {
			logger.Error(ctx, "Save uploaded file error: %s", err.Error())
			return err
		}

		if ext != ".webp" {
			ok := pkg.ProcessImageToWebP(ctx, dst, 90)
			if !ok {
				if removeErr := os.Remove(dst); removeErr != nil {
					logger.Error(ctx, "Remove failed conversion file error: %s", removeErr.Error())
				}
				return fmt.Errorf("failed to convert image to webp format")
			}
		}
		finalFileName := "avatar-" + userid + ".webp"
		path = filepath.Join("/static/avatar", finalFileName)
		dst = filepath.Join("static", "avatar", finalFileName)
	}

	err := s.UserDao.UpdateUserInfo(ctx, userid, nickName, path)
	if err != nil {
		if dst != "" {
			if errs := os.Remove(dst); errs != nil {
				logger.Error(ctx, "Remove file error: %s", errs.Error())
			}
		}
		return err
	}

	if delErr := rdb.Rdb.Del(rdb.Ctx, infoPrefix+userid).Err(); delErr != nil {
		logger.Error(ctx, "Delete redis cache error: %s", delErr.Error())
	}
	if delErr := rdb.Rdb.Del(rdb.Ctx, allinfoKey).Err(); delErr != nil {
		logger.Error(ctx, "Delete all users cache error: %s", delErr.Error())
	}

	delUserInfoByPage(ctx)

	return nil
}

// DestroyUser 注销用户
func (s *Service) DestroyUser(ctx *gin.Context) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	uuidV, _ := ctx.Get("user_id")
	uuid := uuidV.(string)

	path, err := s.UserDao.GetUserAvatar(ctx, uuid)
	if err != nil {
		return err
	}

	err = s.UserDao.DestroyUser(ctx, uuid)
	if err != nil {
		return err
	}

	if rdberr := rdbClient.Del(rCtx, uuid).Err(); rdberr != nil {
		logger.Error(ctx, "Remove user form redis error: %s", rdberr.Error())
	}

	if rdberr := rdbClient.Del(rCtx, infoPrefix+uuid).Err(); rdberr != nil {
		logger.Error(ctx, "Remove user form redis error: %s", rdberr.Error())
	}

	if rdberr := rdbClient.Del(rCtx, allinfoKey).Err(); rdberr != nil {
		logger.Error(ctx, "Remove user form redis error: %s", rdberr.Error())
	}

	delUserInfoByPage(ctx)

	dst := strings.TrimPrefix(path, "/")
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		logger.Warn(ctx, "Remove avatar file error (non-critical): %s", err.Error())
	}

	return nil
}

// ResetUserPassword 更新用户密码
func (s *Service) ResetUserPassword(ctx *gin.Context, req *user_dto.UpdatePasswordRequest) error {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	uuid, err := s.UserDao.GetUserIDByEmail(ctx, req.Email)
	if err != nil {
		return err
	}

	if uuid == "" {
		return fmt.Errorf("user not exist")
	}

	exists, err := rdbClient.Exists(rCtx, uuid).Result()
	if err != nil {
		logger.Error(ctx, "check user exist error: %s", err.Error())
		return err
	}

	if exists != 0 {
		return fmt.Errorf("user is logged in so editing is prohibited")
	}

	code, err := rdbClient.Get(rCtx, codePrefix+req.Email).Result()
	if err != nil {
		logger.Error(ctx, "get code from redis error: %s", err.Error())
		return err
	}

	if code != req.VerifyCode {
		logger.Error(ctx, "verify code is not correct: %s", req.VerifyCode)
		return err
	}

	newPasswordHash, err := pkg.HashPassword(req.NewPassWord)
	if err != nil {
		logger.Error(ctx, "hashPassword error: %s", err.Error())
		return err
	}

	err = s.UserDao.ResetUserPassword(ctx, uuid, newPasswordHash)
	if err != nil {
		logger.Error(ctx, "ResetUserPassword error: %s", err.Error())
		return err
	}

	_, err = rdbClient.Del(rCtx, codePrefix+req.Email).Result()
	if err != nil {
		logger.Error(ctx, "Delete verify code error: %s", err.Error())
		return err
	}

	return nil
}

func delUserInfoByPage(ctx *gin.Context) {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	pattern := allinfoKeyPrefix + "*"

	iter := rdbClient.Scan(rCtx, 0, pattern, 0).Iterator()
	for iter.Next(rCtx) {
		key := iter.Val()
		if err := rdbClient.Del(rCtx, key).Err(); err != nil {
			logger.Error(ctx, "Delete key %s failed: %v", key, err)
		}
	}
	if err := iter.Err(); err != nil {
		logger.Error(ctx, "Scan error: %v", err)
	}
}
