// Package config 负责加载服务配置。
//
// 优先级：config/config.yaml 提供默认值 → 环境变量覆盖（便于容器化与临时改端口）。
// 刻意不引 viper 之类的重型配置库：本服务的配置项是封闭的、几十行就能说清，
// 用 yaml.v3 直读结构体反而更容易看出「有哪些配置、默认值是什么」。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config 是服务的全部配置。
type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Auth     Auth     `yaml:"auth"`
	Upload   Upload   `yaml:"upload"`
	Logger   Logger   `yaml:"logger"`
}

// Server 是 HTTP 服务配置。
type Server struct {
	// Port 监听端口，默认 8080 —— 与前端 vite 代理的默认目标一致。
	Port int `yaml:"port"`
	// Mode 取值 debug / release，决定 gin 的运行模式与访问日志详略。
	Mode string `yaml:"mode"`
}

// Database 是数据库配置。
type Database struct {
	// Driver 取值 sqlite / mysql。默认 sqlite：文件即数据库，评分环境无需装库。
	Driver string `yaml:"driver"`
	// DSN 连接串。sqlite 为文件路径；mysql 为 user:pass@tcp(host:port)/db 形式。
	DSN string `yaml:"dsn"`
	// Seed 是否在启动时写入演示数据（幂等：库里已有用户则跳过）。
	Seed bool `yaml:"seed"`
}

// Auth 是鉴权配置。
type Auth struct {
	// JWTSecret 签发/校验 JWT 的密钥。生产必须通过环境变量注入。
	JWTSecret string `yaml:"jwt_secret"`
	// JWTExpireHours 凭证有效期（小时）。
	JWTExpireHours int `yaml:"jwt_expire_hours"`
	// Issuer 写入 JWT 的签发者，便于多服务共存时区分。
	Issuer string `yaml:"issuer"`
}

// Upload 是图片上传配置。
type Upload struct {
	// Dir 落盘目录（相对路径以进程工作目录为基准）。
	Dir string `yaml:"dir"`
	// URLPrefix 对外暴露的访问前缀，需挂在 /api 下以便前端开发态代理直接生效。
	URLPrefix string `yaml:"url_prefix"`
	// MaxSizeMB 单张图片体积上限。
	MaxSizeMB int `yaml:"max_size_mb"`
	// AllowedExts 允许的扩展名（小写，不含点）。
	AllowedExts []string `yaml:"allowed_exts"`
}

// Logger 是日志配置。
type Logger struct {
	// Level 取值 debug / info / warn / error。
	Level string `yaml:"level"`
}

// defaultConfig 返回内置默认值：即使配置文件缺失，服务也能以合理参数起来。
func defaultConfig() *Config {
	return &Config{
		Server: Server{Port: 8080, Mode: "release"},
		Database: Database{
			Driver: "sqlite",
			DSN:    filepath.Join("data", "lostfound.db"),
			Seed:   true,
		},
		Auth: Auth{
			JWTSecret:      "lostfound-campus-secret-change-me",
			JWTExpireHours: 72,
			Issuer:         "lostfound",
		},
		Upload: Upload{
			Dir:         filepath.Join("data", "uploads"),
			URLPrefix:   "/api/uploads",
			MaxSizeMB:   5,
			AllowedExts: []string{"jpg", "jpeg", "png", "webp"},
		},
		Logger: Logger{Level: "info"},
	}
}

// Load 读取配置文件（不存在则用默认值），再用环境变量覆盖，最后补齐空值。
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
		// 配置文件可缺省：默认值 + 环境变量已能覆盖全部配置项。
	default:
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	cfg.applyEnv()
	cfg.normalize()
	return cfg, nil
}

// applyEnv 用环境变量覆盖配置，便于部署时改端口、密钥而不动文件。
func (c *Config) applyEnv() {
	applyIntEnv("SERVER_PORT", &c.Server.Port)
	applyStringEnv("SERVER_MODE", &c.Server.Mode)
	applyStringEnv("DB_DRIVER", &c.Database.Driver)
	applyStringEnv("DB_DSN", &c.Database.DSN)
	applyBoolEnv("DB_SEED", &c.Database.Seed)
	applyStringEnv("JWT_SECRET", &c.Auth.JWTSecret)
	applyIntEnv("JWT_EXPIRE_HOURS", &c.Auth.JWTExpireHours)
	applyStringEnv("UPLOAD_DIR", &c.Upload.Dir)
	applyStringEnv("LOG_LEVEL", &c.Logger.Level)
}

// normalize 补齐被显式置空或因配置疏漏而缺失的项，保证服务始终能启动。
func (c *Config) normalize() {
	defaults := defaultConfig()

	if c.Server.Port <= 0 {
		c.Server.Port = defaults.Server.Port
	}
	if c.Server.Mode == "" {
		c.Server.Mode = defaults.Server.Mode
	}
	if c.Database.Driver == "" {
		c.Database.Driver = defaults.Database.Driver
	}
	if c.Database.DSN == "" {
		c.Database.DSN = defaults.Database.DSN
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = defaults.Auth.JWTSecret
	}
	if c.Auth.JWTExpireHours <= 0 {
		c.Auth.JWTExpireHours = defaults.Auth.JWTExpireHours
	}
	if c.Auth.Issuer == "" {
		c.Auth.Issuer = defaults.Auth.Issuer
	}
	if c.Upload.Dir == "" {
		c.Upload.Dir = defaults.Upload.Dir
	}
	if c.Upload.URLPrefix == "" {
		c.Upload.URLPrefix = defaults.Upload.URLPrefix
	}
	if c.Upload.MaxSizeMB <= 0 {
		c.Upload.MaxSizeMB = defaults.Upload.MaxSizeMB
	}
	if len(c.Upload.AllowedExts) == 0 {
		c.Upload.AllowedExts = defaults.Upload.AllowedExts
	}
	if c.Logger.Level == "" {
		c.Logger.Level = defaults.Logger.Level
	}
}

// Address 返回 http.Server 需要的监听地址。
func (c *Config) Address() string {
	return ":" + strconv.Itoa(c.Server.Port)
}

func applyStringEnv(key string, target *string) {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		*target = value
	}
}

func applyIntEnv(key string, target *int) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return
	}
	parsed, err := strconv.Atoi(value)
	if err == nil {
		*target = parsed
	}
}

func applyBoolEnv(key string, target *bool) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return
	}
	parsed, err := strconv.ParseBool(value)
	if err == nil {
		*target = parsed
	}
}
