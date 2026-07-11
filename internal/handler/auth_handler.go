package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"docker-manager-backend/internal/auth"
	appmetrics "docker-manager-backend/internal/metrics"
	"docker-manager-backend/internal/models"
	"docker-manager-backend/internal/response"

	"golang.org/x/crypto/bcrypt"
)

const maxLoginBodyBytes = 8 * 1024

type AuthHandler struct {
	adminEmail        string
	adminPasswordHash []byte

	sessionManager *auth.SessionManager
}

func NewAuthHandler(
	adminEmail string,
	adminPasswordHash string,
	sessionManager *auth.SessionManager,
) *AuthHandler {
	return &AuthHandler{
		adminEmail: strings.ToLower(
			strings.TrimSpace(adminEmail),
		),
		adminPasswordHash: []byte(
			adminPasswordHash,
		),
		sessionManager: sessionManager,
	}
}

func (h *AuthHandler) Login(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Cache-Control", "no-store")

	request, ok := decodeLoginRequest(w, r)
	if !ok {
		return
	}

	email := strings.ToLower(
		strings.TrimSpace(request.Email),
	)

	emailMatches := email == h.adminEmail

	passwordMatches := bcrypt.CompareHashAndPassword(
		h.adminPasswordHash,
		[]byte(request.Password),
	) == nil

	if !emailMatches || !passwordMatches {
		appmetrics.LoginFailures.Inc()
		response.Error(
			w,
			http.StatusUnauthorized,
			"Invalid email or password",
		)
		return
	}

	// ลบ session เดิมก่อนสร้าง token ใหม่
	h.sessionManager.Revoke(r)

	user := auth.User{
		Email: h.adminEmail,
		Role:  "admin",
	}

	if err := h.sessionManager.Create(w, user); err != nil {
		log.Printf("create login session error: %v", err)

		response.Error(
			w,
			http.StatusInternalServerError,
			"Cannot create login session",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		models.AuthResponse{
			Authenticated: true,
			User: models.AuthUserResponse{
				Email: user.Email,
				Role:  user.Role,
			},
		},
	)
}

func (h *AuthHandler) Me(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Cache-Control", "no-store")

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		response.Error(
			w,
			http.StatusUnauthorized,
			"Authentication required",
		)
		return
	}

	response.JSON(
		w,
		http.StatusOK,
		models.AuthResponse{
			Authenticated: true,
			User: models.AuthUserResponse{
				Email: user.Email,
				Role:  user.Role,
			},
		},
	)
}

func (h *AuthHandler) Logout(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Cache-Control", "no-store")

	h.sessionManager.Destroy(w, r)

	response.JSON(
		w,
		http.StatusOK,
		models.MessageResponse{
			Success: true,
			Message: "Logged out successfully",
		},
	)
}

func decodeLoginRequest(
	w http.ResponseWriter,
	r *http.Request,
) (models.LoginRequest, bool) {
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxLoginBodyBytes,
	)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var request models.LoginRequest

	if err := decoder.Decode(&request); err != nil {
		response.Error(
			w,
			http.StatusBadRequest,
			"Invalid JSON request body",
		)
		return models.LoginRequest{}, false
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		response.Error(
			w,
			http.StatusBadRequest,
			"Request body must contain one JSON object",
		)
		return models.LoginRequest{}, false
	}

	request.Email = strings.TrimSpace(request.Email)

	if request.Email == "" || request.Password == "" {
		response.Error(
			w,
			http.StatusBadRequest,
			"Email and password are required",
		)
		return models.LoginRequest{}, false
	}

	if len(request.Password) > 72 {
		response.Error(
			w,
			http.StatusUnauthorized,
			"Invalid email or password",
		)
		return models.LoginRequest{}, false
	}

	return request, true
}
