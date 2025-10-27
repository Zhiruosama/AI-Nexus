package demo

import (
	demo_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/demo"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
)

type DemoDAO struct {
}

func (dd *DemoDAO) GetMessageById(id int) (demo_do.DemoDo, error) {
	var demoDo demo_do.DemoDo
	result := db.GlobalDB.Raw("SELECT id, message FROM test WHERE id = ?", id).Scan(&demoDo)

	if result.Error != nil {
		return demo_do.DemoDo{}, result.Error
	}
	return demoDo, nil
}
