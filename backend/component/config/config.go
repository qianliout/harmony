package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/goccy/go-yaml"

	"harmony/infra/consts"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
}

type ServerConfig struct {
	Port int64  `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

// Global 是全局配置，由 Load 填充。
var Global = Default()

// Default 返回默认配置：debug 模式监听 8080，使用本机 SQLite 文件。
func Default() *Config {
	return &Config{
		Server: ServerConfig{Port: 8080, Mode: consts.ServerModeDebug},
		Database: DatabaseConfig{
			Driver: consts.DBDriverSQLite,
			DSN:    "harmony.db",
		},
	}
}

// Load 解析 path 处的 YAML（path 为空则跳过），再用环境变量覆盖，结果写入 Global。
func Load(path string) error {
	cfg := Default()
	if path != "" {
		if err := loadFile(path, cfg); err != nil {
			return err
		}
	}
	applyEnv(cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}
	Global = cfg
	return nil
}

func loadFile(path string, cfg *Config) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	// 严格模式：未知键视为笔误，静默丢弃会让配置里的声明在启动后凭空失效。
	if err := yaml.UnmarshalWithOptions(content, cfg, yaml.Strict()); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}
	return nil
}

func (cfg *Config) Validate() error {
	portInRange := cfg.Server.Port > 0 && cfg.Server.Port <= 65535
	if !portInRange {
		return fmt.Errorf("server port must be in 1..65535, got %d", cfg.Server.Port)
	}
	if !consts.ValidServerMode(cfg.Server.Mode) {
		return fmt.Errorf("unsupported server mode %q", cfg.Server.Mode)
	}
	if !consts.ValidDBDriver(cfg.Database.Driver) {
		return fmt.Errorf("unsupported database driver %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN == "" {
		return fmt.Errorf("database dsn is empty")
	}
	return nil
}

func applyEnv(cfg *Config) {
	envInt("SERVER_PORT", &cfg.Server.Port)
	envStr("SERVER_MODE", &cfg.Server.Mode)
	envStr("DATABASE_DRIVER", &cfg.Database.Driver)
	envStr("DATABASE_DSN", &cfg.Database.DSN)
}

func envStr(key string, dst *string) {
	if v, ok := os.LookupEnv(key); ok {
		*dst = v
	}
}

func envInt(key string, dst *int64) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return
	}
	n, err := strconv.ParseInt(v, 10, 64)
	// 取值不是整数时保留默认值，不中断启动。
	if err != nil {
		return
	}
	*dst = n
}
