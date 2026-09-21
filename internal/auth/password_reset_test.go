package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPasswordResetIsTenantScopedOneTimeAndRevokesSessions(t *testing.T) {
	store := NewMemoryStore()
	s := NewService(store)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s.now = func() time.Time { return now }
	a, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	b, _ := s.CreateOrganization(CreateOrganizationInput{Name: "B"})
	u, err := s.CreateUser(a.ID, CreateUserInput{Email: "reset@example.com", Name: "U", Password: "oldpassword"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.CreateSession(u, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.RequestPasswordReset(a.ID, u.Email)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(b.ID, token, "newpassword"); err != ErrInvalidResetToken {
		t.Fatalf("cross-tenant reset should fail, got %v", err)
	}
	if err := s.ResetPassword(a.ID, token, "newpassword"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(a.ID, u.Email, "oldpassword"); err == nil {
		t.Fatal("old password still authenticates")
	}
	if _, err := s.Authenticate(a.ID, u.Email, "newpassword"); err != nil {
		t.Fatalf("new password rejected: %v", err)
	}
	if _, err := s.GetActiveSession(session.ID, a.ID, u.ID); err != ErrNotFound {
		t.Fatalf("session should be revoked, got %v", err)
	}
	if err := s.ResetPassword(a.ID, token, "thirdpassword"); err != ErrInvalidResetToken {
		t.Fatalf("token should be one-time, got %v", err)
	}

	_ = b
}

func TestPasswordResetTokenExpires(t *testing.T) {
	store := NewMemoryStore()
	s := NewService(store)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	s.now = func() time.Time { return now }
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	_, _ = s.CreateUser(o.ID, CreateUserInput{Email: "expired@example.com", Name: "U", Password: "oldpassword"})
	token, err := s.RequestPasswordReset(o.ID, "expired@example.com")
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(16 * time.Minute)
	if err := s.ResetPassword(o.ID, token, "newpassword"); err != ErrInvalidResetToken {
		t.Fatalf("expired token accepted: %v", err)
	}
}

func TestPasswordResetRequestDoesNotEnumerateAccounts(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	_, _ = s.CreateUser(o.ID, CreateUserInput{Email: "known@example.com", Name: "U", Password: "oldpassword"})
	h := NewHandler(s)
	for _, email := range []string{"known@example.com", "unknown@example.com"} {
		req := httptest.NewRequest(http.MethodPost, "/organizations/"+o.ID+"/sessions/password-reset", strings.NewReader(`{"email":"`+email+`"}`))
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != http.StatusAccepted {
			t.Fatalf("email %s status=%d body=%s", email, res.Code, res.Body.String())
		}
		var body map[string]string
		if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil || body["message"] == "" {
			t.Fatalf("missing generic response: %s", res.Body.String())
		}
	}
}
