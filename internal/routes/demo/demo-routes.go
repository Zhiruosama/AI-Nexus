// Package demo 提供演示相关的路由配置
package demo

import (
	demo_controller "github.com/Zhiruosama/ai_nexus/internal/controller/demo"
	demo_service "github.com/Zhiruosama/ai_nexus/internal/service/demo"
	"github.com/gin-gonic/gin"
)

// InitDemoRoutes 初始化演示模块的路由
func InitDemoRoutes(r *gin.Engine) {
	ds := demo_service.NewService()
	dc := demo_controller.NewController(ds)

	demo := r.Group("/demo")
	{
		demo.GET("/get-message", dc.GetMessageByID)
		demo.GET("/panic-test", dc.PanicTest)
	}
}
