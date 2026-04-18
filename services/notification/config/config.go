package config

import "github.com/spf13/viper"

// Config holds runtime settings for notification service.
type Config struct {
	AppEnv             string
	AppPort            string
	DBDSN              string
	RedisAddr          string
	EventBusHMACSecret string

	NotifConsumerName          string
	NotifStreamConsumerEnabled bool
	NotificationWebhookURL     string
}

// New reads configuration from environment variables.
func New() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("APP_PORT", "8081")
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("NOTIF_CONSUMER_NAME", "notification-1")
	v.SetDefault("NOTIF_STREAM_CONSUMER_ENABLED", true)

	return &Config{
		AppEnv:                     v.GetString("APP_ENV"),
		AppPort:                    v.GetString("APP_PORT"),
		DBDSN:                      v.GetString("DB_DSN"),
		RedisAddr:                  v.GetString("REDIS_ADDR"),
		EventBusHMACSecret:         v.GetString("EVENTBUS_HMAC_SECRET"),
		NotifConsumerName:          v.GetString("NOTIF_CONSUMER_NAME"),
		NotifStreamConsumerEnabled: v.GetBool("NOTIF_STREAM_CONSUMER_ENABLED"),
		NotificationWebhookURL:     v.GetString("NOTIFICATION_WEBHOOK_URL"),
	}, nil
}

// StreamConsumerConfigured returns true when Redis, HMAC secret, and master enable flag allow starting the stream consumer.
func (c *Config) StreamConsumerConfigured() bool {
	if c == nil {
		return false
	}
	return c.NotifStreamConsumerEnabled && c.RedisAddr != "" && c.EventBusHMACSecret != ""
}

// GetAppEnv implements pkg/common/logger.AppEnvProvider.
func (c *Config) GetAppEnv() string { return c.AppEnv }

// GetDBDSN implements infra/postgres.DBConfig.
func (c *Config) GetDBDSN() string { return c.DBDSN }

// GetRedisAddr implements infra/redis.RedisConfig.
func (c *Config) GetRedisAddr() string { return c.RedisAddr }
