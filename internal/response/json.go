package response

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func JSON(
	w http.ResponseWriter,
	statusCode int,
	data any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("encode JSON response error: %v", err)
	}
}

func Error(
	w http.ResponseWriter,
	statusCode int,
	message string,
) {
	JSON(
		w,
		statusCode,
		ErrorResponse{
			Error: message,
		},
	)
}
