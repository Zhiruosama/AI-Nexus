// Package internal 私有方法使用集合，提供 app 初始化方法
package internal

import (
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	"github.com/Zhiruosama/ai_nexus/internal/middleware"
	routes_demo "github.com/Zhiruosama/ai_nexus/internal/routes/demo"
	routes_image_generation "github.com/Zhiruosama/ai_nexus/internal/routes/image-generation"
	routes_user "github.com/Zhiruosama/ai_nexus/internal/routes/user"
	"github.com/gin-gonic/gin"
)

// Run 运行一个 app 实例
func Run() {
	// 初始化路由引擎
	 gin.SetMode(gin.ReleaseMode)
	route := gin.New()

	// 注册全局中间件
	route.Use(middleware.Recovery())
	route.Use(gin.Logger())
	route.Use(middleware.RequestID())
	route.Use(middleware.SecurityHeaders())
	route.Use(middleware.CORS(middleware.DefaultCORSConfig()))

	// 静态目录
	route.Static("/static", "./static")

	// 注册路由
	routes_demo.InitDemoRoutes(route)
	routes_user.InitUserRoutes(route)
	routes_image_generation.InitImageGenerationRoutes(route)

	// 启动 app
	log.Printf("[Server] Server start on: %s:%d\n", configs.GlobalConfig.Server.Host, configs.GlobalConfig.Server.Port)
	err := route.Run(configs.GlobalConfig.Server.SerialString())
	if err != nil {
		log.Fatalln("[ERROR] Server start error:", err.Error())
	}
}
