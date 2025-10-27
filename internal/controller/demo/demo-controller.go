package demo

import (
	"net/http"
	"strconv"

	dquery "github.com/Zhiruosama/ai_nexus/internal/domain/query/demo"
	"github.com/Zhiruosama/ai_nexus/internal/service/demo"
	"github.com/gin-gonic/gin"
)

type DemoController struct {
	demoService *demo.DemoService
}

func NewDemoController(ds *demo.DemoService) *DemoController {
	return &DemoController{
		demoService: ds,
	}
}

func (dc *DemoController) GetMessageById(c *gin.Context) {
	idStr := c.DefaultQuery("id", "1")
	id, _ := strconv.Atoi(idStr)

	demoQuery := &dquery.DemoQuery{
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
