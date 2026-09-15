package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 1 && parts[0] == "organizations" && r.Method == http.MethodPost {
		h.createOrganization(w, r)
		return
	}
	if len(parts) >= 2 && parts[0] == "organizations" {
		orgID := parts[1]
		if len(parts) == 3 && parts[2] == "users" {
			if r.Method == http.MethodPost {
				h.createUser(w, r, orgID)
				return
			}
			if r.Method == http.MethodGet {
				h.listUsers(w, orgID)
				return
			}
		}
		if len(parts) == 3 && parts[2] == "roles" {
			if r.Method == http.MethodPost {
				h.createRole(w, r, orgID)
				return
			}
			if r.Method == http.MethodGet {
				h.listRoles(w, orgID)
				return
			}
		}
	}
	http.NotFound(w, r)
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
func (h *Handler) listUsers(w http.ResponseWriter, orgID string) {
	writeJSON(w, http.StatusOK, h.service.ListUsers(orgID))
}
func (h *Handler) listRoles(w http.ResponseWriter, orgID string) {
	writeJSON(w, http.StatusOK, h.service.ListRoles(orgID))
}
