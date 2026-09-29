package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	c := Load()
	if c.HTTPAddr == "" || c.PostgresURL == "" {
		t.Fatalf("defaults missing: %+v", c)
	}
}

func TestValidateAPIRejectsWeakSecret(t *testing.T) {
	c := Config{PostgresURL: "postgres://example", AuthTokenSecret: "short", Environment: "production"}
	if err := c.ValidateAPI(); err == nil {
		t.Fatal("expected weak secret error")
	}
}

func TestValidateAPIAcceptsStrongSecret(t *testing.T) {
	c := Config{PostgresURL: "postgres://example", AuthTokenSecret: "0123456789abcdef0123456789abcdef", Environment: "production"}
	if err := c.ValidateAPI(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadIncludesMailAndPublicURLConfiguration(t *testing.T) {
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_FROM", "no-reply@example.com")
	t.Setenv("APP_PUBLIC_URL", "https://app.example.com/")
	c := Load()
	if c.SMTPHost != "smtp.example.com" || c.SMTPPort != "587" || c.SMTPFrom != "no-reply@example.com" || c.AppPublicURL != "https://app.example.com" {
		t.Fatalf("unexpected notification config: %+v", c)
	}
}
