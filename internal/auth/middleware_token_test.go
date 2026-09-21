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

func TestRequireTokenPermissionRejectsDifferentURLTenant(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	r, _ := s.CreateRole(o.ID, CreateRoleInput{Name: "writer", Permissions: []Permission{PermissionUserWrite}})
	u, _ := s.CreateUser(o.ID, CreateUserInput{Email: "token@x.com", Name: "U", Password: "password123", RoleIDs: []string{r.ID}})
	tok, _ := IssueToken("secret", Claims{UserID: u.ID, OrganizationID: o.ID, ExpiresAt: time.Now().Add(time.Minute).Unix()})
	h := RequireTokenPermission(s, "secret", PermissionUserWrite, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be called")
	}))
	req := httptest.NewRequest(http.MethodGet, "/organizations/different/users", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status=%d", res.Code)
	}
}

func TestRequireTokenPermissionResolverUsesRequest(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	r, _ := s.CreateRole(o.ID, CreateRoleInput{Name: "reader", Permissions: []Permission{PermissionProductRead}})
	u, _ := s.CreateUser(o.ID, CreateUserInput{Email: "reader@x.com", Name: "U", Password: "password123", RoleIDs: []string{r.ID}})
	tok, _ := IssueToken("secret", Claims{UserID: u.ID, OrganizationID: o.ID, ExpiresAt: time.Now().Add(time.Minute).Unix()})
	h := RequireTokenPermissionResolver(s, []string{"secret"}, func(r *http.Request) Permission {
		if r.Method == http.MethodGet {
			return PermissionProductRead
		}
		return PermissionProductWrite
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	get := httptest.NewRequest(http.MethodGet, "/organizations/"+o.ID+"/products", nil)
	get.Header.Set("Authorization", "Bearer "+tok)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, get)
	if res.Code != http.StatusNoContent {
		t.Fatalf("read status=%d", res.Code)
	}
	post := httptest.NewRequest(http.MethodPost, "/organizations/"+o.ID+"/products", nil)
	post.Header.Set("Authorization", "Bearer "+tok)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, post)
	if res.Code != http.StatusForbidden {
		t.Fatalf("write status=%d", res.Code)
	}
}
