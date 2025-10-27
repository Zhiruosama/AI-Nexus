package demo

import (
	"net/http"
	"strconv"

	demo_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/demo"
	demo_service "github.com/Zhiruosama/ai_nexus/internal/service/demo"
	"github.com/gin-gonic/gin"
)

type DemoController struct {
	demoService *demo_service.DemoService
}

func NewDemoController(ds *demo_service.DemoService) *DemoController {
	return &DemoController{
		demoService: ds,
	}
}

func (dc *DemoController) GetMessageById(c *gin.Context) {
	idStr := c.DefaultQuery("id", "1")
	id, _ := strconv.Atoi(idStr)

	demoQuery := &demo_query.DemoQuery{
		Id: id,
	}

	result, err := dc.demoService.GetMessageById(demoQuery)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"message": "This user doesn't exists",
		})
	}
	c.JSON(http.StatusOK, result)
}
