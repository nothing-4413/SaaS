package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr        string
	PostgresURL     string
	RedisURL        string
	AuthTokenSecret string
}

func Load() Config {
	return Config{
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		PostgresURL:     env("DATABASE_URL", "postgres://saas:saas@localhost:5432/saas?sslmode=disable"),
		RedisURL:        env("REDIS_URL", "redis://localhost:6379/0"),
		AuthTokenSecret: env("AUTH_TOKEN_SECRET", "development-only-change-me"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func IntEnv(key string, fallback int) int {
	if v, e := strconv.Atoi(os.Getenv(key)); e == nil {
		return v
	}
	return fallback
}
