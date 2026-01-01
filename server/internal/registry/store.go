package registry

import (
	"context"
	"sync"
	"time"
)

// Instance represents a registered SDK instance.
type Instance struct {
	InstanceID   string `json:"instance_id"`
	ServiceName  string `json:"service_name"`
	Hostname     string `json:"hostname"`
	IP           string `json:"ip"`
	SdkVersion   string `json:"sdk_version"`
	Language     string `json:"language"`
	RegisteredAt int64  `json:"registered_at"`
	LastSeenAt   int64  `json:"last_seen_at"`
}

// ConfigResponse represents the dynamic configuration sent back to the SDK.
type ConfigResponse struct {
	Level      string `json:"level"`       // "INFO", "DEBUG"
	SampleRate int    `json:"sample_rate"` // 0-100
}

// Store handles the storage of SDK instances.
type Store struct {
	mu        sync.RWMutex
	instances map[string]*Instance
}

// NewStore creates a new registry store.
func NewStore() *Store {
	return &Store{
		instances: make(map[string]*Instance),
	}
}

// RegisterOrUpdate adds a new instance or updates an existing one.
func (s *Store) RegisterOrUpdate(instance Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// If it already exists, preserve RegisteredAt unless 0 (new)
	if existing, ok := s.instances[instance.InstanceID]; ok {
		instance.RegisteredAt = existing.RegisteredAt
	} else {
		if instance.RegisteredAt == 0 {
			instance.RegisteredAt = time.Now().Unix()
		}
	}

	instance.LastSeenAt = time.Now().Unix()
	s.instances[instance.InstanceID] = &instance
}

// GetInstance retrieves an instance by ID.
func (s *Store) GetInstance(instanceID string) (*Instance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	inst, ok := s.instances[instanceID]
	if !ok {
		return nil, false
	}
	// Return a copy to avoid race conditions if caller modifies it (though pointers are risky)
	// For this simple struct, dereferencing *inst copies the struct value.
	// But since we store *Instance, let's return a copy of the struct.
	val := *inst
	return &val, true
}

// ListInstances returns all registered instances.
func (s *Store) ListInstances() []Instance {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		list = append(list, *inst)
	}
	return list
}

// PruneStaleInstances removes instances that haven't been seen for a duration.
func (s *Store) PruneStaleInstances(timeout time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().Unix()
	count := 0
	timeoutSec := int64(timeout.Seconds())

	for id, inst := range s.instances {
		if now-inst.LastSeenAt > timeoutSec {
			delete(s.instances, id)
			count++
		}
	}
	return count
}

// StartCleanupLoop starts a background goroutine to prune stale instances.
func (s *Store) StartCleanupLoop(ctx context.Context, interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.PruneStaleInstances(timeout)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// KeepAlive updates the LastSeenAt timestamp for a given instance.
func (s *Store) KeepAlive(instanceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inst, ok := s.instances[instanceID]; ok {
		inst.LastSeenAt = time.Now().Unix()
	}
}
