package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

type DatabaseConfig struct {
	URL string
}

type RedisConfig struct {
	URL string
}

type AuthConfig struct {
	ClerkSecretKey string
	ClerkPublicKey string
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./internal/config")
	viper.AutomaticEnv()

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", 30)
	viper.SetDefault("server.write_timeout", 30)
	viper.SetDefault("server.idle_timeout", 60)

	viper.SetDefault("database.url", "postgres://deployguard:deployguard@localhost:5432/deployguard?sslmode=disable")
	viper.SetDefault("redis.url", "redis://localhost:6379/0")

	// Bind environment variables
	viper.BindEnv("database.url", "DATABASE_URL")
	viper.BindEnv("redis.url", "REDIS_URL")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: config file not found, using defaults: %v", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         viper.GetInt("server.port"),
			ReadTimeout:  viper.GetInt("server.read_timeout"),
			WriteTimeout: viper.GetInt("server.write_timeout"),
			IdleTimeout:  viper.GetInt("server.idle_timeout"),
		},
		Database: DatabaseConfig{
			URL: viper.GetString("database.url"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("redis.url"),
		},
		Auth: AuthConfig{
			ClerkSecretKey: viper.GetString("auth.clerk_secret_key"),
			ClerkPublicKey: viper.GetString("auth.clerk_public_key"),
		},
	}

	return cfg
}

func (c *DatabaseConfig) DSN() string {
	return c.URL
}
