package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestUserCannotUseRoleFromAnotherOrganization(t *testing.T) {
	s := NewService(NewMemoryStore())
	a, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	b, _ := s.CreateOrganization(CreateOrganizationInput{Name: "B"})
	r, _ := s.CreateRole(a.ID, CreateRoleInput{Name: "admin"})
	if _, err := s.CreateUser(b.ID, CreateUserInput{Email: "u@example.com", Name: "U", Password: "password123", RoleIDs: []string{r.ID}}); err != ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestEmailNormalizedAndDuplicateRejected(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	u, err := s.CreateUser(o.ID, CreateUserInput{Email: " U@EXAMPLE.COM ", Name: "U", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "u@example.com" {
		t.Fatalf("email not normalized: %q", u.Email)
	}
	if _, err := s.CreateUser(o.ID, CreateUserInput{Email: "u@example.com", Name: "Other", Password: "password123"}); err != ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestHasPermissionIsTenantScoped(t *testing.T) {
	s := NewService(NewMemoryStore())
	a, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	b, _ := s.CreateOrganization(CreateOrganizationInput{Name: "B"})
	r, _ := s.CreateRole(a.ID, CreateRoleInput{Name: "admin", Permissions: []Permission{PermissionUserWrite}})
	u, _ := s.CreateUser(a.ID, CreateUserInput{Email: "u@example.com", Name: "U", Password: "password123", RoleIDs: []string{r.ID}})
	ok, err := s.HasPermission(a.ID, u.ID, PermissionUserWrite)
	if err != nil || !ok {
		t.Fatalf("expected permission, ok=%v err=%v", ok, err)
	}
	if _, err = s.HasPermission(b.ID, u.ID, PermissionUserWrite); err != ErrNotFound {
		t.Fatalf("expected tenant isolation, got %v", err)
	}
}

func TestCreateOrganizationBootstrapsOwner(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, err := s.CreateOrganization(CreateOrganizationInput{Name: "Acme", OwnerEmail: " OWNER@EXAMPLE.COM ", OwnerName: "Owner", OwnerPassword: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.Authenticate(o.ID, "owner@example.com", "password123")
	if err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte("password123")) != nil {
		t.Fatal("owner password was not hashed")
	}
	for _, permission := range AllPermissions {
		allowed, err := s.HasPermission(o.ID, u.ID, permission)
		if err != nil || !allowed {
			t.Fatalf("permission %q missing: allowed=%v err=%v", permission, allowed, err)
		}
	}
}

func TestCreateOrganizationRejectsPartialOwner(t *testing.T) {
	s := NewService(NewMemoryStore())
	if _, err := s.CreateOrganization(CreateOrganizationInput{Name: "Acme", OwnerName: "Owner"}); err != ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
