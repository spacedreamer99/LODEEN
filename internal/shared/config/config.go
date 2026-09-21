package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env        string
	LogLevel   string
	Repository RepositoryConfig
	Server     ServerConfig
	Postgres   PostgresConfig
	Redis      RedisConfig
	Client     ClientConfig
}

type RepositoryConfig struct {
	HTTPAddr        string
	ShutdownTimeout time.Duration
}

type ServerConfig struct {
	TCPAddr   string
	AdminAddr string
	TickRate  int
}

type PostgresConfig struct{ DSN string }

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type ClientConfig struct {
	PlanetModel string
	StartAddr   string
	Nick        string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:      getEnv("LODEEN_ENV", "dev"),
		LogLevel: getEnv("LODEEN_LOG_LEVEL", "info"),
		Repository: RepositoryConfig{
			HTTPAddr:        getEnv("LODEEN_REPOSITORY_HTTP_ADDR", ":8080"),
			ShutdownTimeout: getEnvDuration("LODEEN_REPOSITORY_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Server: ServerConfig{
			TCPAddr:   getEnv("LODEEN_SERVER_TCP_ADDR", ":7777"),
			AdminAddr: getEnv("LODEEN_SERVER_ADMIN_ADDR", ":9091"),
			TickRate:  getEnvInt("LODEEN_SERVER_TICK_RATE", 20),
		},
		Postgres: PostgresConfig{DSN: getEnv("LODEEN_POSTGRES_DSN", "")},
		Redis: RedisConfig{
			Addr:     getEnv("LODEEN_REDIS_ADDR", "localhost:6379"),
			Password: getEnv("LODEEN_REDIS_PASSWORD", ""),
			DB:       getEnvInt("LODEEN_REDIS_DB", 0),
		},
		Client: ClientConfig{
			PlanetModel: getEnv("LODEEN_CLIENT_PLANET_MODEL", "assets/models/planet.glb"),
			StartAddr:   getEnv("LODEEN_CLIENT_START_ADDR", "127.0.0.1:7777"),
			Nick:        getEnv("LODEEN_CLIENT_NICK", "pilot"),
		},
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.TickRate <= 0 || c.Server.TickRate > 1000 {
		return fmt.Errorf("LODEEN_SERVER_TICK_RATE must be in (0, 1000], got %d", c.Server.TickRate)
	}
	if c.Repository.HTTPAddr == "" {
		return fmt.Errorf("LODEEN_REPOSITORY_HTTP_ADDR must not be empty")
	}
	if c.Server.TCPAddr == "" {
		return fmt.Errorf("LODEEN_SERVER_TCP_ADDR must not be empty")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
