package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequirePermission(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	r, _ := s.CreateRole(o.ID, CreateRoleInput{Name: "writer", Permissions: []Permission{PermissionUserWrite}})
	u, _ := s.CreateUser(o.ID, CreateUserInput{Email: "u@x.com", Name: "U", RoleIDs: []string{r.ID}})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	h := RequirePermission(s, PermissionUserWrite, next)
	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 401 {
		t.Fatalf("missing headers=%d", res.Code)
	}
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Organization-ID", o.ID)
	req.Header.Set("X-User-ID", u.ID)
	res = httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != 204 {
		t.Fatalf("allowed=%d", res.Code)
	}
}
