// Package configs 提供配置文件的统一入口,读取 yaml 文件以及提供序列化操作
package configs

import (
	"fmt"
	"log"
	"os"

	"github.com/goccy/go-yaml"
)

// GlobalConfig 是配置文件的全局唯一实例
var GlobalConfig *Config

// Config 定义统一配置文件结构
type Config struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MysqlConfig  `yaml:"mysql"`
}

// ServerConfig 定义主服务配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// MysqlConfig 定义 Mysql 相关配置
type MysqlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Pass     string `yaml:"pass"`
	DataBase string `yaml:"database"`
}

// SerialString 返回服务信息的序列化字符串
func (sc ServerConfig) SerialString() string {
	return fmt.Sprintf("%s:%d", sc.Host, sc.Port)
}

// DsnString 返回 DSN 信息的序列化字符串
func (mc MysqlConfig) DsnString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", mc.User, mc.Pass, mc.Host, mc.Port, mc.DataBase)
}

func init() {
	var err error
	GlobalConfig, err = loadConfig("configs/config.yaml")

	if err != nil {
		log.Fatalln("[ERROR] Failed to load config:", err.Error())
	}
	log.Println("[INFO] Config loaded successfully")
}

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
