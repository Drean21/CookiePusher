package handler

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standardized JSON response structure.
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// RespondWithJSON is a helper to write a standard JSON response.
func RespondWithJSON(w http.ResponseWriter, code int, message string, payload interface{}) {
	response := APIResponse{
		Code:    code,
		Message: message,
		Data:    payload,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

// RespondWithError is a helper for error responses.
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, message, nil)
}
