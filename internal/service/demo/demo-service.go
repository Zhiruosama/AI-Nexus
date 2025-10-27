// Package demo 是 demo 模块的 service 部分，书写业务处理
package demo

import (
	"log"

	demo_dao "github.com/Zhiruosama/ai_nexus/internal/dao/demo"
	demo_query "github.com/Zhiruosama/ai_nexus/internal/domain/query/demo"
	demo_vo "github.com/Zhiruosama/ai_nexus/internal/domain/vo/demo"
)

// Service 对应 demo 模块的 Service 结构
type Service struct {
	DemoDao *demo_dao.DAO
}

// NewService 对应 demo 模块的 Service 工厂方法
func NewService() *Service {
	return &Service{
		DemoDao: &demo_dao.DAO{},
	}
}

// GetMessageByID 对应 Service 的处理
func (ds *Service) GetMessageByID(dqu *demo_query.GetMessageByIDQuery) (demo_vo.GetMessageByIDVO, error) {
	demoDO, err := ds.DemoDao.GetMessageByID(dqu.ID)

	if err != nil {
		log.Fatalln("[ERROR] GetMessageById in sql error:", err.Error())
		return demo_vo.GetMessageByIDVO{}, err
	}

	demoVo := demo_vo.GetMessageByIDVO{
		ID:      demoDO.ID,
		Message: demoDO.Message,
	}
	return demoVo, nil
}
