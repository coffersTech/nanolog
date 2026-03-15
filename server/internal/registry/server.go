package registry

import (
	"encoding/json"
	"github.com/coffersTech/nanolog/server/internal/models"
	"net/http"
	"strings"
	"time"
)

// PersistentStore defines the interface for persisting registry data.
type PersistentStore interface {
	AddOrUpdateDevice(device models.Instance) error
	DeleteDevice(id string) error
	GetDevices() []models.Instance
}

// Server handles registry-related HTTP requests.
type Server struct {
	store     *Store
	metaStore PersistentStore
}

// NewServer creates a new registry server.
func NewServer(store *Store, meta PersistentStore) *Server {
	return &Server{
		store:     store,
		metaStore: meta,
	}
}

// HandleHandshake handles SDK registration and heartbeat requests.
// POST /api/registry/handshake
func (s *Server) HandleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var instance models.Instance
	if err := json.NewDecoder(r.Body).Decode(&instance); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if instance.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	// Basic enrichment
	now := time.Now().Unix()
	instance.LastSeenAt = now
	if instance.RegisteredAt == 0 {
		instance.RegisteredAt = now
	}

	if instance.IP == "" {
		instance.IP = r.RemoteAddr
		// Strip port if present
		if idx := strings.LastIndex(instance.IP, ":"); idx != -1 {
			instance.IP = instance.IP[:idx]
		}
	}

	s.store.RegisterOrUpdate(instance)

	// Persist device if metaStore is available
	if s.metaStore != nil {
		go s.metaStore.AddOrUpdateDevice(instance)
	}

	// Mock Configuration Logic
	// Future: Fetch from DB or Config Store using instance.ServiceName
	resp := models.ConfigResponse{
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
	if instances == nil {
		instances = []models.Instance{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}

// HandleListDevices returns a list of all historically registered devices.
// GET /api/registry/devices
func (s *Server) HandleListDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metaStore == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	devices := s.metaStore.GetDevices()
	if devices == nil {
		devices = []models.Instance{}
	}

	// Dynamic enrichment: merge live heartbeat from memory store
	for i := range devices {
		if live, ok := s.store.GetInstance(devices[i].InstanceID); ok {
			devices[i].LastSeenAt = live.LastSeenAt
			// Also fill missing info if live instance has it
			if devices[i].Hostname == "" {
				devices[i].Hostname = live.Hostname
			}
			if devices[i].IP == "" {
				devices[i].IP = live.IP
			}
			if devices[i].ServiceName == "" {
				devices[i].ServiceName = live.ServiceName
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// HandleDeleteDevice removes a device from persistent metadata.
// DELETE /api/registry/devices/{id}
func (s *Server) HandleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.metaStore == nil {
		http.Error(w, "Metadata store not available", http.StatusServiceUnavailable)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	id := parts[len(parts)-1]

	if err := s.metaStore.DeleteDevice(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
