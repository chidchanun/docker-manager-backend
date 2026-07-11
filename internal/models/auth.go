package models

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthUserResponse struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type AuthResponse struct {
	Authenticated bool             `json:"authenticated"`
	User          AuthUserResponse `json:"user"`
}

type MessageResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}