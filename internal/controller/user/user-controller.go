// Package user 用户controller
package user

import (
	"math/rand"
	"net/http"
	"regexp"
	"time"

	user_dto "github.com/Zhiruosama/ai_nexus/internal/domain/dto/user"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/rdb"
	user_service "github.com/Zhiruosama/ai_nexus/internal/service/user"
	"github.com/gin-gonic/gin"
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
)

// SendEmailCode 发送验证码
func (uc *Controller) SendEmailCode(ctx *gin.Context) {
	rdbClient := rdb.Rdb
	rCtx := rdb.Ctx
	defaultName := nickNamePrefix + generateRandomString(5)

	nickName := ctx.DefaultPostForm("nickname", defaultName)
	email := ctx.DefaultPostForm("email", "")
	password := ctx.DefaultPostForm("password", "-1")
	repeatPassword := ctx.DefaultPostForm("repeat_password", "-1")

	if email == "" || password == "-1" || repeatPassword == "=1" {
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

	if password != repeatPassword {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.PasswordMismatch,
			"message": "The entered password was not equal.",
		})
		return
	}

	if !passwordValidator.MatchString(password) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"code":    middleware.PasswordInvalid,
			"message": "Password format is invalid. It must be 6-20 characters long and contain only letters, numbers, and symbols: !@#$%^&*",
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

	dto := &user_dto.SendEmailCode{
		NickName:       nickName,
		Email:          email,
		PassWord:       password,
		RepeatPassWord: repeatPassword,
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

	if req.NickName == "" {
		req.NickName = nickNamePrefix + generateRandomString(5)
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

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(n int) string {
	b := make([]byte, n)

	for i := range b {
		randomIndex := seededRand.Intn(len(charset))
		b[i] = charset[randomIndex]
	}

	return string(b)
}
