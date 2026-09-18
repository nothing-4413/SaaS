package auth

import (
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
