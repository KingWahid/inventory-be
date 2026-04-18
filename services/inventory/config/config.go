package config

import (
	"time"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/spf13/viper"
)

// Config holds runtime settings loaded from environment (via Viper).
type Config struct {
	AppEnv               string
	AppPort              string
	DBDSN                string
	RedisAddr            string
	JWTSecret            string
	JWTAccessSecret      string
	JWTRefreshSecret     string
	JWTIssuer            string
	JWTAudience          string
	JWTAccessTTLSeconds  int
	JWTRefreshTTLSeconds int
}

// New reads configuration from the process environment.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("APP_PORT", "8080")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("JWT_SECRET", "change-me-in-production")
	v.SetDefault("JWT_ACCESS_TTL_SECONDS", 900)
	v.SetDefault("JWT_REFRESH_TTL_SECONDS", 604800)

	return &Config{
		AppEnv:               v.GetString("APP_ENV"),
		AppPort:              v.GetString("APP_PORT"),
		DBDSN:                v.GetString("DB_DSN"),
		RedisAddr:            v.GetString("REDIS_ADDR"),
		JWTSecret:            v.GetString("JWT_SECRET"),
		JWTAccessSecret:      v.GetString("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:     v.GetString("JWT_REFRESH_SECRET"),
		JWTIssuer:            v.GetString("JWT_ISSUER"),
		JWTAudience:          v.GetString("JWT_AUDIENCE"),
		JWTAccessTTLSeconds:  v.GetInt("JWT_ACCESS_TTL_SECONDS"),
		JWTRefreshTTLSeconds: v.GetInt("JWT_REFRESH_TTL_SECONDS"),
	}, nil
}

// GetAppEnv implements pkg/common/logger.AppEnvProvider.
func (c *Config) GetAppEnv() string {
	return c.AppEnv
}

// GetRedisAddr implements infra/redis.RedisConfig.
func (c *Config) GetRedisAddr() string {
	return c.RedisAddr
}

// GetDBDSN implements infra/postgres.DBConfig.
func (c *Config) GetDBDSN() string {
	return c.DBDSN
}

// NewJWTService builds the JWT verifier for inventory (same env contract as authentication-service).
func NewJWTService(cfg *Config) (*commonjwt.Service, error) {
	return commonjwt.NewServiceFromSharedOrSplit(
		cfg.JWTSecret,
		cfg.JWTAccessSecret,
		cfg.JWTRefreshSecret,
		time.Duration(cfg.JWTAccessTTLSeconds)*time.Second,
		time.Duration(cfg.JWTRefreshTTLSeconds)*time.Second,
		cfg.JWTIssuer,
		cfg.JWTAudience,
	)
}


