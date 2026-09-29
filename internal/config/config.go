package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

const defaultAuthTokenSecret = "development-only-change-this-secret-1234"

type Config struct {
	HTTPAddr                 string
	PostgresURL              string
	AuthTokenSecret          string
	AuthTokenPreviousSecrets []string
	Environment              string
	SMTPHost                 string
	SMTPPort                 string
	SMTPUsername             string
	SMTPPassword             string
	SMTPFrom                 string
	AppPublicURL             string
}

func Load() Config {
	return Config{
		HTTPAddr:                 env("HTTP_ADDR", ":8080"),
		PostgresURL:              env("DATABASE_URL", "postgres://saas:saas@localhost:5432/saas?sslmode=disable"),
		AuthTokenSecret:          env("AUTH_TOKEN_SECRET", defaultAuthTokenSecret),
		AuthTokenPreviousSecrets: splitSecrets(os.Getenv("AUTH_TOKEN_PREVIOUS_SECRETS")),
		Environment:              strings.ToLower(env("APP_ENV", "development")),
		SMTPHost:                 strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                 strings.TrimSpace(os.Getenv("SMTP_PORT")),
		SMTPUsername:             strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:             os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:                 strings.TrimSpace(os.Getenv("SMTP_FROM")),
		AppPublicURL:             strings.TrimRight(strings.TrimSpace(env("APP_PUBLIC_URL", "http://localhost:8080")), "/"),
	}
}

func splitSecrets(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			out = append(out, value)
		}
	}
	return out
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
	for _, secret := range c.AuthTokenPreviousSecrets {
		if len(secret) < 32 {
			return errors.New("AUTH_TOKEN_PREVIOUS_SECRETS entries must contain at least 32 characters")
		}
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
