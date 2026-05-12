package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig   `yaml:"server"`
	Redis     RedisConfig    `yaml:"redis"`
	MongoDB   MongoDBConfig  `yaml:"mongodb"`
	Master    MasterConfig   `yaml:"master"`
	Worker    WorkerConfig   `yaml:"worker"`
	Workflows WorkflowConfig `yaml:"workflows"`
}

type WorkflowConfig struct {
	Dir string `yaml:"dir"` // JSON file storage directory for workflows
}

type ServerConfig struct {
	GRPCAddr string `yaml:"grpc_addr"`
	HTTPAddr string `yaml:"http_addr"` // REST API listen address, e.g. ":8080"
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
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
	TaskPullInterval  time.Duration `yaml:"task_pull_interval"`
}

type WorkerConfig struct {
	MasterGRPCAddr string        `yaml:"master_grpc_addr"`
	Redis          RedisConfig  `yaml:"redis"`
	Plugin         PluginConfig  `yaml:"plugin"`
}

type PluginConfig struct {
	Dir              string           `yaml:"dir"`
	ReloadInterval   time.Duration    `yaml:"reload_interval"`
	TrustedSigners   []TrustedSigner  `yaml:"trusted_signers"`
	RequireSignature bool             `yaml:"require_signature"`
}

type TrustedSigner struct {
	Name      string `yaml:"name"`
	Algorithm string `yaml:"algorithm"`
	PublicKey string `yaml:"public_key"`
}

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
