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
