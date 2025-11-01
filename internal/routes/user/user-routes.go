// Package user 用户相关路由组
package user

import (
	user_controller "github.com/Zhiruosama/ai_nexus/internal/controller/user"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	user_service "github.com/Zhiruosama/ai_nexus/internal/service/user"
	"github.com/gin-gonic/gin"
)

// InitUserRoutes 初始化演示模块的路由
func InitUserRoutes(r *gin.Engine) {
	us := user_service.NewService()
	uc := user_controller.NewController(us)

	user := r.Group("/user")
	{
		user.POST("/send-code", uc.SendEmailCode)
		user.POST("/register", uc.Register)
		user.GET("/login", uc.Login)
		user.GET("/logout", middleware.AuthMiddleware(), middleware.RateLimitingMiddleware(), middleware.DeduplicationMiddleware(), uc.Logout)
		user.GET("/getuserinfo", middleware.AuthMiddleware(), middleware.RateLimitingMiddleware(), middleware.DeduplicationMiddleware(), uc.GetUserInfo)
	}
}
