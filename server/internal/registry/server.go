package registry

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Server handles registry-related HTTP requests.
type Server struct {
	store *Store
}

// NewServer creates a new registry server.
func NewServer(store *Store) *Server {
	return &Server{
		store: store,
	}
}

// HandleHandshake handles SDK registration and heartbeat requests.
// POST /api/registry/handshake
func (s *Server) HandleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var instance Instance
	if err := json.NewDecoder(r.Body).Decode(&instance); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if instance.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	// Basic enrichment
	if instance.IP == "" {
		instance.IP = r.RemoteAddr
		// Strip port if present
		if idx := strings.LastIndex(instance.IP, ":"); idx != -1 {
			instance.IP = instance.IP[:idx]
		}
	}

	s.store.RegisterOrUpdate(instance)


	// Mock Configuration Logic
	// Future: Fetch from DB or Config Store using instance.ServiceName
	resp := ConfigResponse{
		Level:      "INFO",
		SampleRate: 100,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleListInstances returns a list of registered instances.
// GET /api/registry/instances
func (s *Server) HandleListInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	instances := s.store.ListInstances()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}
