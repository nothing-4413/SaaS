package postgres

import (
	"context"
	"testing"
)

// Open is the only entry point the API and the integration tests use. These two
// cases pin its failure behaviour (a malformed DSN and an unreachable server)
// without needing a database, so the package is not entirely untested.
func TestOpenRejectsUnusableConnection(t *testing.T) {
	cases := map[string]string{
		"invalid dsn": "not-a-dsn",
		// Port 1 refuses connections immediately, so this stays fast.
		"unreachable server": "postgres://user@127.0.0.1:1/db?sslmode=disable",
	}
	for name, dsn := range cases {
		t.Run(name, func(t *testing.T) {
			db, err := Open(context.Background(), dsn)
			if err == nil {
				_ = db.Close()
				t.Fatalf("Open(%q) returned no error", dsn)
			}
		})
	}
}
