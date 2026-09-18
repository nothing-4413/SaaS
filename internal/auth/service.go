package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/nothing-4413/saas/internal/platform/idgen"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }
func (s *Service) id() string         { return idgen.New() }

func (s *Service) CreateOrganization(in CreateOrganizationInput) (Organization, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Organization{}, ErrInvalidInput
	}
	v := Organization{ID: s.id(), Name: name, CreatedAt: s.now().UTC()}
	email := strings.ToLower(strings.TrimSpace(in.OwnerEmail))
	ownerName := strings.TrimSpace(in.OwnerName)
	hasOwnerFields := email != "" || ownerName != "" || in.OwnerPassword != ""
	if hasOwnerFields {
		if email == "" || ownerName == "" || len(in.OwnerPassword) < 8 {
			return Organization{}, ErrInvalidInput
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.OwnerPassword), bcrypt.DefaultCost)
		if err != nil {
			return Organization{}, err
		}
		role := Role{ID: s.id(), OrganizationID: v.ID, Name: "owner", Permissions: append([]Permission(nil), AllPermissions...)}
		user := User{ID: s.id(), OrganizationID: v.ID, Email: email, Name: ownerName, PasswordHash: string(hash), RoleIDs: []string{role.ID}, CreatedAt: v.CreatedAt}
		store, ok := s.store.(BootstrapStore)
		if !ok {
			return Organization{}, ErrInvalidInput
		}
		return v, store.CreateOrganizationOwner(v, role, user)
	}
	return v, s.store.CreateOrganization(v)
}
func (s *Service) CreateRole(orgID string, in CreateRoleInput) (Role, error) {
	if _, err := s.store.GetOrganization(orgID); err != nil {
		return Role{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Role{}, ErrInvalidInput
	}
	v := Role{ID: s.id(), OrganizationID: orgID, Name: name, Permissions: append([]Permission(nil), in.Permissions...)}
	return v, s.store.CreateRole(v)
}
func (s *Service) CreateUser(orgID string, in CreateUserInput) (User, error) {
	if _, err := s.store.GetOrganization(orgID); err != nil {
		return User{}, err
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	name := strings.TrimSpace(in.Name)
	if email == "" || name == "" || len(in.Password) < 8 {
		return User{}, ErrInvalidInput
	}
	for _, roleID := range in.RoleIDs {
		r, err := s.store.GetRole(roleID)
		if err != nil || r.OrganizationID != orgID {
			return User{}, ErrInvalidInput
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}
	v := User{ID: s.id(), OrganizationID: orgID, Email: email, Name: name, PasswordHash: string(hash), RoleIDs: append([]string(nil), in.RoleIDs...), CreatedAt: s.now().UTC()}
	return v, s.store.CreateUser(v)
}
func (s *Service) Authenticate(org, email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if org == "" || email == "" || password == "" {
		return User{}, ErrInvalidInput
	}
	u, err := s.store.FindUserByEmail(org, email)
	if err != nil {
		return User{}, ErrNotFound
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return User{}, ErrNotFound
	}
	return u, nil
}
func (s *Service) ListUsers(orgID string) []User { return s.store.ListUsers(orgID) }
func (s *Service) ListRoles(orgID string) []Role { return s.store.ListRoles(orgID) }

func (s *Service) HasPermission(orgID, userID string, permission Permission) (bool, error) {
	if strings.TrimSpace(orgID) == "" || strings.TrimSpace(userID) == "" || permission == "" {
		return false, ErrInvalidInput
	}
	u, err := s.store.GetUser(userID)
	if err != nil {
		return false, err
	}
	if u.OrganizationID != orgID {
		return false, ErrNotFound
	}
	for _, roleID := range u.RoleIDs {
		r, err := s.store.GetRole(roleID)
		if err != nil {
			continue
		}
		if r.OrganizationID != orgID {
			continue
		}
		for _, p := range r.Permissions {
			if p == permission {
				return true, nil
			}
		}
	}
	return false, nil
}
