package response

import (
	"encoding/json"
	"net/http"
)

// SendJSON sends a JSON response with the given status code and data.
func SendJSON(w http.ResponseWriter, statusCode int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// SendError sends a JSON error response with the given status code and message.
func SendError(w http.ResponseWriter, statusCode int, message string) error {
	return SendJSON(w, statusCode, map[string]string{"error": message})
}