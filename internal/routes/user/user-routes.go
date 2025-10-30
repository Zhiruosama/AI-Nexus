// Package user 用户相关路由组
package user

import (
	user_controller "github.com/Zhiruosama/ai_nexus/internal/controller/user"
	user_service "github.com/Zhiruosama/ai_nexus/internal/service/user"
	"github.com/gin-gonic/gin"
)

// InitUserRoutes 初始化演示模块的路由
func InitUserRoutes(r *gin.Engine) {
	us := user_service.NewService()
	uc := user_controller.NewController(us)

	demo := r.Group("/user")
	{
		demo.GET("/send-code", uc.SendEmailCode)
	}
}
