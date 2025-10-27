package demo

import (
	"log"

	demoDAO "github.com/Zhiruosama/ai_nexus/internal/dao/demo"
	dquery "github.com/Zhiruosama/ai_nexus/internal/domain/query/demo"
	demoVO "github.com/Zhiruosama/ai_nexus/internal/domain/vo/demo"
)

type DemoService struct {
	DemoDao *demoDAO.DemoDAO
}

func NewDemoService() *DemoService {
	return &DemoService{
		DemoDao: &demoDAO.DemoDAO{},
	}
}

func (ds *DemoService) GetMessageById(dqu *dquery.DemoQuery) (demoVO.DemoVO, error) {
	demoDO, err := ds.DemoDao.GetMessageById(dqu.Id)

	if err != nil {
		log.Fatalln("[ERROR] GetMessageById in sql error:", err.Error())
		return demoVO.DemoVO{}, err
	}

	demoVo := demoVO.DemoVO{
		Id:      demoDO.Id,
		Message: demoDO.Message,
	}
	return demoVo, nil
}
