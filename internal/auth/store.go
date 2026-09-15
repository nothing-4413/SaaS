package auth

import (
	"errors"
	"sync"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

type Store interface {
	CreateOrganization(Organization) error
	GetOrganization(string) (Organization, error)
	CreateRole(Role) error
	ListRoles(string) []Role
	GetRole(string) (Role, error)
	CreateUser(User) error
	ListUsers(string) []User
	GetUser(string) (User, error)
}

type MemoryStore struct {
	mu            sync.RWMutex
	organizations map[string]Organization
	roles         map[string]Role
	users         map[string]User
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{organizations: make(map[string]Organization), roles: make(map[string]Role), users: make(map[string]User)}
}

func (s *MemoryStore) CreateOrganization(v Organization) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.organizations[v.ID]; ok {
		return ErrConflict
	}
	s.organizations[v.ID] = v
	return nil
}
func (s *MemoryStore) GetOrganization(id string) (Organization, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.organizations[id]
	if !ok {
		return Organization{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) CreateRole(v Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.roles[v.ID]; ok {
		return ErrConflict
	}
	s.roles[v.ID] = v
	return nil
}
func (s *MemoryStore) ListRoles(orgID string) []Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Role, 0)
	for _, v := range s.roles {
		if v.OrganizationID == orgID {
			out = append(out, v)
		}
	}
	return out
}
func (s *MemoryStore) GetRole(id string) (Role, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.roles[id]
	if !ok {
		return Role{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) CreateUser(v User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.users {
		if u.Email == v.Email && u.OrganizationID == v.OrganizationID {
			return ErrConflict
		}
	}
	s.users[v.ID] = v
	return nil
}
func (s *MemoryStore) ListUsers(orgID string) []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]User, 0)
	for _, v := range s.users {
		if v.OrganizationID == orgID {
			out = append(out, v)
		}
	}
	return out
}

func (s *MemoryStore) GetUser(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return v, nil
}
