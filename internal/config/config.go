package config

import "os"

type Config struct {
	DSN       string
	JWTSecret string
	Port      string
}

func Load() *Config {
	return &Config{
		DSN:       getEnv("DB_DSN", "root:@tcp(127.0.0.1:3306)/lost_found?charset=utf8mb4&parseTime=True&loc=Local"),
		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-me"),
		Port:      getEnv("PORT", "8080"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
