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

func TestParseTokenWithSecretsSupportsRotation(t *testing.T) {
	claims := Claims{UserID: "u", OrganizationID: "o", ExpiresAt: time.Now().Add(time.Minute).Unix()}
	token, err := IssueToken("old-secret", claims)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseTokenWithSecrets([]string{"new-secret", "old-secret"}, token)
	if err != nil || got.UserID != claims.UserID {
		t.Fatalf("rotation parse failed: claims=%+v err=%v", got, err)
	}
	if _, err := ParseTokenWithSecrets([]string{"new-secret"}, token); err == nil {
		t.Fatal("removed previous secret should reject old token")
	}
}
