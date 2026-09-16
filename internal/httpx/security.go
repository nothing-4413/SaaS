package httpx

import (
	"net/http"
	"sync"
	"time"
)

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		next.ServeHTTP(w, r)
	})
}
func MaxBodyBytes(limit int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limit <= 0 {
			next.ServeHTTP(w, r)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	seen   map[string]bucket
}
type bucket struct {
	started time.Time
	count   int
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{limit: limit, window: window, seen: map[string]bucket{}}
}
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	b := l.seen[key]
	if b.started.IsZero() || now.Sub(b.started) >= l.window {
		l.seen[key] = bucket{started: now, count: 1}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	l.seen[key] = b
	return true
}
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Forwarded-For")
		if key == "" {
			key = r.RemoteAddr
		}
		if !l.Allow(key) {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
