package auth

import (
	"errors"
	"sync"
	"time"
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
	UpdateRole(Role) error
	CreateUser(User) error
	UpdateUser(User) error
	ListUsers(string) []User
	GetUser(string) (User, error)
	FindUserByEmail(string, string) (User, error)
	CreateSession(Session) error
	GetSession(string) (Session, error)
	RevokeSession(string, time.Time) error
	RecordLoginFailure(string, string, time.Time, int, time.Duration) (bool, error)
	ClearLoginFailures(string, string) error
	IsLoginLocked(string, string, time.Time) (bool, error)
	CreatePasswordResetToken(string, string, string, time.Time, time.Time) error
	ResetPassword(string, string, string, time.Time) error
}

type BootstrapStore interface {
	CreateOrganizationOwner(Organization, Role, User) error
}

type MemoryStore struct {
	mu             sync.RWMutex
	organizations  map[string]Organization
	roles          map[string]Role
	users          map[string]User
	sessions       map[string]Session
	loginFailures  map[string]loginFailure
	passwordResets map[string]passwordReset
}

type loginFailure struct {
	Count  int
	Locked time.Time
}

type passwordReset struct {
	OrganizationID string
	UserID         string
	ExpiresAt      time.Time
	UsedAt         *time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{organizations: make(map[string]Organization), roles: make(map[string]Role), users: make(map[string]User), sessions: make(map[string]Session), loginFailures: make(map[string]loginFailure), passwordResets: make(map[string]passwordReset)}
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
func (s *MemoryStore) CreateOrganizationOwner(org Organization, role Role, user User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.organizations[org.ID]; ok {
		return ErrConflict
	}
	for _, existing := range s.users {
		if existing.OrganizationID == org.ID && existing.Email == user.Email {
			return ErrConflict
		}
	}
	s.organizations[org.ID] = org
	s.roles[role.ID] = role
	s.users[user.ID] = user
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
func (s *MemoryStore) UpdateRole(v Role) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.roles[v.ID]
	if !ok || old.OrganizationID != v.OrganizationID {
		return ErrNotFound
	}
	for id, role := range s.roles {
		if id != v.ID && role.OrganizationID == v.OrganizationID && role.Name == v.Name {
			return ErrConflict
		}
	}
	s.roles[v.ID] = v
	return nil
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
func (s *MemoryStore) UpdateUser(v User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.users[v.ID]
	if !ok || old.OrganizationID != v.OrganizationID {
		return ErrNotFound
	}
	for id, user := range s.users {
		if id != v.ID && user.OrganizationID == v.OrganizationID && user.Email == v.Email {
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
func (s *MemoryStore) FindUserByEmail(orgID, email string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.users {
		if v.OrganizationID == orgID && v.Email == email {
			return v, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *MemoryStore) CreateSession(v Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[v.ID]; ok {
		return ErrConflict
	}
	s.sessions[v.ID] = v
	return nil
}
func (s *MemoryStore) GetSession(id string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return v, nil
}
func (s *MemoryStore) RevokeSession(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.sessions[id]
	if !ok {
		return ErrNotFound
	}
	v.RevokedAt = &at
	s.sessions[id] = v
	return nil
}

func loginKey(org, email string) string { return org + "\x00" + email }

func (s *MemoryStore) RecordLoginFailure(org, email string, now time.Time, maxAttempts int, lockout time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	k := loginKey(org, email)
	v := s.loginFailures[k]
	if v.Locked.After(now) {
		return true, nil
	}
	if !v.Locked.IsZero() {
		v = loginFailure{}
	}
	v.Count++
	if v.Count >= maxAttempts {
		v.Locked = now.Add(lockout)
	}
	s.loginFailures[k] = v
	return !v.Locked.IsZero() && v.Locked.After(now), nil
}

func (s *MemoryStore) ClearLoginFailures(org, email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.loginFailures, loginKey(org, email))
	return nil
}

func (s *MemoryStore) IsLoginLocked(org, email string, now time.Time) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v := s.loginFailures[loginKey(org, email)]
	return v.Locked.After(now), nil
}

func (s *MemoryStore) CreatePasswordResetToken(org, userID, tokenHash string, expiresAt, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[userID]
	if !ok || u.OrganizationID != org || !u.Active {
		return ErrNotFound
	}
	s.passwordResets[tokenHash] = passwordReset{OrganizationID: org, UserID: userID, ExpiresAt: expiresAt}
	return nil
}

func (s *MemoryStore) ResetPassword(org, tokenHash, passwordHash string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.passwordResets[tokenHash]
	if !ok || v.OrganizationID != org || v.UsedAt != nil || !v.ExpiresAt.After(now) {
		return ErrInvalidResetToken
	}
	u, ok := s.users[v.UserID]
	if !ok || u.OrganizationID != v.OrganizationID {
		return ErrInvalidResetToken
	}
	u.PasswordHash = passwordHash
	s.users[u.ID] = u
	for id, session := range s.sessions {
		if session.OrganizationID == v.OrganizationID && session.UserID == u.ID && session.RevokedAt == nil {
			revokedAt := now
			session.RevokedAt = &revokedAt
			s.sessions[id] = session
		}
	}
	v.UsedAt = &now
	s.passwordResets[tokenHash] = v
	return nil
}
