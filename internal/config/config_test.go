package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	c := Load()
	if c.HTTPAddr == "" || c.PostgresURL == "" || c.RedisURL == "" {
		t.Fatalf("defaults missing: %+v", c)
	}
}
