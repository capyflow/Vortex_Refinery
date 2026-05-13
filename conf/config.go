package conf

import (
	"fmt"
	"os"
	"strings"

	"github.com/capyflow/allspark-go/ds"
	vpkg "github.com/capyflow/vortexv3/pkg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Group      string          `yaml:"group"`       // 分组
	Port       int64           `yaml:"port"`        // 端口
	DBConfig   *ds.DsConfig    `yaml:"-"`           // 数据库配置（程序构造）
	Jwt        *vpkg.JwtOption `yaml:"jwt"`         // JWT 配置
	Server     ServerConfig    `yaml:"server"`
	Redis      RedisConfig     `yaml:"redis"`
	MongoDB    MongoDBConfig   `yaml:"mongodb"`
	Master     MasterConfig    `yaml:"master"`
	Worker     WorkerConfig    `yaml:"worker"`
	Workflows  WorkflowConfig  `yaml:"workflows"`
}

type ServerConfig struct {
	GRPCAddr string `yaml:"grpc_addr"`
	HTTPAddr string `yaml:"http_addr"`
}

type RedisConfig struct {
	Addr          string `yaml:"addr"`
	Password      string `yaml:"password"`
	DB            int    `yaml:"db"`
	StreamKey     string `yaml:"stream_key"`
	TaskStreamKey string `yaml:"task_stream_key"`
}

type MongoDBConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type MasterConfig struct {
	HeartbeatInterval string `yaml:"heartbeat_interval"`
	TaskPullInterval  string `yaml:"task_pull_interval"`
}

type WorkerConfig struct {
	MasterGRPCAddr string       `yaml:"master_grpc_addr"`
	Redis          RedisConfig  `yaml:"redis"`
	Plugin         PluginConfig `yaml:"plugin"`
}

type PluginConfig struct {
	Dir              string          `yaml:"dir"`
	ReloadInterval   string          `yaml:"reload_interval"`
	TrustedSigners   []TrustedSigner `yaml:"trusted_signers"`
	RequireSignature bool            `yaml:"require_signature"`
}

type TrustedSigner struct {
	Name      string `yaml:"name"`
	Algorithm string `yaml:"algorithm"`
	PublicKey string `yaml:"public_key"`
}

type WorkflowConfig struct {
	Dir string `yaml:"dir"`
}

// Load reads and parses a YAML config file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// BuildDBConfig 构建 allspark-go 所需的 DBConfig
func (c *Config) BuildDBConfig() {
	c.DBConfig = &ds.DsConfig{
		Redis: &ds.DataBaseConfig{
			Host:     c.extractHost(c.Redis.Addr),
			Port:     c.extractPort(c.Redis.Addr),
			Username: "",
			Password: c.Redis.Password,
		},
		Mongo: &ds.DataBaseConfig{
			Host:     c.extractHost(c.MongoDB.URI),
			Port:     c.extractPort(c.MongoDB.URI),
			Username: c.extractUsername(c.MongoDB.URI),
			Password: c.extractPassword(c.MongoDB.URI),
		},
	}
}

// extractHost 从 addr 字符串中提取 host
func (c *Config) extractHost(addr string) string {
	if addr == "" {
		return ""
	}
	if strings.Contains(addr, "://") {
		parts := strings.Split(addr, "://")
		after := parts[1]
		if strings.Contains(after, "@") {
			return strings.Split(after, "@")[1]
		}
		return after
	}
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[0]
	}
	return addr
}

// extractPort 从 addr 字符串中提取 port
func (c *Config) extractPort(addr string) int {
	if addr == "" {
		return 0
	}
	var portStr string
	if strings.Contains(addr, "://") {
		parts := strings.Split(addr, "://")
		after := parts[1]
		if strings.Contains(after, "@") {
			after = strings.Split(after, "@")[1]
		}
		if strings.Contains(after, "/") {
			portStr = strings.Split(after, "/")[0]
		} else {
			portStr = after
		}
	} else {
		parts := strings.Split(addr, ":")
		if len(parts) == 2 {
			portStr = parts[1]
		}
	}
	var port int
	fmt.Sscanf(portStr, "%d", &port)
	return port
}

// extractUsername 从 mongodb URI 中提取 username
func (c *Config) extractUsername(uri string) string {
	if !strings.Contains(uri, "://") {
		return ""
	}
	parts := strings.Split(uri, "://")[1]
	if strings.Contains(parts, "@") {
		userPart := strings.Split(parts, "@")[0]
		if strings.Contains(userPart, ":") {
			return strings.Split(userPart, ":")[0]
		}
		return userPart
	}
	return ""
}

// extractPassword 从 mongodb URI 中提取 password
func (c *Config) extractPassword(uri string) string {
	if !strings.Contains(uri, "://") {
		return ""
	}
	parts := strings.Split(uri, "://")[1]
	if strings.Contains(parts, "@") {
		userPart := strings.Split(parts, "@")[0]
		if strings.Contains(userPart, ":") {
			return strings.Split(userPart, ":")[1]
		}
	}
	return ""
}
