package api

import (
	"encoding/json"
	"net/http"
)

// Error envelope shared by every endpoint, per docs/ARCHITECTURE.md:
// {"error": {"code": "...", "message": "...", "fields": {...}}}
type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers are already sent; all we can do is log.
		s.log.Error("encoding response", "error", err)
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, code, message string) {
	s.respondJSON(w, status, errorEnvelope{Error: errorBody{Code: code, Message: message}})
}

func (s *Server) respondValidationError(w http.ResponseWriter, fields map[string]string) {
	s.respondJSON(w, http.StatusBadRequest, errorEnvelope{Error: errorBody{
		Code:    "validation_failed",
		Message: "one or more fields are invalid",
		Fields:  fields,
	}})
}
