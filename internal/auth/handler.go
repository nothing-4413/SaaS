package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/nothing-4413/saas/internal/platform/pagination"
)

type Handler struct {
	service       *Service
	tokenSecret   string
	tokenSecrets  []string
	resetNotifier func(org, email, token string) error
}

func NewHandler(service *Service, tokenSecret ...string) *Handler {
	secret := ""
	if len(tokenSecret) > 0 {
		secret = tokenSecret[0]
	}
	secrets := append([]string(nil), tokenSecret...)
	return &Handler{service: service, tokenSecret: secret, tokenSecrets: secrets}
}

// SetPasswordResetNotifier connects reset token delivery to an asynchronous provider.
func (h *Handler) SetPasswordResetNotifier(notifier func(org, email, token string) error) {
	h.resetNotifier = notifier
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 1 && parts[0] == "organizations" && r.Method == http.MethodPost {
		h.createOrganization(w, r)
		return
	}
	if len(parts) >= 2 && parts[0] == "organizations" {
		orgID := parts[1]
		if len(parts) == 4 && parts[2] == "sessions" && parts[3] == "password-reset" {
			if r.Method == http.MethodPost {
				h.requestPasswordReset(w, r, orgID)
				return
			}
			if r.Method == http.MethodPut {
				h.confirmPasswordReset(w, r, orgID)
				return
			}
		}
		if len(parts) == 3 && parts[2] == "sessions" && r.Method == http.MethodPost {
			h.login(w, r, orgID)
			return
		}
		if len(parts) == 3 && parts[2] == "sessions" && r.Method == http.MethodDelete {
			h.revokeSession(w, r)
			return
		}
		if len(parts) == 3 && parts[2] == "sessions" && r.Method == http.MethodPut {
			h.refreshSession(w, r, orgID)
			return
		}
		if len(parts) == 3 && parts[2] == "users" {
			if r.Method == http.MethodPost {
				h.createUser(w, r, orgID)
				return
			}
			if r.Method == http.MethodGet {
				h.listUsers(w, r, orgID)
				return
			}
		}
		if len(parts) == 4 && parts[2] == "users" && r.Method == http.MethodPut {
			h.updateUser(w, r, orgID, parts[3])
			return
		}
		if len(parts) == 3 && parts[2] == "roles" {
			if r.Method == http.MethodPost {
				h.createRole(w, r, orgID)
				return
			}
			if r.Method == http.MethodGet {
				h.listRoles(w, r, orgID)
				return
			}
		}
		if len(parts) == 4 && parts[2] == "roles" && r.Method == http.MethodPut {
			h.updateRole(w, r, orgID, parts[3])
			return
		}
	}
	http.NotFound(w, r)
}

func (h *Handler) requestPasswordReset(w http.ResponseWriter, r *http.Request, orgID string) {
	var in PasswordResetRequestInput
	if !decode(w, r, &in) {
		return
	}
	// Always return the same response to prevent account enumeration. Delivery
	// of the generated token is handled by the configured notification worker.
	if token, err := h.service.RequestPasswordReset(orgID, in.Email); err == nil && h.resetNotifier != nil {
		_ = h.resetNotifier(orgID, strings.ToLower(strings.TrimSpace(in.Email)), token)
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "if the account exists, reset instructions will be sent"})
}

func (h *Handler) confirmPasswordReset(w http.ResponseWriter, r *http.Request, orgID string) {
	var in PasswordResetConfirmInput
	if !decode(w, r, &in) {
		return
	}
	if err := h.service.ResetPassword(orgID, in.Token, in.Password); err != nil {
		if errors.Is(err, ErrInvalidResetToken) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) login(w http.ResponseWriter, r *http.Request, orgID string) {
	if h.tokenSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "token service unavailable")
		return
	}
	var in LoginInput
	if !decode(w, r, &in) {
		return
	}
	u, err := h.service.Authenticate(orgID, in.Email, in.Password)
	if err != nil {
		if errors.Is(err, ErrLoginLocked) {
			writeError(w, http.StatusTooManyRequests, "login temporarily locked")
			return
		}
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	session, err := h.service.CreateSession(u, 30*24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "session creation failed")
		return
	}
	token, err := IssueToken(h.tokenSecret, Claims{UserID: u.ID, OrganizationID: orgID, SessionID: session.ID, TokenType: "access", ExpiresAt: h.service.now().Add(15 * time.Minute).Unix()})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token creation failed")
		return
	}
	refresh, err := IssueToken(h.tokenSecret, Claims{UserID: u.ID, OrganizationID: orgID, SessionID: session.ID, TokenType: "refresh", ExpiresAt: session.ExpiresAt.Unix()})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "refresh token creation failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"access_token": token, "refresh_token": refresh, "token_type": "Bearer", "expires_in": 900})
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}
	claims, err := ParseTokenWithSecrets(h.tokenSecrets, parts[1])
	if err != nil || claims.SessionID == "" {
		writeError(w, http.StatusUnauthorized, "missing session")
		return
	}
	if _, err := h.service.GetActiveSession(claims.SessionID, claims.OrganizationID, claims.UserID); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	if err := h.service.RevokeSession(claims.SessionID); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) refreshSession(w http.ResponseWriter, r *http.Request, orgID string) {
	var in RefreshInput
	if !decode(w, r, &in) {
		return
	}
	claims, err := ParseTokenWithSecrets(h.tokenSecrets, in.RefreshToken)
	if err != nil || claims.TokenType != "refresh" || claims.OrganizationID != orgID {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	if _, err := h.service.GetActiveSession(claims.SessionID, claims.OrganizationID, claims.UserID); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid session")
		return
	}
	token, err := IssueToken(h.tokenSecret, Claims{UserID: claims.UserID, OrganizationID: claims.OrganizationID, SessionID: claims.SessionID, TokenType: "access", ExpiresAt: h.service.now().Add(15 * time.Minute).Unix()})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token creation failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"access_token": token, "token_type": "Bearer", "expires_in": 900})
}

func decode(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
func statusFor(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrConflict) {
		return http.StatusConflict
	}
	return http.StatusBadRequest
}
func (h *Handler) createOrganization(w http.ResponseWriter, r *http.Request) {
	var in CreateOrganizationInput
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.OwnerEmail) == "" || strings.TrimSpace(in.OwnerName) == "" || len(in.OwnerPassword) < 8 {
		writeError(w, http.StatusBadRequest, ErrInvalidInput.Error())
		return
	}
	v, err := h.service.CreateOrganization(in)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *Handler) createRole(w http.ResponseWriter, r *http.Request, orgID string) {
	var in CreateRoleInput
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.CreateRole(orgID, in)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request, orgID string) {
	var in CreateUserInput
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.CreateUser(orgID, in)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}
func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request, orgID string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.ListUsers(orgID)
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.Email), q) || strings.Contains(strings.ToLower(v.Name), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		less := items[i].CreatedAt.Before(items[j].CreatedAt)
		equal := items[i].CreatedAt.Equal(items[j].CreatedAt)
		if p.Sort == "name" {
			less = strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
			equal = strings.EqualFold(items[i].Name, items[j].Name)
		} else if p.Sort == "email" {
			less = strings.ToLower(items[i].Email) < strings.ToLower(items[j].Email)
			equal = strings.EqualFold(items[i].Email, items[j].Email)
		}
		if equal {
			less = items[i].ID < items[j].ID
			if items[i].ID == items[j].ID {
				return false
			}
		}
		if p.Desc {
			return !less
		}
		return less
	})
	writeJSON(w, http.StatusOK, pagination.Slice(items, p))
}
func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request, orgID string) {
	p, err := pagination.Parse(r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items := h.service.ListRoles(orgID)
	if p.Query != "" {
		q := strings.ToLower(p.Query)
		filtered := items[:0]
		for _, v := range items {
			if strings.Contains(strings.ToLower(v.Name), q) {
				filtered = append(filtered, v)
			}
		}
		items = filtered
	}
	sort.SliceStable(items, func(i, j int) bool {
		less := strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
		if strings.EqualFold(items[i].Name, items[j].Name) {
			less = items[i].ID < items[j].ID
			if items[i].ID == items[j].ID {
				return false
			}
		}
		if p.Desc {
			return !less
		}
		return less
	})
	writeJSON(w, http.StatusOK, pagination.Slice(items, p))
}
func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request, orgID, roleID string) {
	var in UpdateRoleInput
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.UpdateRole(orgID, roleID, in)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request, orgID, userID string) {
	var in UpdateUserInput
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.UpdateUser(orgID, userID, in)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, v)
}
