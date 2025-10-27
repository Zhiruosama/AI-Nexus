package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/goccy/go-yaml"
)

var GlobalConfig *Config

type Config struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MysqlConfig  `yaml:"mysql"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type MysqlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Pass     string `yaml:"pass"`
	DataBase string `yaml:"database"`
}

// 配置文件初始化
func init() {
	var err error
	GlobalConfig, err = loadConfig("configs/config.yaml")

	if err != nil {
		log.Fatalln("[ERROR] Failed to load config:", err.Error())
	}
	log.Println("[INFO] Config loaded successfully")
}

// 解析配置文件
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// 服务信息序列字符串
func (sc ServerConfig) SerialString() string {
	return fmt.Sprintf("%s:%d", sc.Host, sc.Port)
}

// dsn信息序列化
func (mc MysqlConfig) DsnString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", mc.User, mc.Pass, mc.Host, mc.Port, mc.DataBase)
}
