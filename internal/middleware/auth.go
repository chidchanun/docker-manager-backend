package middleware

import (
	"net/http"

	"docker-manager-backend/internal/auth"
	"docker-manager-backend/internal/response"
)

type AuthMiddleware struct {
	sessionManager *auth.SessionManager
}

func NewAuthMiddleware(
	sessionManager *auth.SessionManager,
) *AuthMiddleware {
	return &AuthMiddleware{
		sessionManager: sessionManager,
	}
}

func (m *AuthMiddleware) Require(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			user, ok := m.sessionManager.Current(r)
			if !ok {
				w.Header().Set(
					"Cache-Control",
					"no-store",
				)

				response.Error(
					w,
					http.StatusUnauthorized,
					"Authentication required",
				)
				return
			}

			ctx := auth.WithUser(
				r.Context(),
				user,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)
		},
	)
}