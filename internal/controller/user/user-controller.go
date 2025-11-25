// Package user 用户controller
package user

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	user_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/user"
	user_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/user"
	user_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/user"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/ws"
	user_service "github.com/Zhiruosama/ai_nexus/internal/service/user"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Controller 对应 Controller 结构，有一个 Service 成员
type Controller struct {
	UserService *user_service.Service
}

// NewController 对应 Controller 的工厂方法
func NewController(us *user_service.Service) *Controller {
	return &Controller{
		UserService: us,
	}
}

const (
	passwordRegex  = `^[a-zA-Z0-9!@#$%^&*]{6,20}$`
	emailRegex     = `^[^@\s]+@[^@\s]+\.[^@\s]+$`
	charset        = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" + "0123456789"
	nickNamePrefix = "用户_"
	codePrefix     = "code_"
)

var (
	passwordValidator = regexp.MustCompile(passwordRegex)
	emailValidator    = regexp.MustCompile(emailRegex)
	seededRand        = rand.New(rand.NewSource(time.Now().UnixNano()))

	upgrader = websocket.Upgrader{
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(_ *http.Request) bool {
			// origin := r.Header.Get("Origin")
			// allowedOrigins := []string{"https://chulan.xin"}
			// for _, allowed := range allowedOrigins {
			//     if origin == allowed { return true }
			// }
			// return false

			// 暂时允许所有来源
			return true
		},
	}
)

// SendEmailCode 发送验证码
func (uc *Controller) SendEmailCode(ctx *gin.Context) {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	email := ctx.DefaultPostForm("email", "")
	purposeStr := ctx.DefaultPostForm("purpose", "")

	if email == "" || purposeStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if !emailValidator.MatchString(email) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.EmailInvalid,
			"message": "email format is invalid.",
		})
		return
	}

	_, err := rdbClient.Get(rCtx, codePrefix+email).Result()
	if err == nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.VerifyCodeExist,
			"message": "varify code already exists",
		})
		return
	}

	purpose, err := strconv.Atoi(purposeStr)
	if err != nil || (purpose != 1 && purpose != 2) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.PurposeInvalid,
			"message": "Invalid purpose value",
		})
		return
	}

	dto := &user_dto.SendEmailCode{
		Purpose: purpose,
		Email:   email,
	}

	err = uc.UserService.SendEmailCode(ctx, dto)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.RPCSendCodeFailed,
			"message": "Verification code failed to send",
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "send email successful",
	})
}

// Register 用户注册
func (uc *Controller) Register(ctx *gin.Context) {
	var req user_dto.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if req.Purpose != "1" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The purpose is not register",
		})
		return
	}

	if req.NickName == "" {
		req.NickName = nickNamePrefix + generateRandomString(5)
	}

	if req.Email == "" || req.PassWord == "" || req.RepeatPassWord == "" || req.VerifyCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if !emailValidator.MatchString(req.Email) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.EmailInvalid,
			"message": "email format is invalid.",
		})
		return
	}

	if req.PassWord != req.RepeatPassWord {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.PasswordMismatch,
			"message": "The entered password was not equal.",
		})
		return
	}

	if !passwordValidator.MatchString(req.PassWord) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.PasswordInvalid,
			"message": "Password format is invalid. It must be 6-20 characters long and contain only letters, numbers, and symbols: !@#$%^&*",
		})
		return
	}

	// 调用服务层进行注册
	err := uc.UserService.Register(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.RegisterFailed,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "register successful",
	})
}

// Login 登录
func (uc *Controller) Login(ctx *gin.Context) {
	var req = &user_dto.LoginRequest{}
	var loginvo = &user_vo.LoginVO{}

	if err := ctx.ShouldBind(req); err != nil {
		loginvo.Code = int32(middleware.ParamEmpty)
		loginvo.Message = "The input data does not meet the requirements."
		ctx.JSON(http.StatusBadRequest, loginvo)
		return
	}

	if req.Email == "" && req.NickName == "" {
		loginvo.Code = int32(middleware.UserInformationEmpty)
		loginvo.Message = "User information cannot be empty"
		loginvo.JWTToken = ""
		ctx.JSON(http.StatusBadRequest, loginvo)
		return
	}

	if req.PassWord == "" && req.VerifyCode == "" {
		loginvo.Code = int32(middleware.PasswordEmpty)
		loginvo.Message = "Password cannot be empty"
		loginvo.JWTToken = ""
		ctx.JSON(http.StatusBadRequest, loginvo)
		return
	}

	var err error

	// 校验参数 判断是用用户密码登录 还是用邮箱验证码登录
	if req.NickName != "" && req.PassWord != "" {
		err = uc.UserService.LoginWithNicknamePassword(ctx, req, loginvo)
	} else if req.Email != "" && req.VerifyCode != "" {

		if req.Purpose != "3" {
			loginvo.Code = int32(middleware.LoginPurposeError)
			loginvo.Message = "The purpose login error"
			loginvo.JWTToken = ""
			ctx.JSON(http.StatusBadRequest, loginvo)
			return
		}

		err = uc.UserService.LoginWithEmailVerifyCode(ctx, req, loginvo)
	} else if req.Email != "" && req.PassWord != "" {
		err = uc.UserService.LoginWithEmailPassword(ctx, req, loginvo)
	}

	if err != nil {
		loginvo.Code = int32(middleware.LoginFailed)
		loginvo.Message = err.Error()
		loginvo.JWTToken = ""
		ctx.JSON(http.StatusBadRequest, loginvo)
		return
	}

	loginvo.Code = int32(http.StatusOK)
	loginvo.Message = "login successful"

	ctx.JSON(http.StatusOK, loginvo)
}

// Logout 登出
func (uc *Controller) Logout(ctx *gin.Context) {
	UserID, _ := ctx.Get("user_id")
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	_, err := rdbClient.Get(rCtx, UserID.(string)).Result()
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"message": "user already logged out",
		})
		return
	}

	_, err = rdbClient.Del(rCtx, UserID.(string)).Result()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.LogoutFailed,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "logout successful",
	})
}

// GetUserInfo 获取当前已登录用户信息
func (uc *Controller) GetUserInfo(ctx *gin.Context) {
	UserID, _ := ctx.Get("user_id")

	uservo, err := uc.UserService.GetUserInfo(ctx, UserID.(string))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.GetUserInfoFailed,
			"message": err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, uservo)
}

// GetAllUsers 获取所有用户信息
func (uc *Controller) GetAllUsers(ctx *gin.Context) {
	users := &user_vo.ListUserInfoVO{}
	query := &user_query.GetAllUsersQuery{
		PageIndex: -1,
		PageSize:  -1,
	}

	pageIndex := ctx.Query("pageIndex")
	pageSize := ctx.Query("pageSize")

	if pageIndex != "" {
		query.PageIndex, _ = strconv.Atoi(pageIndex)
		query.PageSize, _ = strconv.Atoi(pageSize)
	}

	var err error

	if query.PageIndex != -1 {
		err = uc.UserService.GetAllUsersByPage(ctx, users, query)
	} else {
		err = uc.UserService.GetAllUsers(ctx, users)
	}

	if err != nil {
		users.Code = int32(middleware.GetAllUserInfoFailed)
		users.Message = fmt.Sprintf("Failed to get all user info, err: %s", err.Error())
		users.Count = 0
		users.Users = nil
		ctx.JSON(http.StatusBadRequest, users)
		return
	}

	ctx.JSON(http.StatusOK, users)
}

// UpdateUserInfo 更新用户信息
func (uc *Controller) UpdateUserInfo(ctx *gin.Context) {
	var req *user_dto.UpdateInfoRequest

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if req.Avatar == nil && req.NickName == "" && req.Sha256 == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	err := uc.UserService.UpdateUserInfo(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.UpdateUserInfoFailed,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "update user info successful",
	})
}

// ResetUserPassword 更新用户密码
func (uc *Controller) ResetUserPassword(ctx *gin.Context) {
	var req *user_dto.UpdatePasswordRequest

	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if req.Purpose != "2" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ResetPasswordPurposeError,
			"message": "The purpose reset password error",
		})
	}

	if req.Email == "" || req.NewPassWord == "" || req.RepeatNewPass == "" || req.VerifyCode == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The input data does not meet the requirements.",
		})
		return
	}

	if req.NewPassWord != req.RepeatNewPass {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParamEmpty,
			"message": "The two new passwords are different.",
		})
		return
	}

	if !passwordValidator.MatchString(req.NewPassWord) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.PasswordInvalid,
			"message": "Password format is invalid. It must be 6-20 characters long and contain only letters, numbers, and symbols: !@#$%^&*",
		})
		return
	}

	err := uc.UserService.ResetUserPassword(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.UpdateUserInfoFailed,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "update user password successful",
	})
}

// DestroyUser 注销用户
func (uc *Controller) DestroyUser(ctx *gin.Context) {
	err := uc.UserService.DestroyUser(ctx)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.DestoryUserFailed,
			"message": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"message": "delete user info successful",
	})
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(n int) string {
	b := make([]byte, n)

	for i := range b {
		randomIndex := seededRand.Intn(len(charset))
		b[i] = charset[randomIndex]
	}

	return string(b)
}

// HandleWebSocket 处理 WebSocket 连接
func (uc *Controller) HandleWebSocket(ctx *gin.Context) {
	tokenString := ctx.Query("token")
	claims, err := middleware.ParseToken(tokenString)

	if err != nil {
		log.Printf("[WebSocket] Parse token failed: %v\n", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.ParseTokenFailed,
			"message": err.Error(),
		})
		return
	}

	userUUID := claims.UserID

	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx

	exists, err := rdbClient.Exists(rCtx, userUUID).Result()
	if err != nil {
		log.Printf("[WebSocket] Redis error: %v\n", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"code":    middleware.RedisError,
			"message": "Redis error: " + err.Error(),
		})
		return
	}

	if exists == 0 {
		log.Printf("[WebSocket] User %s not exists\n", userUUID)
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"code":    middleware.UserNotExists,
			"message": "user not exists",
		})
		return
	}

	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade failed: %v\n", err)
		return
	}

	client := ws.NewClient(userUUID, conn, ws.GlobalHub)
	ws.GlobalHub.Register(client)

	go client.ReadPump()
	go client.WritePump()

	log.Printf("[WebSocket] Client %s connection established, pumps started\n", userUUID)
}
