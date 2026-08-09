// Package configs 提供配置文件的统一入口,读取 yaml 文件以及提供序列化操作
package configs

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

// GlobalConfig 是配置文件的全局唯一实例
var GlobalConfig *Config

// Config 定义统一配置文件结构
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	MySQL         MysqlConfig         `yaml:"mysql"`
	Redis         RedisConfig         `yaml:"redis"`
	RateLimit     RateLimitConfig     `yaml:"ratelimit"`
	Idempotency   IdempotencyConfig   `yaml:"idempotency"`
	Deduplication DeduplicationConfig `yaml:"deduplication"`
	Mail          MailConfig          `yaml:"mail"`
	RabbitMQ      RabbitMQConfig      `yaml:"rabbitmq"`
	Chat          ChatConfig          `yaml:"chat"`
}

// ServerConfig 定义主服务配置
type ServerConfig struct {
	Host   string `yaml:"host"`
	Public string `yaml:"public"`
	Port   int    `yaml:"port"`
}

// MysqlConfig 定义 Mysql 相关配置
type MysqlConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Pass     string `yaml:"pass"`
	DataBase string `yaml:"database"`
}

// RedisConfig 定义 Redis 相关配置
type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// RateLimitConfig 定义限流参数
type RateLimitConfig struct {
	LimitMax int           `yaml:"limitmax"` //单次窗口最大请求次数
	Window   time.Duration `yaml:"window"`   //窗口持续时长
}

// IdempotencyConfig 定义幂等性参数
type IdempotencyConfig struct {
	LockDuration time.Duration `yaml:"lockduration"`
}

// DeduplicationConfig 重复请求判断参数
type DeduplicationConfig struct {
	LockDuration time.Duration `yaml:"lockduration"`
}

// MailConfig 定义 Mail Service 客户端、投递回调和验证码策略。
type MailConfig struct {
	Address                string        `yaml:"address"`
	Timeout                time.Duration `yaml:"timeout"`
	AllowInsecure          bool          `yaml:"allowinsecure"`
	TLSCAFile              string        `yaml:"tlscafile"`
	TLSServerName          string        `yaml:"tlsservername"`
	CallbackAddress        string        `yaml:"callbackaddress"`
	CallbackAllowInsecure  bool          `yaml:"callbackallowinsecure"`
	CallbackTLSCertFile    string        `yaml:"callbacktlscertfile"`
	CallbackTLSKeyFile     string        `yaml:"callbacktlskeyfile"`
	SenderIdentityKey      string        `yaml:"senderidentitykey"`
	Locale                 string        `yaml:"locale"`
	VerificationTTL        time.Duration `yaml:"verificationttl"`
	DispatchDeadline       time.Duration `yaml:"dispatchdeadline"`
	PendingTTL             time.Duration `yaml:"pendingttl"`
	Cooldown               time.Duration `yaml:"cooldown"`
	MaxAttempts            int           `yaml:"maxattempts"`
	ReconcileInterval      time.Duration `yaml:"reconcileinterval"`
	HMACSecret             string        `yaml:"hmacsecret"`
	EmailFingerprintSecret string        `yaml:"emailfingerprintsecret"`
}

// RabbitMQConfig 定义 RabbitMQ 相关配置
type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	VHost    string `yaml:"vhost"`
}

// SerialString 返回服务信息的序列化字符串
func (sc ServerConfig) SerialString() string {
	return fmt.Sprintf("%s:%d", sc.Host, sc.Port)
}

// SerialStringPublic 返回公网服务信息的序列化字符串
func (sc ServerConfig) SerialStringPublic() string {
	return fmt.Sprintf("%s:%d", sc.Public, sc.Port)
}

// DsnString 返回 DSN 信息的序列化字符串
func (mc MysqlConfig) DsnString() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", mc.User, mc.Pass, mc.Host, mc.Port, mc.DataBase)
}

// URLString 返回 RabbitMQ 连接 URL
func (rc RabbitMQConfig) URLString() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s", rc.User, rc.Password, rc.Host, rc.Port, rc.VHost)
}

// ChatConfig 定义 AI 对话相关配置
type ChatConfig struct {
	EncryptionKey      string `yaml:"encryptionkey"`
	MaxMessagesPerConv int    `yaml:"maxmessagesperconv"`
}

func init() {
	var err error
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	GlobalConfig, err = loadConfig(configPath)

	if err != nil {
		panic(fmt.Sprintf("[ERROR] Failed to load config: %s\n", err.Error()))
	}
	log.Println("[Config] Config loaded successfully")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Allow local, non-committed configuration to refer to environment variables.
	// This keeps credentials out of tracked YAML while preserving the existing
	// configuration structure.
	data = []byte(os.ExpandEnv(string(data)))

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
