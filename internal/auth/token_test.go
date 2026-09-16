package auth

import (
	"testing"
	"time"
)

func TestTokenRoundTripAndExpiry(t *testing.T) {
	tok, e := IssueToken("secret", Claims{UserID: "u", OrganizationID: "o", ExpiresAt: time.Now().Add(time.Minute).Unix()})
	if e != nil {
		t.Fatal(e)
	}
	c, e := ParseToken("secret", tok)
	if e != nil || c.UserID != "u" {
		t.Fatalf("claims=%+v err=%v", c, e)
	}
	if _, e = ParseToken("wrong", tok); e == nil {
		t.Fatal("expected signature failure")
	}
	expired, _ := IssueToken("secret", Claims{UserID: "u", OrganizationID: "o", ExpiresAt: time.Now().Add(-time.Minute).Unix()})
	if _, e = ParseToken("secret", expired); e == nil {
		t.Fatal("expected expiry failure")
	}
}
