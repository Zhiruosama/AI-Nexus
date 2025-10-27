package internal

import (
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	routes_demo "github.com/Zhiruosama/ai_nexus/internal/routes/demo"
	"github.com/gin-gonic/gin"
)

func Run() {
	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()
	routes_demo.InitDemoRoutes(route)

	log.Printf("[INFO] Server start on port: %d", configs.GlobalConfig.Server.Port)
	err := route.Run(configs.GlobalConfig.Server.SerialString())
	if err != nil {
		log.Fatalln("[ERROR] Server start error:", err.Error())
	}
}
