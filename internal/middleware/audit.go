package middleware

import (
	"docker-manager-backend/internal/audit"
	"docker-manager-backend/internal/auth"
	"net/http"
	"strings"
	"time"
)

type AuditMiddleware struct{ store *audit.Store }

func NewAuditMiddleware(store *audit.Store) *AuditMiddleware { return &AuditMiddleware{store: store} }

type auditWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (m *AuditMiddleware) Record(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || !strings.HasPrefix(r.URL.Path, "/api/containers/") {
			next.ServeHTTP(w, r)
			return
		}
		wrapped := &auditWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(wrapped, r)
		user, _ := auth.UserFromContext(r.Context())
		action := strings.TrimPrefix(r.URL.Path, "/api/containers/"+r.PathValue("id"))
		action = strings.Trim(action, "/")
		if action == "" && r.Method == http.MethodDelete {
			action = "remove"
		}
		_ = m.store.Add(r.Context(), audit.Entry{User: user.Email, Action: action, Container: r.PathValue("id"), IP: r.RemoteAddr, Status: wrapped.status, Success: wrapped.status < 400, Time: time.Now().UTC()})
	})
}
