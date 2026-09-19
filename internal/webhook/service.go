package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("invalid webhook input")

type Delivery struct {
	ID             string
	URL            string
	Secret         string
	EventType      string
	Payload        []byte
	IdempotencyKey string
}
type Sender struct {
	Client       *http.Client
	AllowPrivate bool
	Resolver     *net.Resolver
}

func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
func (s Sender) Send(d Delivery) error {
	if strings.TrimSpace(d.URL) == "" || strings.TrimSpace(d.Secret) == "" || strings.TrimSpace(d.EventType) == "" || strings.TrimSpace(d.IdempotencyKey) == "" || len(d.Payload) == 0 {
		return ErrInvalidInput
	}
	if err := s.validateTarget(d.URL); err != nil {
		return err
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	req, e := http.NewRequest(http.MethodPost, d.URL, bytes.NewReader(d.Payload))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Event", d.EventType)
	req.Header.Set("X-Webhook-Id", d.ID)
	req.Header.Set("X-Idempotency-Key", d.IdempotencyKey)
	req.Header.Set("X-Webhook-Signature", "sha256="+Sign(d.Secret, d.Payload))
	resp, e := client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func (s Sender) validateTarget(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		return ErrInvalidInput
	}
	if s.AllowPrivate {
		return nil
	}
	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return errors.New("webhook target must not be localhost")
	}
	if ip := net.ParseIP(host); ip != nil {
		if isPrivateAddress(ip) {
			return errors.New("webhook target must use a public address")
		}
		return nil
	}
	resolver := s.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	addresses, err := resolver.LookupIPAddr(ctx, host)
	if err != nil || len(addresses) == 0 {
		return fmt.Errorf("resolve webhook target: %w", err)
	}
	for _, address := range addresses {
		if isPrivateAddress(address.IP) {
			return errors.New("webhook target resolves to a private address")
		}
	}
	return nil
}

func isPrivateAddress(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}
