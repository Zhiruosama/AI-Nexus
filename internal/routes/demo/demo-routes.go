package demo

import (
	dC "github.com/Zhiruosama/ai_nexus/internal/controller/demo"
	dS "github.com/Zhiruosama/ai_nexus/internal/service/demo"
	"github.com/gin-gonic/gin"
)

func InitDemoRoutes(r *gin.Engine) {
	ds := dS.NewDemoService()
	dc := dC.NewDemoController(ds)

	demo := r.Group("/demo")
	demo.Use(func(ctx *gin.Context) {
		ctx.Writer.WriteString("Begin demo")
	})
	{
		demo.GET("/get-message", dc.GetMessageById)
	}
}
