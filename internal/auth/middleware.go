package auth

import (
	"context"
	"net/http"
	"strings"
)

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

func RequireTokenPermission(service *Service, secret string, permission Permission, next http.Handler) http.Handler {
	return RequireTokenPermissions(service, []string{secret}, permission, next)
}

func RequireTokenPermissions(service *Service, secrets []string, permission Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		claims, err := ParseTokenWithSecrets(secrets, parts[1])
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		if claims.SessionID != "" {
			if _, err := service.GetActiveSession(claims.SessionID, claims.OrganizationID, claims.UserID); err != nil {
				http.Error(w, "session revoked", http.StatusUnauthorized)
				return
			}
		}
		path := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(path) >= 2 && path[0] == "organizations" && path[1] != claims.OrganizationID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		allowed, err := service.HasPermission(claims.OrganizationID, claims.UserID, permission)
		if err != nil || !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type claimsKey struct{}

func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	v, ok := ctx.Value(claimsKey{}).(Claims)
	return v, ok
}
