package auth

import "net/http"

// RequirePermission protects an endpoint using tenant and user headers.
// The route handler remains responsible for validating the organization in its URL.
func RequirePermission(service *Service, permission Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgID := r.Header.Get("X-Organization-ID")
		userID := r.Header.Get("X-User-ID")
		if orgID == "" || userID == "" {
			http.Error(w, "missing authentication headers", http.StatusUnauthorized)
			return
		}
		allowed, err := service.HasPermission(orgID, userID, permission)
		if err != nil {
			if err == ErrNotFound {
				http.Error(w, "forbidden", http.StatusForbidden)
			} else {
				http.Error(w, "invalid authentication", http.StatusUnauthorized)
			}
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
