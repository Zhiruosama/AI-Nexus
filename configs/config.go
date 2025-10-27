package configs

import (
	"strconv"
)

type ServerConfig struct {
	Host string
	Port int
}

type MysqlConfig struct {
	Host string
	Port int
	User string
	Pass string
}

type MysqlDataBase struct {
	MysqlConfig
	DataBase string
}

func (sc ServerConfig) SerizalString() (SerizalStr string) {
	SerizalStr = sc.Host + ":" + strconv.Itoa(sc.Port)
	return
}

func (mdb MysqlDataBase) DsnString() (DsnString string) {
	DsnString = mdb.User + ":" + mdb.Pass + "@tcp(" + mdb.Host + ":" + strconv.Itoa(mdb.Port) + ")/" + mdb.DataBase + "?charset=utf8mb4&parseTime=True&loc=Local"
	return
}
