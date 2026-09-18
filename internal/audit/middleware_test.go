package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nothing-4413/saas/internal/auth"
)

func TestMiddlewareRecordsAuthenticatedMutation(t *testing.T) {
	authService := auth.NewService(auth.NewMemoryStore())
	org, _ := authService.CreateOrganization(auth.CreateOrganizationInput{Name: "A"})
	role, _ := authService.CreateRole(org.ID, auth.CreateRoleInput{Name: "writer", Permissions: []auth.Permission{auth.PermissionOrderManage}})
	user, _ := authService.CreateUser(org.ID, auth.CreateUserInput{Email: "u@example.com", Name: "U", Password: "password123", RoleIDs: []string{role.ID}})
	token, _ := auth.IssueToken("secret", auth.Claims{UserID: user.ID, OrganizationID: org.ID, ExpiresAt: time.Now().Add(time.Minute).Unix()})
	store := NewMemoryStore()
	handler := auth.RequireTokenPermission(authService, "secret", auth.PermissionOrderManage, Middleware(NewService(store), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})))
	req := httptest.NewRequest(http.MethodPost, "/organizations/"+org.ID+"/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Request-ID", "req-1")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	entries := store.List(org.ID)
	if len(entries) != 1 || entries[0].ActorUserID != user.ID || entries[0].Action != http.MethodPost {
		t.Fatalf("entries=%+v", entries)
	}
	if entries[0].Metadata["status"] != http.StatusCreated {
		t.Fatalf("metadata=%+v", entries[0].Metadata)
	}
}

func TestMiddlewareDoesNotRecordReads(t *testing.T) {
	store := NewMemoryStore()
	handler := Middleware(NewService(store), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/organizations/o/orders", nil))
	if len(store.List("o")) != 0 {
		t.Fatal("read request was audited as mutation")
	}
}
