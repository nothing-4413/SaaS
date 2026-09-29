package notification

import (
	"net/smtp"
	"strconv"
	"strings"
	"testing"

	"github.com/nothing-4413/saas/internal/outbox"
)

func TestEmailSenderDeliversPasswordReset(t *testing.T) {
	called := false
	sender := EmailSender{Host: "smtp.example.com", Port: "587", Username: "u", Password: "p", From: "SaaS <no-reply@example.com>", PublicURL: "https://app.example.com", SendMail: func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		called = true
		if addr != "smtp.example.com:587" || from != "no-reply@example.com" || len(to) != 1 || !strings.Contains(string(msg), "organization_id=org-1") || !strings.Contains(string(msg), "token=reset-token") {
			t.Fatalf("unexpected message addr=%s from=%s to=%v body=%s", addr, from, to, msg)
		}
		return nil
	}}
	if err := sender.Deliver(outbox.Event{OrganizationID: "org-1", Type: "auth.password_reset_requested", Payload: []byte(`{"email":"owner@example.com","token":"reset-token"}`)}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("SMTP sender was not called")
	}
}

func TestEmailSenderIgnoresOtherEventsAndRequiresPublicURL(t *testing.T) {
	sender := EmailSender{Host: "smtp.example.com", Port: "587", From: "no-reply@example.com"}
	if err := sender.Deliver(outbox.Event{Type: "stock.low"}); err != nil {
		t.Fatal(err)
	}
	if err := sender.Deliver(outbox.Event{Type: "auth.password_reset_requested", Payload: []byte(`{"email":"owner@example.com","token":"reset-token"}`)}); err == nil {
		t.Fatal("expected public URL configuration error")
	}
}

func TestEmailSenderRejectsAddressHeaderInjection(t *testing.T) {
	sender := EmailSender{Host: "smtp.example.com", Port: "587", From: "no-reply@example.com", PublicURL: "https://app.example.com", SendMail: func(string, smtp.Auth, string, []string, []byte) error {
		t.Fatal("invalid address must not reach SMTP")
		return nil
	}}
	for _, email := range []string{"owner@example.com\r\nBcc: other@example.com", "Owner <owner@example.com>"} {
		payload := []byte(`{"email":` + strconv.Quote(email) + `,"token":"reset-token"}`)
		if err := sender.Deliver(outbox.Event{Type: "auth.password_reset_requested", Payload: payload}); err == nil {
			t.Fatalf("expected invalid address error for %q", email)
		}
	}
}
