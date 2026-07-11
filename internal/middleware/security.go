package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"docker-manager-backend/internal/response"
)

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		if isUnsafeMethod(r.Method) && !sameOrigin(r) {
			response.Error(w, http.StatusForbidden, "Cross-site request rejected")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isUnsafeMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func sameOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && strings.EqualFold(parsed.Host, r.Host)
}

type loginAttempt struct {
	count   int
	resetAt time.Time
}
type LoginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
	limit    int
	window   time.Duration
}

func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{attempts: make(map[string]loginAttempt), limit: limit, window: window}
}

func (l *LoginRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		now := time.Now()
		l.mu.Lock()
		attempt := l.attempts[host]
		if now.After(attempt.resetAt) {
			attempt = loginAttempt{resetAt: now.Add(l.window)}
		}
		attempt.count++
		l.attempts[host] = attempt
		blocked := attempt.count > l.limit
		l.mu.Unlock()
		if blocked {
			w.Header().Set("Retry-After", "900")
			response.Error(w, http.StatusTooManyRequests, "Too many login attempts")
			return
		}
		next.ServeHTTP(w, r)
	})
}
