// Package config 加载 agent-hub 的运行配置。
package config

import (
	"github.com/spf13/viper"
)

// Config 是所有配置的根结构。
type Config struct {
	Port string `mapstructure:"HUB_PORT"`
	Host string `mapstructure:"HUB_HOST"`
	Env  string `mapstructure:"HUB_ENV"`

	DatabaseURL string `mapstructure:"HUB_DATABASE_URL"`
	RedisURL    string `mapstructure:"HUB_REDIS_URL"`

	JWTSecret string `mapstructure:"HUB_JWT_SECRET"`

	LogLevel string `mapstructure:"HUB_LOG_LEVEL"`
	LogPath  string `mapstructure:"HUB_LOG_PATH"`

	LockDefaultTTL  int `mapstructure:"HUB_LOCK_DEFAULT_TTL_SECONDS"`
	LockCleanupInteval int `mapstructure:"HUB_LOCK_CLEANUP_INTERVAL_SECONDS"`
	LockMaxTTL      int `mapstructure:"HUB_LOCK_MAX_TTL_SECONDS"`

	CORSOrigins string `mapstructure:"HUB_CORS_ORIGINS"`
}

// Load 从 .env / 环境变量加载配置。
func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// .env 不存在也行，靠环境变量
		_ = err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
