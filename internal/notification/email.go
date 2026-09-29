package notification

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"net/url"
	"strings"
	"time"

	"github.com/nothing-4413/saas/internal/outbox"
)

var ErrEmailConfiguration = errors.New("invalid SMTP configuration")

// EmailSender delivers transactional email through an SMTP relay. It is kept
// deliberately small so deployments can use an existing mail provider.
type EmailSender struct {
	Host      string
	Port      string
	Username  string
	Password  string
	From      string
	PublicURL string
	SendMail  func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error
}

func (s EmailSender) Enabled() bool {
	return strings.TrimSpace(s.Host) != "" && strings.TrimSpace(s.Port) != "" && strings.TrimSpace(s.From) != ""
}

func (s EmailSender) Deliver(event outbox.Event) error {
	if event.Type != "auth.password_reset_requested" {
		return nil
	}
	if !s.Enabled() {
		return nil
	}
	var payload struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Token) == "" {
		return fmt.Errorf("decode password reset payload: %w", ErrEmailConfiguration)
	}
	resetURL, err := s.resetURL(event.OrganizationID, payload.Token)
	if err != nil {
		return err
	}
	body := "You requested a password reset for your inventory workspace.\r\n\r\n" +
		"Set a new password within 15 minutes:\r\n" + resetURL + "\r\n\r\n" +
		"If you did not request this, you can ignore this email."
	from, err := mail.ParseAddress(s.From)
	if err != nil {
		return fmt.Errorf("parse SMTP sender: %w", ErrEmailConfiguration)
	}
	to, err := mail.ParseAddress(payload.Email)
	if err != nil || to.Address != payload.Email || to.Name != "" || strings.ContainsAny(payload.Email, "\r\n") || strings.ContainsAny(s.From, "\r\n") {
		return fmt.Errorf("parse SMTP address: %w", ErrEmailConfiguration)
	}
	message := []byte("To: " + to.Address + "\r\n" +
		"From: " + from.String() + "\r\n" +
		"Subject: Reset your workspace password\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body)
	auth := smtp.Auth(nil)
	if strings.TrimSpace(s.Username) != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	send := s.SendMail
	if send == nil {
		send = sendMail
	}
	return send(net.JoinHostPort(s.Host, s.Port), auth, from.Address, []string{to.Address}, bytes.Clone(message))
}

func sendMail(addr string, auth smtp.Auth, from string, to []string, message []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	connection, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return err
	}
	defer connection.Close()
	_ = connection.SetDeadline(time.Now().Add(20 * time.Second))
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return err
	}
	defer client.Quit()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := client.Extension("AUTH"); !ok {
			return errors.New("SMTP server does not support authentication")
		}
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(message); err != nil {
		return err
	}
	return writer.Close()
}

func (s EmailSender) resetURL(org, token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(s.PublicURL), "/")
	if base == "" {
		return "", ErrEmailConfiguration
	}
	parsed, err := url.Parse(base + "/reset-password")
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrEmailConfiguration
	}
	query := parsed.Query()
	query.Set("organization_id", org)
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}
