package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/nothing-4413/saas/internal/platform/idgen"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrLoginLocked = errors.New("login temporarily locked")
var ErrInvalidResetToken = errors.New("invalid or expired password reset token")

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
		user := User{ID: s.id(), OrganizationID: v.ID, Email: email, Name: ownerName, PasswordHash: string(hash), RoleIDs: []string{role.ID}, Active: true, CreatedAt: v.CreatedAt}
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
	v := User{ID: s.id(), OrganizationID: orgID, Email: email, Name: name, PasswordHash: string(hash), RoleIDs: append([]string(nil), in.RoleIDs...), Active: true, CreatedAt: s.now().UTC()}
	return v, s.store.CreateUser(v)
}
func (s *Service) UpdateRole(orgID, roleID string, in UpdateRoleInput) (Role, error) {
	if strings.TrimSpace(orgID) == "" || strings.TrimSpace(roleID) == "" || strings.TrimSpace(in.Name) == "" {
		return Role{}, ErrInvalidInput
	}
	role, err := s.store.GetRole(roleID)
	if err != nil {
		return Role{}, err
	}
	if role.OrganizationID != orgID {
		return Role{}, ErrNotFound
	}
	role.Name = strings.TrimSpace(in.Name)
	role.Permissions = append([]Permission(nil), in.Permissions...)
	return role, s.store.UpdateRole(role)
}

func (s *Service) UpdateUser(orgID, userID string, in UpdateUserInput) (User, error) {
	if strings.TrimSpace(orgID) == "" || strings.TrimSpace(userID) == "" || strings.TrimSpace(in.Name) == "" {
		return User{}, ErrInvalidInput
	}
	user, err := s.store.GetUser(userID)
	if err != nil {
		return User{}, err
	}
	if user.OrganizationID != orgID {
		return User{}, ErrNotFound
	}
	for _, roleID := range in.RoleIDs {
		role, err := s.store.GetRole(roleID)
		if err != nil || role.OrganizationID != orgID {
			return User{}, ErrInvalidInput
		}
	}
	if in.Password != "" {
		if len(in.Password) < 8 {
			return User{}, ErrInvalidInput
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, err
		}
		user.PasswordHash = string(hash)
	}
	user.Name = strings.TrimSpace(in.Name)
	user.RoleIDs = append([]string(nil), in.RoleIDs...)
	if in.Active != nil {
		user.Active = *in.Active
	}
	return user, s.store.UpdateUser(user)
}
func (s *Service) Authenticate(org, email, password string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if org == "" || email == "" || password == "" {
		return User{}, ErrInvalidInput
	}
	locked, err := s.store.IsLoginLocked(org, email, s.now().UTC())
	if err != nil {
		return User{}, err
	}
	if locked {
		return User{}, ErrLoginLocked
	}
	u, err := s.store.FindUserByEmail(org, email)
	if err != nil {
		_, _ = s.store.RecordLoginFailure(org, email, s.now().UTC(), 5, 15*time.Minute)
		return User{}, ErrNotFound
	}
	if !u.Active || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		locked, _ := s.store.RecordLoginFailure(org, email, s.now().UTC(), 5, 15*time.Minute)
		if locked {
			return User{}, ErrLoginLocked
		}
		return User{}, ErrNotFound
	}
	if err := s.store.ClearLoginFailures(org, email); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *Service) RequestPasswordReset(org, email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if strings.TrimSpace(org) == "" || email == "" {
		return "", ErrInvalidInput
	}
	u, err := s.store.FindUserByEmail(org, email)
	if err != nil || !u.Active {
		return "", ErrNotFound
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	now := s.now().UTC()
	if err := s.store.CreatePasswordResetToken(org, u.ID, base64.RawURLEncoding.EncodeToString(hash[:]), now.Add(15*time.Minute), now); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Service) ResetPassword(org, token, password string) error {
	if strings.TrimSpace(org) == "" || strings.TrimSpace(token) == "" || len(password) < 8 {
		return ErrInvalidInput
	}
	hash := sha256.Sum256([]byte(token))
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.ResetPassword(org, base64.RawURLEncoding.EncodeToString(hash[:]), string(hashedPassword), s.now().UTC())
}

func (s *Service) CreateSession(user User, lifetime time.Duration) (Session, error) {
	if !user.Active || lifetime <= 0 {
		return Session{}, ErrInvalidInput
	}
	now := s.now().UTC()
	v := Session{ID: s.id(), OrganizationID: user.OrganizationID, UserID: user.ID, CreatedAt: now, ExpiresAt: now.Add(lifetime)}
	return v, s.store.CreateSession(v)
}
func (s *Service) GetActiveSession(id, orgID, userID string) (Session, error) {
	v, err := s.store.GetSession(id)
	if err != nil {
		return Session{}, err
	}
	if v.OrganizationID != orgID || v.UserID != userID || v.RevokedAt != nil || !v.ExpiresAt.After(s.now()) {
		return Session{}, ErrNotFound
	}
	u, err := s.store.GetUser(userID)
	if err != nil || !u.Active {
		return Session{}, ErrNotFound
	}
	return v, nil
}
func (s *Service) RevokeSession(id string) error { return s.store.RevokeSession(id, s.now().UTC()) }
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
