package api

import (
	"net/http"
	"time"
)

type healthResponse struct {
	Status string `json:"status"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}

// handleSlow pretends to work for 5 seconds, but gives up early if the
// client disconnects (their request context is cancelled).
func (s *Server) handleSlow(w http.ResponseWriter, r *http.Request) {
	select {
	case <-time.After(5 * time.Second):
		s.respondJSON(w, http.StatusOK, map[string]string{"status": "finished slow work"})
	case <-r.Context().Done():
		s.log.Info("slow request abandoned by client", "reason", r.Context().Err())
	}
}
