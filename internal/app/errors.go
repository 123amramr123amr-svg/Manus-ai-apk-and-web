package app

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	manusai "github.com/tigusigalpa/manus-ai-go/v2"
)

// errMissingAPIKey is returned when neither the request nor the server
// configuration provides a Manus API key.
var errMissingAPIKey = errors.New("no Manus API key available: add your key in Settings or set MANUS_API_KEY on the server")

// apiError is the JSON body returned for every failed request.
type apiError struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

// writeAPIError maps SDK and transport errors to HTTP status codes.
func writeAPIError(w http.ResponseWriter, err error) {
	status, code := http.StatusBadGateway, "manus_error"

	var authErr *manusai.AuthenticationError
	var validationErr *manusai.ValidationError
	var apiErr *manusai.Error

	switch {
	case errors.Is(err, errMissingAPIKey):
		status, code = http.StatusUnauthorized, "missing_api_key"
	case errors.As(err, &authErr):
		status, code = http.StatusUnauthorized, "manus_auth_error"
	case errors.As(err, &validationErr):
		status, code = http.StatusBadRequest, "manus_validation_error"
	case errors.As(err, &apiErr):
		code = "manus_api_error"
		if apiErr.StatusCode >= 400 && apiErr.StatusCode <= 599 {
			status = apiErr.StatusCode
		}
	}

	writeJSON(w, status, apiError{OK: false, Error: code, Message: err.Error()})
}

// decodeJSON reads a JSON request body (at most 2 MiB) into v.
func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
