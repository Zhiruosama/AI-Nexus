// Package db 数据库初始模块
package db

import (
	"fmt"

	"github.com/Zhiruosama/ai_nexus/configs"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// GlobalDB 全局的 Mysql 实例
var GlobalDB *gorm.DB

// mysql 初始化
func init() {
	dsn := configs.GlobalConfig.MySQL.DsnString()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(fmt.Sprintf("[ERROR] DBinit error, err is: %s", err.Error()))
	}

	GlobalDB = db
	fmt.Println("Mysql connect success")
}
