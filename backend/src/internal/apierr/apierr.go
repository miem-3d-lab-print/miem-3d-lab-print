package apierr

import (
	"encoding/json"
	"net/http"
)

// APIError describes a single error returned by the API.
type APIError struct {
	Code    string `json:"code" example:"INTERNAL_ERROR"`
	Message string `json:"message" example:"Внутренняя ошибка сервера"`
	Details any    `json:"details,omitempty" swaggertype:"object"`
}

// ErrorResponse is the standard error envelope returned on all 4xx/5xx responses.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

func Write(w http.ResponseWriter, status int, code, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: APIError{Code: code, Message: message, Details: details},
	})
}
