package auth

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	store  Store
	now    func() time.Time
	nextID uint64
}

func NewService(store Store) *Service { return &Service{store: store, now: time.Now} }
func (s *Service) id() string {
	sequence := atomic.AddUint64(&s.nextID, 1)
	return fmt.Sprintf("%d-%d", s.now().UTC().UnixNano(), sequence)
}

func (s *Service) CreateOrganization(in CreateOrganizationInput) (Organization, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return Organization{}, ErrInvalidInput
	}
	v := Organization{ID: s.id(), Name: name, CreatedAt: s.now().UTC()}
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
	if email == "" || name == "" {
		return User{}, ErrInvalidInput
	}
	for _, roleID := range in.RoleIDs {
		r, err := s.store.GetRole(roleID)
		if err != nil || r.OrganizationID != orgID {
			return User{}, ErrInvalidInput
		}
	}
	v := User{ID: s.id(), OrganizationID: orgID, Email: email, Name: name, RoleIDs: append([]string(nil), in.RoleIDs...), CreatedAt: s.now().UTC()}
	return v, s.store.CreateUser(v)
}
func (s *Service) ListUsers(orgID string) []User { return s.store.ListUsers(orgID) }
func (s *Service) ListRoles(orgID string) []Role { return s.store.ListRoles(orgID) }
