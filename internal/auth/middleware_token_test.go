package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequireTokenPermissionInjectsClaims(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	r, _ := s.CreateRole(o.ID, CreateRoleInput{Name: "writer", Permissions: []Permission{PermissionUserWrite}})
	u, _ := s.CreateUser(o.ID, CreateUserInput{Email: "token@x.com", Name: "U", Password: "password123", RoleIDs: []string{r.ID}})
	tok, _ := IssueToken("secret", Claims{UserID: u.ID, OrganizationID: o.ID, ExpiresAt: time.Now().Add(time.Minute).Unix()})
	h := RequireTokenPermission(s, "secret", PermissionUserWrite, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, ok := ClaimsFromContext(r.Context()); !ok || c.UserID != u.ID {
			t.Fatal("claims missing")
		}
		w.WriteHeader(204)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 204 {
		t.Fatalf("status=%d", res.Code)
	}
}
