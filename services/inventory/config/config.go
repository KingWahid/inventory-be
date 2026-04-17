package config

import (
	"github.com/spf13/viper"
)

// Config holds runtime settings loaded from environment (via Viper).
type Config struct {
	AppEnv    string
	AppPort   string
	DBDSN     string
	RedisAddr string
	JWTSecret string
}

// New reads configuration from the process environment.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("APP_ENV", "development")

	return &Config{
		AppEnv:    v.GetString("APP_ENV"),
		AppPort:   v.GetString("APP_PORT"),
		DBDSN:     v.GetString("DB_DSN"),
		RedisAddr: v.GetString("REDIS_ADDR"),
		JWTSecret: v.GetString("JWT_SECRET"),
	}, nil
}
