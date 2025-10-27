// Package demo 是 demo 模块的 controller 部分，集中书写所有回调
package demo

import (
	"net/http"
	"strconv"

	demo_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/demo"
	demo_service "github.com/Zhiruosama/ai_nexus/internal/service/demo"
	"github.com/gin-gonic/gin"
)

// Controller 对应 Controller 结构，有一个 Service 成员
type Controller struct {
	demoService *demo_service.Service
}

// NewController 对应 Controller 的工厂方法
func NewController(ds *demo_service.Service) *Controller {
	return &Controller{
		demoService: ds,
	}
}

// GetMessageByID 通过 ID 查询 test 表的对应 message
func (c *Controller) GetMessageByID(ctx *gin.Context) {
	idStr := ctx.DefaultQuery("id", "1")
	id, _ := strconv.Atoi(idStr)

	demoQuery := &demo_query.GetMessageByIDQuery{
		ID: id,
	}

	result, err := c.demoService.GetMessageByID(demoQuery)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"message": "This user doesn't exists",
		})
	}
	ctx.JSON(http.StatusOK, result)
}
