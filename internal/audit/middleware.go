package audit

import (
	"log"
	"net/http"
	"strings"

	"github.com/nothing-4413/saas/internal/auth"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func Middleware(service *Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		rw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		status := rw.status
		if status == 0 {
			status = http.StatusOK
		}
		resourceType, resourceID := resourceFromPath(r.URL.Path)
		_, err := service.Record(claims.OrganizationID, claims.UserID, r.Method, resourceType, resourceID, map[string]interface{}{
			"path":       r.URL.Path,
			"status":     status,
			"request_id": w.Header().Get("X-Request-ID"),
		})
		if err != nil {
			log.Printf("audit record failed: %v", err)
		}
	})
}

func resourceFromPath(path string) (string, string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 {
		return "unknown", path
	}
	resourceID := strings.Join(parts[3:], "/")
	if resourceID == "" {
		resourceID = parts[1]
	}
	return parts[2], resourceID
}
