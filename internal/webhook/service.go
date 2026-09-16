package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
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
type Sender struct{ Client *http.Client }

func Sign(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
func (s Sender) Send(d Delivery) error {
	if strings.TrimSpace(d.URL) == "" || strings.TrimSpace(d.Secret) == "" || strings.TrimSpace(d.EventType) == "" || strings.TrimSpace(d.IdempotencyKey) == "" || len(d.Payload) == 0 {
		return ErrInvalidInput
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
	_, _ = ioutil.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
