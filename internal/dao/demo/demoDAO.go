package demo

import (
	"github.com/Zhiruosama/ai_nexus/internal/domain/do/demo"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
)

type DemoDAO struct {
}

func (dd *DemoDAO) GetMessageById(id int) (demo.DemoDo, error) {
	var demoDo demo.DemoDo
	result := db.GlobalDB.Raw("SELECT id, message FROM test WHERE id = ?", id).Scan(&demoDo)

	if result.Error != nil {
		return demo.DemoDo{}, result.Error
	}
	return demoDo, nil
}
