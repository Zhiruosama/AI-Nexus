package demo

import (
	"log"

	demo_dao "github.com/Zhiruosama/ai_nexus/internal/dao/demo"
	demo_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/demo"
	demo_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/demo"
)

type DemoService struct {
	DemoDao *demo_dao.DemoDAO
}

func NewDemoService() *DemoService {
	return &DemoService{
		DemoDao: &demo_dao.DemoDAO{},
	}
}

func (ds *DemoService) GetMessageById(dqu *demo_query.DemoQuery) (demo_vo.DemoVO, error) {
	demoDO, err := ds.DemoDao.GetMessageById(dqu.Id)

	if err != nil {
		log.Fatalln("[ERROR] GetMessageById in sql error:", err.Error())
		return demo_vo.DemoVO{}, err
	}

	demoVo := demo_vo.DemoVO{
		Id:      demoDO.Id,
		Message: demoDO.Message,
	}
	return demoVo, nil
}
