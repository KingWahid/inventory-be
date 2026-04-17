package config

import "github.com/spf13/viper"

// Config holds runtime settings for notification service.
type Config struct {
	AppEnv    string
	AppPort   string
	DBDSN     string
	RedisAddr string
}

// New reads configuration from environment variables.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("APP_PORT", "8081")
	v.SetDefault("APP_ENV", "development")

	return &Config{
		AppEnv:    v.GetString("APP_ENV"),
		AppPort:   v.GetString("APP_PORT"),
		DBDSN:     v.GetString("DB_DSN"),
		RedisAddr: v.GetString("REDIS_ADDR"),
	}, nil
}

// GetAppEnv implements pkg/common/logger.AppEnvProvider.
func (c *Config) GetAppEnv() string { return c.AppEnv }

// GetDBDSN implements infra/postgres.DBConfig.
func (c *Config) GetDBDSN() string { return c.DBDSN }

// GetRedisAddr implements infra/redis.RedisConfig.
func (c *Config) GetRedisAddr() string { return c.RedisAddr }
