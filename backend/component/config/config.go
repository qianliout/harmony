package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"

	"harmony/infra/consts"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Database  DatabaseConfig  `yaml:"database"`
	Storage   StorageConfig   `yaml:"storage"`
	Recognize RecognizeConfig `yaml:"recognize"`
	Auth      AuthConfig      `yaml:"auth"`
	Wechat    WechatConfig    `yaml:"wechat"`
	Backup    BackupConfig    `yaml:"backup"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int64  `yaml:"port"`
	Mode string `yaml:"mode"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
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
	HighResolution bool    `yaml:"highResolution"`
	DailyLimit     int64   `yaml:"dailyLimit"`
}

type AuthConfig struct {
	AdminUserIDs []uint64 `yaml:"adminUserIds"`
	MPTokenDays  int64    `yaml:"mpTokenDays"`
	WebTokenDays int64    `yaml:"webTokenDays"`
	Secret       string   `yaml:"secret"`
}

type WechatConfig struct {
	AppID      string `yaml:"appID"`
	AppSecret  string `yaml:"appSecret"`
	EnvVersion string `yaml:"envVersion"`
}

type BackupConfig struct {
	Dir      string `yaml:"dir"`
	KeepDays int    `yaml:"keepDays"`
}

// Global 是全局配置，由 Load 填充。
var Global = Default()

// Default 返回默认配置：本机开发只监听 127.0.0.1:8080，数据存在 data/harmony.db。
func Default() *Config {
	return &Config{
		Server:   ServerConfig{Host: "127.0.0.1", Port: 8080, Mode: consts.ServerModeDebug},
		Database: DatabaseConfig{Path: "data/harmony.db"},
		Storage:  StorageConfig{UploadDir: "data/uploads"},
		Recognize: RecognizeConfig{
			Provider:       "qwen",
			BaseURL:        "https://dashscope.aliyuncs.com/compatible-mode/v1",
			Model:          "qwen3-vl-plus",
			TimeoutSeconds: 90,
			Temperature:    0,
			HighResolution: true,
			DailyLimit:     100,
		},
		Auth: AuthConfig{
			AdminUserIDs: []uint64{1},
			MPTokenDays:  30,
			WebTokenDays: 14,
		},
		Wechat: WechatConfig{
			AppID:      "wx348e5ad03f9cd6ba",
			EnvVersion: "develop",
		},
		Backup: BackupConfig{Dir: "data/backups", KeepDays: 7},
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
	if strings.TrimSpace(cfg.Server.Host) == "" {
		return fmt.Errorf("server host is empty")
	}
	portInRange := cfg.Server.Port > 0 && cfg.Server.Port <= 65535
	if !portInRange {
		return fmt.Errorf("server port must be in 1..65535, got %d", cfg.Server.Port)
	}
	if !consts.ValidServerMode(cfg.Server.Mode) {
		return fmt.Errorf("unsupported server mode %q", cfg.Server.Mode)
	}
	if strings.TrimSpace(cfg.Database.Path) == "" {
		return fmt.Errorf("database path is empty")
	}
	if cfg.Storage.UploadDir == "" {
		return fmt.Errorf("storage uploadDir is empty")
	}
	if cfg.Recognize.TimeoutSeconds <= 0 {
		return fmt.Errorf("recognize timeoutSeconds must be positive")
	}
	if cfg.Recognize.DailyLimit < 0 {
		return fmt.Errorf("recognize dailyLimit must be non-negative")
	}
	if cfg.Auth.MPTokenDays <= 0 || cfg.Auth.WebTokenDays <= 0 {
		return fmt.Errorf("auth token days must be positive")
	}
	if len(cfg.Auth.AdminUserIDs) == 0 {
		cfg.Auth.AdminUserIDs = []uint64{1}
	}
	if cfg.Backup.KeepDays < 0 {
		return fmt.Errorf("backup keepDays must be non-negative")
	}
	return nil
}

func applyEnv(cfg *Config) {
	envStr("SERVER_HOST", &cfg.Server.Host)
	envInt("SERVER_PORT", &cfg.Server.Port)
	envStr("SERVER_MODE", &cfg.Server.Mode)
	envStr("HARMONY_DATABASE_PATH", &cfg.Database.Path)
	envStr("HARMONY_STORAGE_UPLOAD_DIR", &cfg.Storage.UploadDir)
	envStr("HARMONY_RECOGNIZE_PROVIDER", &cfg.Recognize.Provider)
	envStr("HARMONY_RECOGNIZE_BASE_URL", &cfg.Recognize.BaseURL)
	envStr("HARMONY_RECOGNIZE_MODEL", &cfg.Recognize.Model)
	envStr("HARMONY_RECOGNIZE_API_KEY", &cfg.Recognize.APIKey)
	if cfg.Recognize.APIKey == "" {
		envStr("DASHSCOPE_API_KEY", &cfg.Recognize.APIKey)
	}
	if cfg.Recognize.APIKey == "" {
		envStr("ALI_BAILIAN_API_KEY", &cfg.Recognize.APIKey)
	}
	envInt("HARMONY_RECOGNIZE_TIMEOUT_SECONDS", &cfg.Recognize.TimeoutSeconds)
	envStr("HARMONY_AUTH_SECRET", &cfg.Auth.Secret)
	envStr("HARMONY_WECHAT_APP_ID", &cfg.Wechat.AppID)
	envStr("HARMONY_WECHAT_APP_SECRET", &cfg.Wechat.AppSecret)
	envStr("HARMONY_WECHAT_ENV_VERSION", &cfg.Wechat.EnvVersion)
	envStr("HARMONY_BACKUP_DIR", &cfg.Backup.Dir)
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
