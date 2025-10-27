package internal

import (
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	rdemo "github.com/Zhiruosama/ai_nexus/internal/routes/demo"
	"github.com/gin-gonic/gin"
)

func Run() {
	var serverConfig configs.ServerConfig = configs.ServerConfig{
		Host: "127.0.0.1",
		Port: 9000,
	}

	gin.SetMode(gin.ReleaseMode)
	route := gin.Default()
	rdemo.InitDemoRoutes(route)

	log.Printf("[INFO] Server start on port: %d", serverConfig.Port)
	err := route.Run(serverConfig.SerizalString())
	if err != nil {
		log.Fatalln("[ERROR] Server start error:", err.Error())
	}
}
