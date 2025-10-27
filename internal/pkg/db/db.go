package db

import (
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GlobalDB *gorm.DB

func init() {
	var mysqlDataBase configs.MysqlDataBase = configs.MysqlDataBase{
		MysqlConfig: configs.MysqlConfig{
			Host: "127.0.0.1",
			Port: 3306,
			User: "ainexus",
			Pass: "845924",
		},
		DataBase: "ai_nexus",
	}

	dsn := mysqlDataBase.DsnString()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln("[ERROR] DBinit error, err is:", err.Error())
	}

	GlobalDB = db
}
