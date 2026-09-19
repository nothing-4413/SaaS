package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const defaultAuthTokenSecret = "development-only-change-me"

type Config struct {
	HTTPAddr        string
	PostgresURL     string
	RedisURL        string
	AuthTokenSecret string
	Environment     string
}

func Load() Config {
	return Config{
		HTTPAddr:        env("HTTP_ADDR", ":8080"),
		PostgresURL:     env("DATABASE_URL", "postgres://saas:saas@localhost:5432/saas?sslmode=disable"),
		RedisURL:        env("REDIS_URL", "redis://localhost:6379/0"),
		AuthTokenSecret: env("AUTH_TOKEN_SECRET", defaultAuthTokenSecret),
		Environment:     strings.ToLower(env("APP_ENV", "development")),
	}
}

func (c Config) ValidateAPI() error {
	if strings.TrimSpace(c.PostgresURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if len(c.AuthTokenSecret) < 32 {
		return errors.New("AUTH_TOKEN_SECRET must contain at least 32 characters")
	}
	if c.Environment == "production" && c.AuthTokenSecret == defaultAuthTokenSecret {
		return errors.New("AUTH_TOKEN_SECRET must be changed in production")
	}
	return nil
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
