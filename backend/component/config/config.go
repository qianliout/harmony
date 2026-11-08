package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/goccy/go-yaml"

	"harmony/infra/consts"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Storage   StorageConfig   `yaml:"storage"`
	Recognize RecognizeConfig `yaml:"recognize"`
}

type ServerConfig struct {
	Port int64  `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type StorageConfig struct {
	UploadDir string `yaml:"uploadDir"`
}

type RecognizeConfig struct {
	Provider       string  `yaml:"provider"`
	BaseURL        string  `yaml:"baseURL"`
	APIKey         string  `yaml:"apiKey"`
	Model          string  `yaml:"model"`
	TimeoutSeconds int64   `yaml:"timeoutSeconds"`
	Temperature    float64 `yaml:"temperature"`
}

// Global 是全局配置，由 Load 填充。
var Global = Default()

// Default 返回默认配置：debug 模式监听 8080，连接本机 MySQL 的 harmony 库（OrbStack 容器 mysql，root/root）。
func Default() *Config {
	return &Config{
		Server: ServerConfig{Port: 8080, Mode: consts.ServerModeDebug},
		Database: DatabaseConfig{
			Driver: consts.DBDriverMySQL,
			DSN:    "root:root@tcp(127.0.0.1:3306)/harmony?charset=utf8mb4&parseTime=True&loc=Local",
		},
		Storage: StorageConfig{UploadDir: "data/uploads"},
		Recognize: RecognizeConfig{
			Provider:       "deepseek",
			BaseURL:        "https://api.deepseek.com/v1",
			Model:          "deepseek-chat",
			TimeoutSeconds: 60,
			Temperature:    0,
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
	if cfg.Storage.UploadDir == "" {
		return fmt.Errorf("storage uploadDir is empty")
	}
	if cfg.Recognize.TimeoutSeconds <= 0 {
		return fmt.Errorf("recognize timeoutSeconds must be positive")
	}
	return nil
}

func applyEnv(cfg *Config) {
	envInt("SERVER_PORT", &cfg.Server.Port)
	envStr("SERVER_MODE", &cfg.Server.Mode)
	envStr("DATABASE_DRIVER", &cfg.Database.Driver)
	envStr("DATABASE_DSN", &cfg.Database.DSN)
	envStr("HARMONY_STORAGE_UPLOAD_DIR", &cfg.Storage.UploadDir)
	envStr("HARMONY_RECOGNIZE_PROVIDER", &cfg.Recognize.Provider)
	envStr("HARMONY_RECOGNIZE_BASE_URL", &cfg.Recognize.BaseURL)
	envStr("HARMONY_RECOGNIZE_MODEL", &cfg.Recognize.Model)
	envStr("HARMONY_RECOGNIZE_API_KEY", &cfg.Recognize.APIKey)
	// 本机已有 DEEPSEEK_API_KEY 时直接复用，避免额外维护一份密钥。
	if cfg.Recognize.APIKey == "" {
		envStr("DEEPSEEK_API_KEY", &cfg.Recognize.APIKey)
	}
	envInt("HARMONY_RECOGNIZE_TIMEOUT_SECONDS", &cfg.Recognize.TimeoutSeconds)
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
