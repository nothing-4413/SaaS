package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginReturnsBearerToken(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	_, e := s.CreateUser(o.ID, CreateUserInput{Email: "u@example.com", Name: "U", Password: "password123"})
	if e != nil {
		t.Fatal(e)
	}
	h := NewHandler(s, "secret")
	req := httptest.NewRequest(http.MethodPost, "/organizations/"+o.ID+"/sessions", strings.NewReader(`{"email":"u@example.com","password":"password123"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "access_token") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSessionRefreshAndRevoke(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	r, _ := s.CreateRole(o.ID, CreateRoleInput{Name: "writer", Permissions: []Permission{PermissionUserWrite}})
	u, _ := s.CreateUser(o.ID, CreateUserInput{Email: "session@example.com", Name: "U", Password: "password123", RoleIDs: []string{r.ID}})
	h := NewHandler(s, "01234567890123456789012345678901")
	loginReq := httptest.NewRequest(http.MethodPost, "/organizations/"+o.ID+"/sessions", strings.NewReader(`{"email":"session@example.com","password":"password123"}`))
	loginRes := httptest.NewRecorder()
	h.ServeHTTP(loginRes, loginReq)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginRes.Code, loginRes.Body.String())
	}
	var tokens struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(loginRes.Body.Bytes(), &tokens); err != nil || tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatalf("tokens missing: %+v err=%v", tokens, err)
	}
	refreshReq := httptest.NewRequest(http.MethodPut, "/organizations/"+o.ID+"/sessions", strings.NewReader(`{"refresh_token":"`+tokens.RefreshToken+`"}`))
	refreshRes := httptest.NewRecorder()
	h.ServeHTTP(refreshRes, refreshReq)
	if refreshRes.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshRes.Code, refreshRes.Body.String())
	}
	revokeReq := httptest.NewRequest(http.MethodDelete, "/organizations/"+o.ID+"/sessions", nil)
	revokeReq.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	revokeRes := httptest.NewRecorder()
	h.ServeHTTP(revokeRes, revokeReq)
	if revokeRes.Code != http.StatusNoContent {
		t.Fatalf("revoke status=%d body=%s", revokeRes.Code, revokeRes.Body.String())
	}
	claims, err := ParseToken("01234567890123456789012345678901", tokens.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetActiveSession(claims.SessionID, o.ID, u.ID); err != ErrNotFound {
		t.Fatalf("expected revoked session, got %v", err)
	}
}

func TestInactiveUserCannotLogin(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	u, _ := s.CreateUser(o.ID, CreateUserInput{Email: "inactive@example.com", Name: "U", Password: "password123"})
	_, err := s.UpdateUser(o.ID, u.ID, UpdateUserInput{Name: "U", Active: func() *bool { v := false; return &v }()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(o.ID, u.Email, "password123"); err != ErrNotFound {
		t.Fatalf("expected inactive user rejection, got %v", err)
	}
}

func TestLoginFailureLockoutAndSuccessfulReset(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	if _, err := s.CreateUser(o.ID, CreateUserInput{Email: "lock@example.com", Name: "U", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.Authenticate(o.ID, "lock@example.com", "wrong-password"); err != ErrNotFound {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	if _, err := s.Authenticate(o.ID, "lock@example.com", "wrong-password"); err != ErrLoginLocked {
		t.Fatalf("fifth failure should lock account, got %v", err)
	}
	if _, err := s.Authenticate(o.ID, "lock@example.com", "password123"); err != ErrLoginLocked {
		t.Fatalf("locked account accepted password: %v", err)
	}
	if _, err := s.CreateUser(o.ID, CreateUserInput{Email: "reset@example.com", Name: "R", Password: "password123"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.Authenticate(o.ID, "reset@example.com", "wrong-password"); err != ErrNotFound {
			t.Fatalf("reset attempt %d: %v", i+1, err)
		}
	}
	if _, err := s.Authenticate(o.ID, "reset@example.com", "password123"); err != nil {
		t.Fatalf("valid login should clear failures: %v", err)
	}
	for i := 0; i < 4; i++ {
		if _, err := s.Authenticate(o.ID, "reset@example.com", "wrong-password"); err != ErrNotFound {
			t.Fatalf("post-reset attempt %d: %v", i+1, err)
		}
	}
}
