package auth

import "testing"

func TestUserCannotUseRoleFromAnotherOrganization(t *testing.T) {
	s := NewService(NewMemoryStore())
	a, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	b, _ := s.CreateOrganization(CreateOrganizationInput{Name: "B"})
	r, _ := s.CreateRole(a.ID, CreateRoleInput{Name: "admin"})
	if _, err := s.CreateUser(b.ID, CreateUserInput{Email: "u@example.com", Name: "U", RoleIDs: []string{r.ID}}); err != ErrInvalidInput {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestEmailNormalizedAndDuplicateRejected(t *testing.T) {
	s := NewService(NewMemoryStore())
	o, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	u, err := s.CreateUser(o.ID, CreateUserInput{Email: " U@EXAMPLE.COM ", Name: "U"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "u@example.com" {
		t.Fatalf("email not normalized: %q", u.Email)
	}
	if _, err := s.CreateUser(o.ID, CreateUserInput{Email: "u@example.com", Name: "Other"}); err != ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestHasPermissionIsTenantScoped(t *testing.T) {
	s := NewService(NewMemoryStore())
	a, _ := s.CreateOrganization(CreateOrganizationInput{Name: "A"})
	b, _ := s.CreateOrganization(CreateOrganizationInput{Name: "B"})
	r, _ := s.CreateRole(a.ID, CreateRoleInput{Name: "admin", Permissions: []Permission{PermissionUserWrite}})
	u, _ := s.CreateUser(a.ID, CreateUserInput{Email: "u@example.com", Name: "U", RoleIDs: []string{r.ID}})
	ok, err := s.HasPermission(a.ID, u.ID, PermissionUserWrite)
	if err != nil || !ok {
		t.Fatalf("expected permission, ok=%v err=%v", ok, err)
	}
	if _, err = s.HasPermission(b.ID, u.ID, PermissionUserWrite); err != ErrNotFound {
		t.Fatalf("expected tenant isolation, got %v", err)
	}
}
