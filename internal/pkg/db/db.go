package db

import (
	"log"

	"github.com/Zhiruosama/ai_nexus/configs"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GlobalDB *gorm.DB

func init() {
	dsn := configs.GlobalConfig.MySQL.DsnString()

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalln("[ERROR] DBinit error, err is:", err.Error())
	}

	GlobalDB = db
}
