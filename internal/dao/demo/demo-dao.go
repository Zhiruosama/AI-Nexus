// Package demo 是 demo 模块与数据库交互部分
package demo

import (
	demo_do "github.com/Zhiruosama/ai_nexus/internal/domain/do/demo"
	"github.com/Zhiruosama/ai_nexus/internal/pkg/db"
)

// DAO 作为 demo 模块的 dao 结构体
type DAO struct {
}

// GetMessageByID 通过 ID 访问 Message 的 dao 层
func (d *DAO) GetMessageByID(id int) (demo_do.TestDO, error) {
	var demoDo demo_do.TestDO
	result := db.GlobalDB.Raw("SELECT id, message FROM test WHERE id = ?", id).Scan(&demoDo)

	if result.Error != nil {
		return demo_do.TestDO{}, result.Error
	}
	return demoDo, nil
}
