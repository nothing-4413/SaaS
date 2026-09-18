package idgen

import (
	"regexp"
	"testing"
)

func TestNewUUIDV4(t *testing.T) {
	a, b := New(), New()
	if a == b {
		t.Fatal("duplicate uuid")
	}
	ok, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, a)
	if !ok {
		t.Fatalf("invalid uuid: %s", a)
	}
}
