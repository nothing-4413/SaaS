package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	ExpiresAt      int64  `json:"exp"`
}

func IssueToken(secret string, claims Claims) (string, error) {
	if strings.TrimSpace(secret) == "" || claims.UserID == "" || claims.OrganizationID == "" || claims.ExpiresAt <= 0 {
		return "", ErrInvalidToken
	}
	b, e := json.Marshal(claims)
	if e != nil {
		return "", e
	}
	enc := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(enc))
	return enc + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func ParseToken(secret, token string) (Claims, error) {
	if strings.TrimSpace(secret) == "" {
		return Claims{}, ErrInvalidToken
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	sig, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return Claims{}, ErrInvalidToken
	}
	b, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return Claims{}, ErrInvalidToken
	}
	var c Claims
	if json.Unmarshal(b, &c) != nil || c.UserID == "" || c.OrganizationID == "" || time.Now().Unix() >= c.ExpiresAt {
		return Claims{}, ErrInvalidToken
	}
	return c, nil
}
